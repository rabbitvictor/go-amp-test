package domain

import "time"

// Item is a single persisted item.
type Item struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateItem is the input for creating an item.
type CreateItem struct {
	Name string `json:"name"`
}
