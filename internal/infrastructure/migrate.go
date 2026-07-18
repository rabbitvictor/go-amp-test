package infrastructure

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrationsTable is the bookkeeping table tracking applied migrations.
const migrationsTable = "schema_migrations"

// Migrate applies all pending *.sql migrations embedded under migrations/.
// Each migration runs in its own transaction and is recorded by filename.
// Migrations are forward-only and applied in lexical (filename) order.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
			version    TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`, migrationsTable,
	)); err != nil {
		return fmt.Errorf("create %s: %w", migrationsTable, err)
	}

	applied, err := appliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	pending, err := pendingMigrations(applied)
	if err != nil {
		return err
	}

	for _, name := range pending {
		if err := applyMigration(ctx, db, name); err != nil {
			return err
		}
		slog.Info("migration applied", "version", name)
	}

	if len(pending) == 0 {
		slog.Info("database up to date", "applied", len(applied))
	}
	return nil
}

func appliedMigrations(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT version FROM %s", migrationsTable))
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", migrationsTable, err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan %s: %w", migrationsTable, err)
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func pendingMigrations(applied map[string]bool) ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		if applied[e.Name()] {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

func applyMigration(ctx context.Context, db *sql.DB, name string) error {
	content, err := migrationsFS.ReadFile("migrations/" + name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("exec %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s (version) VALUES (?)", migrationsTable), name,
	); err != nil {
		return fmt.Errorf("record %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", name, err)
	}
	return nil
}
