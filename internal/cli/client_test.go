package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, 5*time.Second)
}

func TestClient_Health(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/health" {
			t.Errorf("request = %s %s, want GET /health", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(domain.Health{
			Status:  domain.HealthStatusUp,
			Service: "svc",
			Version: "v1",
		})
	})

	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if h.Status != domain.HealthStatusUp || h.Service != "svc" {
		t.Errorf("health = %+v", h)
	}
}

func TestClient_CreateGetList(t *testing.T) {
	var lastID int64
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /items":
			var in domain.CreateItem
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			lastID++
			_ = json.NewEncoder(w).Encode(domain.Item{ID: lastID, Name: in.Name})
		case "GET /items":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []domain.Item{{ID: 1, Name: "alpha"}, {ID: 2, Name: "beta"}},
			})
		case "GET /items/2":
			_ = json.NewEncoder(w).Encode(domain.Item{ID: 2, Name: "beta"})
		default:
			http.NotFound(w, r)
		}
	})

	created, err := c.CreateItem(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if created.ID != 1 || created.Name != "alpha" {
		t.Errorf("created = %+v", created)
	}

	got, err := c.GetItem(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if got.ID != 2 || got.Name != "beta" {
		t.Errorf("got = %+v", got)
	}

	items, err := c.ListItems(context.Background())
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}

func TestClient_APIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
	})

	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Status != http.StatusInternalServerError {
		t.Errorf("APIError.Status = %d, want %d", apiErr.Status, http.StatusInternalServerError)
	}
	if apiErr.Body == "" {
		t.Error("APIError.Body should not be empty")
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("http://example.com/", time.Second)
	if c.BaseURL != "http://example.com" {
		t.Errorf("BaseURL = %q, want no trailing slash", c.BaseURL)
	}
}
