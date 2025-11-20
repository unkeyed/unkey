package gateway

// Config holds configuration for the Gateway server
type Config struct {
	// GatewayID is the unique identifier for this gateway instance
	GatewayID string

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
	// Add validation as needed
	return nil
}
