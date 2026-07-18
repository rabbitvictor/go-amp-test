package server

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

// HealthHandler holds dependencies for health-related HTTP handlers.
type HealthHandler struct {
	service string
	version string
}

// NewHealthHandler creates a HealthHandler for the given service identity.
func NewHealthHandler(service, version string) *HealthHandler {
	return &HealthHandler{service: service, version: version}
}

// Health returns the current service health.
func (h *HealthHandler) Health(c *echo.Context) error {
	return c.JSON(http.StatusOK, domain.Health{
		Status:  domain.HealthStatusUp,
		Service: h.service,
		Version: h.version,
	})
}
