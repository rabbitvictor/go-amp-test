package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

// FuzzItemRepository_Create round-trips arbitrary name strings (null bytes,
// invalid UTF-8, unicode) through Create + Get against in-memory SQLite. It
// must never panic. Any value accepted by Create must be stored and read back
// byte-for-byte; a rejection must come back as a clean error.
func FuzzItemRepository_Create(f *testing.F) {
	cases := []string{
		"alpha", "", " ", "null\x00byte",
		"\xff\xfe\xfd", "unicode \u00e9\u4e16\u754c",
		"\n\t\r", strings.Repeat("a", 10000),
		`{"name":"nested"}`, `name with "quotes"`,
	}
	for _, tc := range cases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, name string) {
		repo := NewItemRepository(newMemoryDB(t))
		ctx := context.Background()

		created, err := repo.Create(ctx, domain.CreateItem{Name: name})
		if err != nil {
			// A clean rejection is fine; a panic is not (the test framework
			// turns panics into failures automatically).
			return
		}
		got, err := repo.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Get after Create failed: %v", err)
		}
		if got.Name != name {
			t.Fatalf("name round-trip mismatch: sent %q, got %q", name, got.Name)
		}
	})
}
