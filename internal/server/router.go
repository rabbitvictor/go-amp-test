package server

import (
	"database/sql"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/rabbitvictor/go-amp-test/internal/repository"
)

// Config holds settings used to build the Echo server.
type Config struct {
	Service string
	Version string
	// DB is the SQLite database. If nil, only health routes are registered.
	DB *sql.DB
}

// New builds and configures an Echo instance with registered routes and
// middleware. It does not start the server.
func New(cfg Config) *echo.Echo {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	health := NewHealthHandler(cfg.Service, cfg.Version)

	e.GET("/health", health.Health)

	if cfg.DB != nil {
		items := NewItemHandler(repository.NewItemRepository(cfg.DB))
		e.POST("/items", items.Create)
		e.GET("/items", items.List)
		e.GET("/items/:id", items.Get)
	}

	return e
}
