package api

import (
	"net"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/tls"
)

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
	VaultURL   string
	VaultToken string

	// --- Gossip cluster configuration ---

	// GossipEnabled controls whether gossip-based cache invalidation is active
	GossipEnabled bool

	// GossipBindAddr is the address to bind gossip listeners on (default "0.0.0.0")
	GossipBindAddr string

	// GossipLANPort is the LAN memberlist port (default 7946)
	GossipLANPort int

	// GossipWANPort is the WAN memberlist port for gateways (default 7947)
	GossipWANPort int

	// GossipLANSeeds are addresses of existing LAN cluster members (e.g. k8s headless service DNS)
	GossipLANSeeds []string

	// GossipWANSeeds are addresses of cross-region gateways
	GossipWANSeeds []string

	// GossipSecretKey is a base64-encoded shared secret for AES-256 encryption of gossip traffic.
	// When set, nodes must share this key to join and communicate.
	// Generate with: openssl rand -base64 32
	GossipSecretKey string

	// --- ClickHouse proxy configuration ---

	// ChproxyToken is the authentication token for ClickHouse proxy endpoints
	ChproxyToken string

	// --- CTRL service configuration ---

	// CtrlURL is the CTRL service connection URL
	CtrlURL string

	// CtrlToken is the Bearer token for CTRL service authentication
	CtrlToken string

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

	// --- Logging sampler configuration ---

	// LogSampleRate is the baseline probability (0.0-1.0) of emitting log events.
	LogSampleRate float64

	// LogSlowThreshold defines what duration qualifies as "slow" for sampling.
	LogSlowThreshold time.Duration
}

func (c Config) Validate() error {
	// TLS configuration is validated when it's created from files
	// Other validations may be added here in the future
	return nil
}
