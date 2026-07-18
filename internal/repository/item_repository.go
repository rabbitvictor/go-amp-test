package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

// ErrNotFound is returned when no row matches the lookup.
var ErrNotFound = errors.New("not found")

// ItemRepository persists Item rows in SQLite via database/sql.
type ItemRepository struct {
	db *sql.DB
}

// NewItemRepository creates an ItemRepository backed by the given *sql.DB.
func NewItemRepository(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// itemColumns returns the canonical, ordered column list for the items table.
// It is the single source of truth for SELECT column lists in this repository
// and must stay in sync with the scan order in Get and List.
func itemColumns() string {
	return "id, name, created_at"
}

// Create inserts a new item and returns the freshly persisted row.
func (r *ItemRepository) Create(ctx context.Context, in domain.CreateItem) (*domain.Item, error) {
	res, err := r.db.ExecContext(ctx, "INSERT INTO items (name) VALUES (?)", in.Name)
	if err != nil {
		return nil, fmt.Errorf("insert item: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return r.Get(ctx, id)
}

// Get returns a single item by id, or ErrNotFound if it does not exist.
func (r *ItemRepository) Get(ctx context.Context, id int64) (*domain.Item, error) {
	var it domain.Item
	err := r.db.QueryRowContext(ctx,
		"SELECT "+itemColumns()+" FROM items WHERE id = ?", id,
	).Scan(&it.ID, &it.Name, &it.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get item %d: %w", id, err)
	}
	return &it, nil
}

// List returns all items, newest first.
func (r *ItemRepository) List(ctx context.Context) ([]domain.Item, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+itemColumns()+" FROM items ORDER BY id DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var it domain.Item
		if err := rows.Scan(&it.ID, &it.Name, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}
