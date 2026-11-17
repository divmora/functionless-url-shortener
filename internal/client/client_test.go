package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(srv.URL+"/", "ap-south-1", aws.NewCredentialsCache(
		credentials.NewStaticCredentialsProvider("AKIDEXAMPLE", "secret", "")))
}

func TestCreateSignsAndDecodes(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") || !strings.Contains(auth, "/ap-south-1/execute-api/aws4_request") {
			t.Errorf("missing or wrong SigV4 Authorization header: %q", auth)
		}
		if r.Header.Get("X-Amz-Content-Sha256") == "" && r.Header.Get("X-Amz-Date") == "" {
			t.Error("missing SigV4 date header")
		}
		body, _ := io.ReadAll(r.Body)
		var in CreateInput
		if err := json.Unmarshal(body, &in); err != nil {
			t.Fatal(err)
		}
		if in.ID != "abc" || in.URL != "https://example.com" || in.TTL != 60 {
			t.Errorf("unexpected body %s", body)
		}
		_, _ = w.Write([]byte(`{"id":"abc","url":"https://example.com","shortUrl":"https://go.example.com/?go=abc","owner":"me"}`))
	})

	link, err := c.Create(context.Background(), CreateInput{ID: "abc", URL: "https://example.com", TTL: 60})
	if err != nil {
		t.Fatal(err)
	}
	if link.ShortURL != "https://go.example.com/?go=abc" || link.Owner != "me" {
		t.Errorf("unexpected link %+v", link)
	}
}

func TestCreateOmitsZeroTTL(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "ttl") {
			t.Errorf("ttl should be omitted: %s", body)
		}
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	})
	if _, err := c.Create(context.Background(), CreateInput{ID: "abc", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
}

func TestGetEmptyItemIsNotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/missing" {
			t.Errorf("got path %s", r.URL.Path)
		}
		// Shape the GetItem mapping template returns for a missing item.
		_, _ = w.Write([]byte(`{"id":"", "url":"", "shortUrl":"/?go=", "timestamp":"", "owner":""}`))
	})
	if _, err := c.Get(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestStatusMapping(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnprocessableEntity, ErrExists},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusForbidden, ErrForbidden},
	}
	for _, tc := range cases {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"message":"x"}`))
		})
		_, err := c.Delete(context.Background(), "abc")
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: want %v, got %v", tc.status, tc.want, err)
		}
	}

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	var apiErr *APIError
	if _, err := c.Get(context.Background(), "abc"); !errors.As(err, &apiErr) || apiErr.Status != 500 {
		t.Errorf("want APIError 500, got %v", err)
	}
}

func TestDeleteUsesDeleteMethod(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/abc" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"abc","url":"https://example.com"}`))
	})
	link, err := c.Delete(context.Background(), "abc")
	if err != nil || link.ID != "abc" {
		t.Fatalf("got %+v, %v", link, err)
	}
}
