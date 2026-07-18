package infrastructure

import (
	"context"
	"testing"
)

func TestOpenDB_InMemory_RunsMigrations(t *testing.T) {
	db, err := OpenDB(context.Background(), DBConfig{Path: ":memory:", MaxOpenConns: 1})
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// The initial migration creates the items table; a schema_migrations row
	// records that 0001_init.sql was applied.
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if n != 1 {
		t.Errorf("applied migrations = %d, want 1", n)
	}

	// items table must exist and be writable.
	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO items (name) VALUES ('probe')`); err != nil {
		t.Fatalf("insert into items: %v", err)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db, err := OpenDB(context.Background(), DBConfig{Path: ":memory:", MaxOpenConns: 1})
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// Running Migrate again must be a no-op, not an error.
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if n != 1 {
		t.Errorf("applied migrations after re-run = %d, want 1", n)
	}
}
