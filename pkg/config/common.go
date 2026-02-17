package config

import "time"

// Observability holds configuration for tracing, logging, and metrics collection.
// All fields are optional; omitting a section leaves it nil and enables sensible defaults.
type Observability struct {
	Tracing *TracingConfig `toml:"tracing"`
	Logging *LoggingConfig `toml:"logging"`
	Metrics *MetricsConfig `toml:"metrics"`
}

// MetricsConfig controls Prometheus metrics exposition.
type MetricsConfig struct {
	// PrometheusPort is the TCP port where Prometheus-compatible metrics are served.
	// Set to 0 to disable metrics exposure.
	PrometheusPort int
}

// TracingConfig controls OpenTelemetry tracing and metrics export.
// SampleRate determines what fraction of traces are exported; the rest are dropped
// to reduce storage costs and processing overhead.
type TracingConfig struct {

	// SampleRate is the probability (0.0–1.0) that a trace is sampled.
	SampleRate float64 `toml:"sample_rate" config:"default=0.25,min=0,max=1"`
}

// LoggingConfig controls log sampling. Events faster than SlowThreshold are
// emitted with probability SampleRate; events at or above the threshold are
// always emitted.
type LoggingConfig struct {
	// SampleRate is the probability (0.0–1.0) of emitting a fast log event.
	// Set to 1.0 to log everything.
	SampleRate float64 `toml:"sample_rate" config:"default=1.0,min=0,max=1"`

	// SlowThreshold is the duration above which a request is always logged
	// regardless of SampleRate.
	SlowThreshold time.Duration `toml:"slow_threshold" config:"default=1s"`
}

// DatabaseConfig holds MySQL connection strings. ReadonlyReplica is optional;
// when set, read queries are routed there to reduce load on the primary.
type DatabaseConfig struct {
	// Primary is the MySQL DSN for the read-write database.
	Primary string `toml:"primary" config:"required,nonempty"`

	// ReadonlyReplica is an optional MySQL DSN. When set, read queries are
	// routed here to reduce load on the primary.
	ReadonlyReplica string `toml:"readonly_replica"`
}

// VaultConfig configures the connection to a HashiCorp Vault service for
// encrypting and decrypting sensitive data at rest.
type VaultConfig struct {
	// URL is the vault service endpoint.
	URL string `toml:"url"`

	// Token is the bearer token used to authenticate with the vault service.
	Token string `toml:"token"`
}

// TLSFiles holds paths to PEM-encoded certificate and private key files for TLS.
// Used for serving HTTPS or mTLS connections.
type TLSFiles struct {
	// CertFile is the path to a PEM-encoded TLS certificate.
	CertFile string `toml:"cert_file"`

	// KeyFile is the path to a PEM-encoded TLS private key.
	KeyFile string `toml:"key_file"`
}

// GossipConfig controls memberlist-based distributed cache invalidation.
// Typically referenced as a pointer field on the parent config struct so that
// omitting the [gossip] TOML section leaves it nil, disabling gossip.
type GossipConfig struct {
	// BindAddr is the address to bind gossip listeners on.
	BindAddr string `toml:"bind_addr" config:"default=0.0.0.0"`

	// LANPort is the LAN memberlist port.
	LANPort int `toml:"lan_port" config:"default=7946,min=1,max=65535"`

	// WANPort is the WAN memberlist port for cross-region bridges.
	WANPort int `toml:"wan_port" config:"default=7947,min=1,max=65535"`

	// LANSeeds are addresses of existing LAN cluster members
	// (e.g. k8s headless service DNS).
	LANSeeds []string `toml:"lan_seeds"`

	// WANSeeds are addresses of cross-region bridge nodes.
	WANSeeds []string `toml:"wan_seeds"`

	// SecretKey is a base64-encoded AES-256 key for encrypting gossip traffic.
	// All cluster nodes must share this key. Generate with: openssl rand -base64 32
	SecretKey string `toml:"secret_key" config:"required,min=32,max=128"`
}
