package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(context.Background(), args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestUsageErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"bogus"},
		{"create"},
		{"create", "ftp://example.com"},
		{"create", "https://example.com", "--id", "has space"},
		{"get"},
		{"delete", "a", "b"},
		{"get", "abc", "--output", "yaml"},
	}
	for _, args := range cases {
		if code, _, _ := runCLI(t, args...); code != exitUsage {
			t.Errorf("%v: want exit %d, got %d", args, exitUsage, code)
		}
	}
}

func TestRegionFromEndpoint(t *testing.T) {
	if got := regionFromEndpoint("https://abc123.execute-api.ap-south-1.amazonaws.com/Prod"); got != "ap-south-1" {
		t.Errorf("got %q", got)
	}
	if got := regionFromEndpoint("https://go.example.com"); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestRandomID(t *testing.T) {
	id, err := randomID(7)
	if err != nil || len(id) != 7 {
		t.Fatalf("got %q, %v", id, err)
	}
	if validateID(id) != nil {
		t.Errorf("generated id %q fails validation", id)
	}
}

func TestEndToEndAgainstFakeAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/Prod/api/abc":
			_, _ = w.Write([]byte(`{"id":"abc","url":"https://example.com","shortUrl":"https://go.example.com/?go=abc"}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"","url":""}`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusUnprocessableEntity)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIDEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_CONFIG_FILE", "/dev/null")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/dev/null")
	conn := []string{"--endpoint", srv.URL + "/Prod", "--region", "ap-south-1"}

	code, out, _ := runCLI(t, append([]string{"get", "abc"}, conn...)...)
	if code != exitOK || !strings.Contains(out, "https://go.example.com/?go=abc") {
		t.Errorf("get: code %d, out %q", code, out)
	}
	if code, _, _ := runCLI(t, append([]string{"get", "nope"}, conn...)...); code != exitNotFound {
		t.Errorf("get missing: want %d, got %d", exitNotFound, code)
	}
	if code, _, _ := runCLI(t, append([]string{"create", "https://example.com", "--id", "abc"}, conn...)...); code != exitExists {
		t.Errorf("create existing: want %d, got %d", exitExists, code)
	}
	if code, _, _ := runCLI(t, append([]string{"delete", "nope"}, conn...)...); code != exitNotFound {
		t.Errorf("delete missing: want %d, got %d", exitNotFound, code)
	}
}
