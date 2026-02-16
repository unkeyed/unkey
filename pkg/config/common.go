package config

import "time"

// OtelConfig controls OpenTelemetry tracing and metrics export. When Enabled
// is false, no collector connection is established and no spans are recorded.
type OtelConfig struct {
	// Enabled activates OpenTelemetry tracing and metrics export to the
	// collector endpoint. Defaults to false.
	Enabled bool `toml:"enabled" config:"default=false"`

	// TraceSamplingRate is the probability (0.0–1.0) that any given trace is
	// sampled. Lower values reduce overhead in high-throughput deployments.
	// Only meaningful when Enabled is true.
	TraceSamplingRate float64 `toml:"trace_sampling_rate" config:"default=0.25,min=0,max=1"`
}

// LoggingConfig controls log sampling behavior. The sampler reduces log volume
// in production while ensuring slow requests are always captured. Events
// faster than SlowThreshold are emitted with probability SampleRate; events
// at or above the threshold are always emitted.
type LoggingConfig struct {
	// SampleRate is the baseline probability (0.0–1.0) of emitting a log event
	// that completes faster than SlowThreshold. Set to 1.0 to log everything.
	SampleRate float64 `toml:"sample_rate" config:"default=1.0,min=0,max=1"`

	// SlowThreshold is the duration above which a request is considered slow
	// and always logged regardless of SampleRate. Uses Go duration syntax
	// (e.g. "1s", "500ms", "2m30s").
	SlowThreshold time.Duration `toml:"slow_threshold" config:"default=1s"`
}

// DatabaseConfig holds connection strings for the primary MySQL database and an
// optional read-replica. The primary is required for all deployments; the replica
// reduces read load on the primary when set.
type DatabaseConfig struct {
	// Primary is the MySQL DSN for the read-write database.
	// Example: "user:pass@tcp(host:3306)/unkey?parseTime=true&interpolateParams=true"
	Primary string `toml:"primary" config:"required,nonempty"`

	// ReadonlyReplica is an optional MySQL DSN for a read-replica. When set,
	// read queries are routed here to reduce load on the primary. The connection
	// string format is identical to Primary.
	ReadonlyReplica string `toml:"readonly_replica"`
}

// VaultConfig configures the connection to a remote vault service used for
// encrypting and decrypting sensitive data. When URL is empty,
// vault-dependent features are disabled.
type VaultConfig struct {
	// URL is the vault service endpoint.
	// Example: "http://vault:8060"
	URL string `toml:"url"`

	// Token is the bearer token used to authenticate with the vault service.
	Token string `toml:"token"`
}

// TLSFiles holds filesystem paths to a TLS certificate and private key.
// Both fields must be set together to enable HTTPS; setting only one is a
// validation error.
type TLSFiles struct {
	// CertFile is the filesystem path to a PEM-encoded TLS certificate.
	CertFile string `toml:"cert_file"`

	// KeyFile is the filesystem path to a PEM-encoded TLS private key.
	KeyFile string `toml:"key_file"`
}
