package api

import (
	"fmt"
	"net"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/tls"
)

// ClickHouseConfig configures connections to ClickHouse for analytics storage.
// All fields are optional; when URL is empty, a no-op analytics backend is used.
type ClickHouseConfig struct {
	// URL is the ClickHouse connection string for the shared analytics cluster.
	// When empty, analytics writes are silently discarded.
	// Example: "clickhouse://default:password@clickhouse:9000?secure=false&skip_verify=true"
	URL string `toml:"url"`

	// AnalyticsURL is the base URL for workspace-specific analytics connections.
	// Unlike URL, this endpoint receives per-workspace credentials injected at
	// connection time by the analytics service. Only used when both this field
	// and a [VaultConfig] are configured.
	// Example: "http://clickhouse:8123/default"
	AnalyticsURL string `toml:"analytics_url"`

	// ProxyToken is the bearer token for authenticating against ClickHouse proxy
	// endpoints exposed by the API server itself.
	ProxyToken string `toml:"proxy_token"`
}

// CtrlConfig configures the connection to the CTRL service, which manages
// deployments and rolling updates across the cluster.
type CtrlConfig struct {
	// URL is the CTRL service endpoint.
	// Example: "http://ctrl-api:7091"
	URL string `toml:"url"`

	// Token is the bearer token used to authenticate with the CTRL service.
	Token string `toml:"token"`
}

// PprofConfig controls the Go pprof profiling endpoints served at /debug/pprof/*.
// Pprof is enabled when this section is present in the config file and disabled
// when omitted (the field is a pointer on [Config]).
type PprofConfig struct {
	// Username is the Basic Auth username for pprof endpoints. When both
	// Username and Password are empty, pprof endpoints are served without
	// authentication — only appropriate in development environments.
	Username string `toml:"username"`

	// Password is the Basic Auth password for pprof endpoints.
	Password string `toml:"password"`
}

// Config holds the complete configuration for the API server. It is designed to
// be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[api.Config]("/etc/unkey/api.toml")
//
// Environment variables are expanded in file values using ${VAR} or
// ${VAR:-default} syntax before parsing. Struct tag defaults are applied to
// any field left at its zero value after parsing, and validation runs
// automatically via [Config.Validate].
//
// Three fields — Listener, Clock, and TLSConfig — are runtime-only and cannot
// be set through a config file. They are tagged toml:"-" and must be set
// programmatically after loading.
type Config struct {
	// InstanceID identifies this particular API server instance. Used in log
	// attribution, Kafka consumer group membership, and cache invalidation
	// messages so that a node can ignore its own broadcasts.
	InstanceID string `toml:"instance_id"`

	// Platform identifies the cloud platform where this node runs
	// (e.g. "aws", "gcp", "hetzner", "kubernetes"). Appears in structured
	// logs and metrics labels for filtering by infrastructure.
	Platform string `toml:"platform"`

	// Image is the container image identifier (e.g. "unkey/api:v1.2.3").
	// Logged at startup for correlating deployments with behavior changes.
	Image string `toml:"image"`

	// HttpPort is the TCP port the API server binds to. Ignored when Listener
	// is set, which is the case in test harnesses that use ephemeral ports.
	HttpPort int `toml:"http_port" config:"default=7070,min=1,max=65535"`

	// Region is the geographic region identifier (e.g. "us-east-1", "eu-west-1").
	// Included in structured logs and used by the key service when recording
	// which region served a verification request.
	Region string `toml:"region" config:"default=unknown"`

	// RedisURL is the connection string for the Redis instance backing
	// distributed rate limiting counters and usage tracking.
	// Example: "redis://redis:6379"
	RedisURL string `toml:"redis_url" config:"required,nonempty"`

	// TestMode relaxes certain security checks and trusts client-supplied
	// headers that would normally be rejected. This exists for integration
	// tests that need to inject specific request metadata.
	// Do not enable in production.
	TestMode bool `toml:"test_mode" config:"default=false"`

	// PrometheusPort starts a Prometheus /metrics HTTP endpoint on the
	// specified port. Set to 0 (the default) to disable the endpoint entirely.
	PrometheusPort int `toml:"prometheus_port"`

	// MaxRequestBodySize caps incoming request bodies at this many bytes.
	// The zen server rejects requests exceeding this limit with a 413 status.
	// Set to 0 or negative to disable the limit. Defaults to 10 MiB.
	MaxRequestBodySize int64 `toml:"max_request_body_size" config:"default=10485760"`

	// Database configures MySQL connections. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	// ClickHouse configures analytics storage. See [ClickHouseConfig].
	ClickHouse ClickHouseConfig `toml:"clickhouse"`

	// Otel configures OpenTelemetry export. See [config.OtelConfig].
	Otel config.OtelConfig `toml:"otel"`

	// TLS provides filesystem paths for HTTPS certificate and key.
	// See [config.TLSFiles].
	TLS config.TLSFiles `toml:"tls"`

	// Vault configures the encryption/decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Gossip configures distributed cache invalidation. See [config.GossipConfig].
	// When nil (section omitted), gossip is disabled and invalidation is local-only.
	Gossip *config.GossipConfig `toml:"gossip"`

	// Ctrl configures the deployment management service. See [CtrlConfig].
	Ctrl CtrlConfig `toml:"ctrl"`

	// Pprof configures Go profiling endpoints. See [PprofConfig].
	// When nil (section omitted), pprof endpoints are not registered.
	Pprof *PprofConfig `toml:"pprof"`

	// Logging configures log sampling. See [config.LoggingConfig].
	Logging config.LoggingConfig `toml:"logging"`

	// Listener is a pre-created [net.Listener] for the HTTP server. When set,
	// the server uses this listener instead of binding to HttpPort. This is
	// intended for tests that need ephemeral ports to avoid conflicts.
	Listener net.Listener `toml:"-"`

	// Clock provides time operations and is injected for testability. Production
	// callers set this to [clock.New]; tests can substitute a fake clock to
	// control time progression.
	Clock clock.Clock `toml:"-"`

	// TLSConfig is the resolved [tls.Config] built from [TLSFiles.CertFile]
	// and [TLSFiles.KeyFile] at startup. This field is populated by the CLI
	// entrypoint after loading the config file and must not be set in TOML.
	TLSConfig *tls.Config `toml:"-"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
//
// Currently validates that TLS certificate and key paths are either both
// provided or both absent — setting only one is an error.
func (c *Config) Validate() error {
	certFile := c.TLS.CertFile
	keyFile := c.TLS.KeyFile
	if (certFile == "") != (keyFile == "") {
		return fmt.Errorf("both tls.cert_file and tls.key_file must be provided to enable HTTPS")
	}
	return nil
}
