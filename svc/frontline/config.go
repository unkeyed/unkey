package frontline

import "time"

type Config struct {
	// FrontlineID is the unique identifier for this instance of the Frontline server
	FrontlineID string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the Gate server to listen on (default: 7070)
	HttpPort int

	// HttpsPort defines the HTTPS port for the Gate server to listen on (default: 7443)
	HttpsPort int

	// Region identifies the geographic region where this node is deployed.
	// Used for observability, latency optimization, and compliance requirements.
	// Must match the region identifier used by the underlying cloud platform
	// and control plane configuration.
	Region string

	// EnableTLS specifies whether TLS should be enabled for the Frontline server
	EnableTLS bool

	// TLSCertFile is the path to a static TLS certificate file (for dev mode)
	// When set along with TLSKeyFile, frontline uses file-based TLS instead of dynamic certs
	TLSCertFile string

	// TLSKeyFile is the path to a static TLS key file (for dev mode)
	// When set along with TLSCertFile, frontline uses file-based TLS instead of dynamic certs
	TLSKeyFile string

	// ApexDomain is the apex domain for region routing (e.g., unkey.cloud)
	// Cross-region requests are forwarded to frontline.{region}.{ApexDomain}
	// Example: frontline.us-east-1.aws.unkey.cloud
	ApexDomain string

	// MaxHops is the maximum number of frontline hops allowed before rejecting the request
	// This prevents infinite routing loops. Default: 3
	MaxHops int

	// -- Control Plane Configuration ---

	// CtrlAddr is the address of the control plane (e.g., control.unkey.com)
	CtrlAddr string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations
	DatabasePrimary string

	// DatabaseReadonlyReplica is an optional read-replica database connection string for read operations
	DatabaseReadonlyReplica string

	// --- OpenTelemetry configuration ---

	// OtelEnabled specifies whether OpenTelemetry tracing is enabled
	OtelEnabled bool

	// OtelTraceSamplingRate specifies the sampling rate for OpenTelemetry traces (0.0 - 1.0)
	OtelTraceSamplingRate float64

	// PrometheusPort specifies the port for Prometheus metrics
	PrometheusPort int

	// --- Vault Configuration ---

	// VaultURL is the URL of the remote vault service (e.g., http://vault:8080)
	VaultURL string

	// VaultToken is the authentication token for the vault service
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

	// --- Logging sampler configuration ---

	// LogSampleRate is the baseline probability (0.0-1.0) of emitting log events.
	LogSampleRate float64

	// LogSlowThreshold defines what duration qualifies as "slow" for sampling.
	LogSlowThreshold time.Duration
}

func (c Config) Validate() error {
	return nil
}
