package api

import (
	"net"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/tls"
)

const (
	// DefaultCacheInvalidationTopic is the default Kafka topic name for cache invalidation events
	DefaultCacheInvalidationTopic = "cache-invalidations"
)

type S3Config struct {
	URL             string
	Bucket          string
	AccessKeyID     string
	AccessKeySecret string
}

type Config struct {
	// InstanceID is the unique identifier for this instance of the API server
	InstanceID string

	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the API server to listen on (default: 7070)
	// Used in production deployments. Ignored if Listener is provided.
	HttpPort int

	// Listener defines a pre-created network listener for the HTTP server
	// If provided, the server will use this listener instead of creating one from HttpPort
	// This is intended for testing scenarios where ephemeral ports are needed to avoid conflicts
	Listener net.Listener

	// Region identifies the geographic region where this node is deployed
	Region string

	// RedisUrl is the Redis database connection string
	RedisUrl string

	// Enable TestMode
	TestMode bool

	// --- ClickHouse configuration ---

	// ClickhouseURL is the ClickHouse database connection string
	ClickhouseURL string

	// ClickhouseAnalyticsURL is the base URL for workspace-specific analytics connections
	// Workspace credentials are injected programmatically at connection time
	// Examples: "http://clickhouse:8123/default", "clickhouse://clickhouse:9000/default"
	ClickhouseAnalyticsURL string

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

	// Vault Configuration
	VaultMasterKeys []string
	VaultS3         *S3Config

	// --- Kafka configuration ---

	// KafkaBrokers is the list of Kafka broker addresses
	KafkaBrokers []string

	// CacheInvalidationTopic is the Kafka topic name for cache invalidation events
	// If empty, defaults to DefaultCacheInvalidationTopic
	CacheInvalidationTopic string

	// --- ClickHouse proxy configuration ---

	// ChproxyToken is the authentication token for ClickHouse proxy endpoints
	ChproxyToken string

	// --- pprof configuration ---

	// PprofEnabled controls whether the pprof profiling endpoints are available
	PprofEnabled bool

	// PprofUsername is the username for pprof Basic Auth
	// If empty along with PprofPassword, pprof endpoints will be accessible without authentication
	PprofUsername string

	// PprofPassword is the password for pprof Basic Auth
	// If empty along with PprofUsername, pprof endpoints will be accessible without authentication
	PprofPassword string

	// MaxRequestBodySize sets the maximum allowed request body size in bytes.
	// If 0 or negative, no limit is enforced. Default is 0 (no limit).
	// This helps prevent DoS attacks from excessively large request bodies.
	MaxRequestBodySize int64

	// DebugCacheHeaders enables cache debug headers (X-Unkey-Debug-Cache) in responses.
	// When enabled, cache operations add headers showing hit/miss status and latency.
	// Should typically only be enabled in development or specific production debugging.
	DebugCacheHeaders bool
}

func (c Config) Validate() error {
	// TLS configuration is validated when it's created from files
	// Other validations may be added here in the future
	if c.VaultS3 != nil {
		err := assert.All(
			assert.NotEmpty(c.VaultS3.URL, "vault s3 url is empty"),
			assert.NotEmpty(c.VaultS3.Bucket, "vault s3 bucket is empty"),
			assert.NotEmpty(c.VaultS3.AccessKeyID, "vault s3 access key id is empty"),
			assert.NotEmpty(c.VaultS3.AccessKeySecret, "vault s3 secret access key is empty"),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
