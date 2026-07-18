package domain

// HealthStatus represents the service health state.
type HealthStatus string

const (
	HealthStatusUp   HealthStatus = "up"
	HealthStatusDown HealthStatus = "down"
)

// Health is the response model for service health checks.
type Health struct {
	Status  HealthStatus `json:"status"`
	Service string       `json:"service"`
	Version string       `json:"version"`
}
