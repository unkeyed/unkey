package config

import "time"

// OtelConfig controls OpenTelemetry tracing and metrics export.
type OtelConfig struct {
	// Enabled activates trace and metric export to the collector.
	Enabled bool `toml:"enabled" config:"default=false"`

	// TraceSamplingRate is the probability (0.0–1.0) that a trace is sampled.
	TraceSamplingRate float64 `toml:"trace_sampling_rate" config:"default=0.25,min=0,max=1"`
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

// VaultConfig configures the connection to a remote vault service for
// encrypting and decrypting sensitive data.
type VaultConfig struct {
	// URL is the vault service endpoint.
	URL string `toml:"url"`

	// Token is the bearer token used to authenticate with the vault service.
	Token string `toml:"token"`
}

// TLSFiles holds PEM file paths for a TLS certificate and private key.
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
