package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

func TestHealthHandler_Health(t *testing.T) {
	srv, _ := newTestEcho(t)

	rec := do(t, srv, http.MethodGet, "/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var h domain.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &h); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if h.Status != domain.HealthStatusUp {
		t.Errorf("status = %q, want %q", h.Status, domain.HealthStatusUp)
	}
	if h.Service != "go-amp-test" || h.Version != "test" {
		t.Errorf("health = %+v, want service=go-amp-test version=test", h)
	}
}
