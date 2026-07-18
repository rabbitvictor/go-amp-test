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
}

// OpenDB opens a SQLite database with sensible production defaults (WAL mode,
// busy timeout, foreign keys), runs all pending migrations, and returns a
// ready-to-use *sql.DB.
//
// SetMaxOpenConns is set to 1 because SQLite serializes writes against a
// single file; a single connection avoids SQLITE_BUSY errors under load.
// Raise this if you have a read-heavy workload and rely on WAL concurrent
// readers.
func OpenDB(ctx context.Context, cfg DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=on",
		cfg.Path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", cfg.Path, err)
	}
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := Migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	slog.Info("database ready", "path", cfg.Path)
	return db, nil
}
