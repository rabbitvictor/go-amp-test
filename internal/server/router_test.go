package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Without a DB, only the health route is registered; item routes must 404.
func TestNew_NoDB_OnlyHealthRoutes(t *testing.T) {
	e := New(Config{Service: "go-amp-test", Version: "test"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("POST /items without DB = %d, want 404", rec.Code)
	}
}
