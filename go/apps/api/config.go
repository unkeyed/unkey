package api

import (
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/tls"
)

type Config struct {

	// InstanceID is the unique identifier for this instance of the API server
	InstanceID string
	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the API server to listen on (default: 7070)
	HttpPort int

	// Region identifies the geographic region where this node is deployed
	Region string

	// RedisUrl is the Redis database connection string
	RedisUrl string

	// Enable TestMode
	TestMode bool

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
	Clock          clock.Clock

	// --- TLS configuration ---

	// TLSConfig provides HTTPS support when set
	TLSConfig *tls.Config
}

func (c Config) Validate() error {
	// TLS configuration is validated when it's created from files
	// Other validations may be added here in the future
	return nil
}
