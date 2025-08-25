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

	// EnableTLS specifies whether TLS should be enabled for the Gateway server
	EnableTLS bool

	// DefaultCertDomain is the domain to use for fallback TLS certificate
	DefaultCertDomain string

	// MainDomain is the primary domain for the gateway (e.g., gateway.unkey.com)
	// Internal endpoints like /_internal/liveness are only accessible on this domain
	MainDomain string

	// --- ClickHouse configuration ---

	// ClickhouseURL is the ClickHouse database connection string
	ClickhouseURL string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations (partition_001)
	DatabasePrimary string

	// DatabaseReadonlyReplica is an optional read-replica database connection string for read operations (partition_001)
	DatabaseReadonlyReplica string

	// --- Key Service Database configuration ---

	// KeysDatabasePrimary is the primary database connection string for the keys service
	KeysDatabasePrimary string

	// KeysDatabaseReadonlyReplica is an optional read-replica database connection string for the keys service
	KeysDatabaseReadonlyReplica string

	// RedisURL is the Redis connection string for the keys service
	RedisURL string

	// -- OpenTelemetry configuration ---
	// OtelEnabled specifies whether OpenTelemetry is enabled for the Gateway server
	OtelEnabled bool

	// OtelTraceSamplingRate specifies the sampling rate for OpenTelemetry traces
	OtelTraceSamplingRate float64

	// PrometheusPort specifies the port for Prometheus metrics
	PrometheusPort int
}

func (c Config) Validate() error {
	return nil
}
