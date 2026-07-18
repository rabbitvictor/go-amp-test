package server

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// Config holds settings used to build the Echo server.
type Config struct {
	Service string
	Version string
}

// New builds and configures an Echo instance with registered routes and
// middleware. It does not start the server.
func New(cfg Config) *echo.Echo {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	health := NewHealthHandler(cfg.Service, cfg.Version)

	e.GET("/health", health.Health)

	return e
}
