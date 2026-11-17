// Package client is a SigV4-signing HTTP client for the URL shortener management API.
package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// signingService is the SigV4 service name for API Gateway invocations.
const signingService = "execute-api"

// Sentinel errors returned by the client so callers can map them to exit codes and messages.
var (
	ErrNotFound  = errors.New("link not found")
	ErrExists    = errors.New("link ID already exists")
	ErrForbidden = errors.New("access denied")
)

// Link is a short link as returned by the management API.
type Link struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	ShortURL  string `json:"shortUrl"`
	Timestamp string `json:"timestamp,omitempty"`
	Owner     string `json:"owner,omitempty"`
}

// CreateInput is the request body for creating a link.
type CreateInput struct {
	ID  string `json:"id"`
	URL string `json:"url"`
	TTL int64  `json:"ttl,omitempty"`
}

// HTTPSigner signs HTTP requests; satisfied by *v4.Signer.
type HTTPSigner interface {
	SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string,
		service string, region string, signingTime time.Time, optFns ...func(*v4.SignerOptions)) error
}

// Client calls the management API at Endpoint (the execute-api stage URL, not CloudFront).
type Client struct {
	Endpoint    string
	Region      string
	Credentials aws.CredentialsProvider
	Signer      HTTPSigner
	HTTP        *http.Client
	Now         func() time.Time
}

// New returns a Client with a default SigV4 signer and HTTP client.
func New(endpoint, region string, creds aws.CredentialsProvider) *Client {
	return &Client{
		Endpoint:    strings.TrimRight(endpoint, "/"),
		Region:      region,
		Credentials: creds,
		Signer:      v4.NewSigner(),
		HTTP:        &http.Client{Timeout: 30 * time.Second},
		Now:         time.Now,
	}
}

// Create creates a new short link. Returns ErrExists if the ID is taken.
func (c *Client) Create(ctx context.Context, in CreateInput) (*Link, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var link Link
	if err := c.do(ctx, http.MethodPost, "/api", body, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

// Get fetches a link by ID. Returns ErrNotFound if it doesn't exist.
func (c *Client) Get(ctx context.Context, id string) (*Link, error) {
	var link Link
	if err := c.do(ctx, http.MethodGet, "/api/"+url.PathEscape(id), nil, &link); err != nil {
		return nil, err
	}
	// The GetItem mapping template returns 200 with empty fields when the item is missing.
	if link.ID == "" {
		return nil, ErrNotFound
	}
	return &link, nil
}

// Delete removes a link by ID. Returns ErrNotFound if it doesn't exist.
func (c *Client) Delete(ctx context.Context, id string) (*Link, error) {
	var link Link
	if err := c.do(ctx, http.MethodDelete, "/api/"+url.PathEscape(id), nil, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

// APIError is returned for unexpected non-2xx responses.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API returned %d: %s", e.Status, strings.TrimSpace(e.Body))
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.Endpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	creds, err := c.Credentials.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS credentials: %w", err)
	}
	sum := sha256.Sum256(body)
	if err := c.Signer.SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), signingService, c.Region, c.Now()); err != nil {
		return fmt.Errorf("signing request: %w", err)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	switch {
	case resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrForbidden, strings.TrimSpace(string(data)))
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusConflict:
		return ErrExists
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	if out != nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}
