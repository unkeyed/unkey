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
	PrometheusPort int `toml:"prometheus_port"`
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

// TLS holds paths to PEM-encoded certificate and private key files for TLS.
// Used for serving HTTPS or mTLS connections.
// Disabled defaults to false (TLS enabled). Set Disabled = true to explicitly disable TLS.
type TLS struct {
	// Disabled when set to true, disables TLS even when certificate sources are available.
	Disabled bool `toml:"disabled"`

	// CertFile is the path to a PEM-encoded TLS certificate.
	CertFile string `toml:"cert_file"`

	// KeyFile is the path to a PEM-encoded TLS private key.
	KeyFile string `toml:"key_file"`
}

// GossipConfig configures the Serf-backed event bus that distributes cache
// invalidations and other broadcast events. Typically referenced as a
// pointer field on the parent config so that omitting [gossip] leaves it
// nil and the process runs with bus.NewNoop().
type GossipConfig struct {
	// BindAddr is the address to bind gossip listeners on.
	BindAddr string `toml:"bind_addr" config:"default=0.0.0.0"`

	// Port is the gossip port (TCP+UDP). Single port; the legacy LAN/WAN
	// split was removed when the bus collapsed onto one Serf cluster.
	Port int `toml:"port" config:"default=7946,min=1,max=65535"`

	// AdvertiseAddr is the address peers should reach this node on. For
	// pods inside a peered VPC, set this to the pod IP. Empty leaves Serf
	// to derive it from the bind address.
	AdvertiseAddr string `toml:"advertise_addr"`

	// Seeds is the list of addresses to dial at startup. Mix the local
	// k8s headless service with the per-region NLB hostnames; the first
	// reachable one is enough to bootstrap into the mesh.
	Seeds []string `toml:"seeds"`

	// SecretKey is a base64-encoded AES-256 key for encrypting gossip
	// traffic. All cluster nodes must share this key. Generate with:
	// openssl rand -base64 32.
	SecretKey string `toml:"secret_key" config:"required,min=32,max=128"`
}

// ControlConfig configures the connection to the control plane service, which manages
// deployments and rolling updates across the cluster.
type ControlConfig struct {
	// URL is the control plane service endpoint.
	// Example: "http://control-api:7091"
	URL string `toml:"url" config:"required"`

	// Token is the bearer token used to authenticate with the control plane service.
	Token string `toml:"token" config:"required"`
}

// PprofConfig controls the Go pprof profiling endpoints.
// The path prefix is set per-service (e.g. /debug/pprof/* for the API,
// /_unkey/internal/pprof/* for frontline and sentinel).
// Pprof is enabled only when both Username and Password are non-empty;
// otherwise the endpoints are not registered.
type PprofConfig struct {
	// Username is the Basic Auth username for pprof endpoints.
	Username string `toml:"username"`

	// Password is the Basic Auth password for pprof endpoints.
	Password string `toml:"password"`

	// Port is the TCP port for a loopback-only (127.0.0.1) pprof server.
	// When set to a positive value, pprof endpoints are served on an internal
	// listener that is not reachable from outside the host.
	// Defaults to 6060 when omitted.
	Port int `toml:"port" config:"default=6060,min=0,max=65535"`
}
