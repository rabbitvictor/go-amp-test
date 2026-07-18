package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/rabbitvictor/go-amp-test/internal/infrastructure"
)

// newMemoryDB returns a migrated in-memory SQLite *sql.DB for repository
// tests. The single-connection pool keeps the ephemeral schema alive for the
// lifetime of the *sql.DB.
func newMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := infrastructure.OpenDB(context.Background(), infrastructure.DBConfig{
		Path:         ":memory:",
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
