package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

// Client is a thin HTTP client for the go-amp-test web API.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient builds a Client for the given base URL and request timeout.
// BaseURL must not have a trailing slash.
func NewClient(baseURL string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: timeout},
	}
}

// APIError is returned when the server responds with a non-2xx status. Its
// Error string includes the status and the server's error body, so it can be
// surfaced to the user.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("server returned %d", e.Status)
	}
	return fmt.Sprintf("server returned %d: %s", e.Status, e.Body)
}

// Health calls GET /health.
func (c *Client) Health(ctx context.Context) (domain.Health, error) {
	var h domain.Health
	if err := c.do(ctx, http.MethodGet, "/health", nil, &h); err != nil {
		return h, err
	}
	return h, nil
}

// ListItems calls GET /items.
func (c *Client) ListItems(ctx context.Context) ([]domain.Item, error) {
	var out struct {
		Items []domain.Item `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, "/items", nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// GetItem calls GET /items/:id.
func (c *Client) GetItem(ctx context.Context, id int64) (domain.Item, error) {
	var it domain.Item
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/items/%d", id), nil, &it); err != nil {
		return it, err
	}
	return it, nil
}

// CreateItem calls POST /items with the given name.
func (c *Client) CreateItem(ctx context.Context, name string) (domain.Item, error) {
	body, err := json.Marshal(domain.CreateItem{Name: name})
	if err != nil {
		return domain.Item{}, fmt.Errorf("marshal request: %w", err)
	}
	var it domain.Item
	if err := c.do(ctx, http.MethodPost, "/items", body, &it); err != nil {
		return it, err
	}
	return it, nil
}

// do performs an HTTP request against the API, decodes a JSON success body
// into out, and returns an *APIError on non-2xx responses.
func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	u, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		return fmt.Errorf("build url: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(raw))}
	}

	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
