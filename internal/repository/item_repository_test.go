package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

func TestItemRepository_CreateAndGet(t *testing.T) {
	repo := NewItemRepository(newMemoryDB(t))
	ctx := context.Background()

	created, err := repo.Create(ctx, domain.CreateItem{Name: "alpha"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created.ID should be non-zero")
	}
	if created.Name != "alpha" {
		t.Errorf("created.Name = %q, want %q", created.Name, "alpha")
	}
	if created.CreatedAt.IsZero() {
		t.Error("created.CreatedAt should be populated by the database")
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Errorf("Get returned %+v, want %+v", got, created)
	}
}

func TestItemRepository_Get_NotFound(t *testing.T) {
	repo := NewItemRepository(newMemoryDB(t))
	_, err := repo.Get(context.Background(), 9999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing row returned %v, want ErrNotFound", err)
	}
}

func TestItemRepository_List_NewestFirst(t *testing.T) {
	repo := NewItemRepository(newMemoryDB(t))
	ctx := context.Background()

	for _, name := range []string{"one", "two", "three"} {
		if _, err := repo.Create(ctx, domain.CreateItem{Name: name}); err != nil {
			t.Fatalf("Create %q: %v", name, err)
		}
	}

	items, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	// ORDER BY id DESC => last inserted comes first.
	if items[0].Name != "three" || items[2].Name != "one" {
		t.Errorf("List order = %s, %s, %s; want three, two, one",
			items[0].Name, items[1].Name, items[2].Name)
	}
}

func TestItemRepository_List_Empty(t *testing.T) {
	repo := NewItemRepository(newMemoryDB(t))
	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if items != nil {
		t.Errorf("List on empty table returned %v, want nil", items)
	}
}
