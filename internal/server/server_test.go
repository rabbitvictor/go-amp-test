package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rabbitvictor/go-amp-test/internal/infrastructure"
)

// newTestEcho builds an Echo server backed by a migrated in-memory SQLite DB.
func newTestEcho(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	db, err := infrastructure.OpenDB(context.Background(), infrastructure.DBConfig{
		Path:         ":memory:",
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	e := New(Config{Service: "go-amp-test", Version: "test", DB: db})
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv, db
}

// do dispatches a request through the Echo engine behind srv and returns the
// recorded response. body is sent as application/json when non-empty.
func do(t *testing.T, srv *httptest.Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	srv.Config.Handler.ServeHTTP(rec, req)
	return rec
}
