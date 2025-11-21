package gateway

import "github.com/unkeyed/unkey/go/pkg/assert"

// Config holds configuration for the Gateway server
type Config struct {
	// GatewayID is the unique identifier for this gateway instance
	GatewayID string

	// WorkspaceID identifies which workspace this gateway serves
	WorkspaceID string

	// EnvironmentID identifies which environment this gateway serves
	// A single environment may have multiple deployments, and this gateway
	// handles all of them based on the deployment ID passed in each request
	EnvironmentID string

	// Platform identifies the cloud platform (e.g., aws, gcp, hetzner)
	Platform string

	// Region identifies the geographic region
	Region string

	// HttpPort is the port for the HTTP proxy server
	HttpPort int

	// Database configuration
	DatabasePrimary         string
	DatabaseReadonlyReplica string

	// OpenTelemetry configuration
	OtelEnabled           bool
	OtelTraceSamplingRate float64
	PrometheusPort        int
}

func (c Config) Validate() error {
	return assert.All(
		assert.NotEmpty(c.WorkspaceID, "workspace ID is required"),
		assert.NotEmpty(c.EnvironmentID, "environment ID is required"),
	)
}
