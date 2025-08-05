package gw

type Config struct {
	// GatewayID is the unique identifier for this instance of the Gateway server
	GatewayID string

	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the Gateway server to listen on (default: 6060)
	HttpPort int

	// Region identifies the geographic region where this node is deployed
	Region string

	// --- ClickHouse configuration ---

	// ClickhouseURL is the ClickHouse database connection string
	ClickhouseURL string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations
	DatabasePrimary string

	// DatabaseReadonlyReplica is an optional read-replica database connection string for read operations
	DatabaseReadonlyReplica string

	// --- OpenTelemetry configuration ---

	// Enable sending otel data to the  collector endpoint for metrics, traces, and logs
	OtelEnabled           bool
	OtelTraceSamplingRate float64

	PrometheusPort int
}

func (c Config) Validate() error {
	return nil
}
