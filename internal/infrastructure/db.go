package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// DBConfig controls how the SQLite database is opened.
type DBConfig struct {
	// Path is the filesystem path to the SQLite file. Use ":memory:" for an
	// ephemeral, in-process database (useful for tests).
	Path string
	// MaxOpenConns caps the connection pool. SQLite serializes writes against
	// a single file; 1 avoids SQLITE_BUSY. Raise for read-heavy WAL workloads.
	MaxOpenConns int
	// DSN is the full driver-specific data source name. If empty, it is built
	// from Path with production-safe defaults.
	DSN string
}

// OpenDB opens a SQLite database with sensible production defaults (WAL mode,
// busy timeout, foreign keys), runs all pending migrations, and returns a
// ready-to-use *sql.DB.
//
// If cfg.DSN is empty, a default DSN is built from cfg.Path. SetMaxOpenConns
// defaults to 1 when MaxOpenConns is non-positive.
func OpenDB(ctx context.Context, cfg DBConfig) (*sql.DB, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = fmt.Sprintf(
			"file:%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=on",
			cfg.Path,
		)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 1
	}
	db.SetMaxOpenConns(maxOpen)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := Migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	slog.Info("database ready", "path", cfg.Path, "max_open_conns", maxOpen)
	return db, nil
}
