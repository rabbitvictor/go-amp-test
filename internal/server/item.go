package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
	"github.com/rabbitvictor/go-amp-test/internal/repository"
)

// ItemHandler exposes Item CRUD over HTTP.
type ItemHandler struct {
	repo *repository.ItemRepository
}

// NewItemHandler creates an ItemHandler backed by the given repository.
func NewItemHandler(repo *repository.ItemRepository) *ItemHandler {
	return &ItemHandler{repo: repo}
}

// Create handles POST /items.
func (h *ItemHandler) Create(c *echo.Context) error {
	var in domain.CreateItem
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}
	if in.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "name is required"})
	}

	item, err := h.repo.Create(c.Request().Context(), in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to create item"})
	}
	return c.JSON(http.StatusCreated, item)
}

// List handles GET /items.
func (h *ItemHandler) List(c *echo.Context) error {
	items, err := h.repo.List(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to list items"})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Get handles GET /items/:id.
func (h *ItemHandler) Get(c *echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid id"})
	}

	item, err := h.repo.Get(c.Request().Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "item not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to get item"})
	}
	return c.JSON(http.StatusOK, item)
}
