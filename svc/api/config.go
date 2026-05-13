package api

import (
	"fmt"
	"net"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/counter"
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
}

// Config holds the complete configuration for the API server. It is designed to
// be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[api.Config]("/etc/unkey/api.toml")
//
// Environment variables are expanded in file values using ${VAR}
// syntax before parsing. Struct tag defaults are applied to
// any field left at its zero value after parsing, and validation runs
// automatically via [Config.Validate].
//
// Several fields — Clock, TLSConfig, and the [TestConfig] group — are
// runtime-only and cannot be set through a config file. They are tagged
// toml:"-" and must be set programmatically after loading.
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

	// HttpPort is the TCP port the API server binds to. Ignored when
	// [TestConfig.Listener] is set, which is the case in test harnesses that
	// use ephemeral ports.
	HttpPort int `toml:"http_port" config:"default=7070,min=1,max=65535"`

	// Region is the geographic region identifier (e.g. "us-east-1", "eu-west-1").
	// Included in structured logs and used by the key service when recording
	// which region served a verification request.
	Region string `toml:"region" config:"default=unknown"`

	// RedisURL is the connection string for the Redis instance backing
	// distributed rate limiting counters and usage tracking.
	// Example: "redis://redis:6379"
	RedisURL string `toml:"redis_url" config:"required,nonempty"`

	Observability config.Observability `toml:"observability"`

	// MaxRequestBodySize caps incoming request bodies at this many bytes.
	// The zen server rejects requests exceeding this limit with a 413 status.
	// Set to 0 or negative to disable the limit. Defaults to 10 MiB.
	MaxRequestBodySize int64 `toml:"max_request_body_size" config:"default=10485760"`

	// Database configures MySQL connections. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	// ClickHouse configures analytics storage. See [ClickHouseConfig].
	ClickHouse ClickHouseConfig `toml:"clickhouse"`

	// TLS provides filesystem paths for HTTPS certificate and key.
	// See [config.TLSFiles].
	TLS config.TLS `toml:"tls"`

	// Vault configures the encryption/decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Gossip configures distributed cache invalidation. See [config.GossipConfig].
	// When nil (section omitted), gossip is disabled and invalidation is local-only.
	Gossip *config.GossipConfig `toml:"gossip"`

	// Control configures the deployment management service. See [config.ControlConfig].
	Control config.ControlConfig `toml:"control"`

	// PortalBaseURL is the base URL for the customer portal (e.g. "https://portal.unkey.com").
	// Used to construct session redirect URLs in portal.createSession responses.
	// When a customer has a verified custom domain, that domain is used instead.
	PortalBaseURL string `toml:"portal_base_url" config:"default=https://portal.unkey.com"`

	// Pprof configures Go profiling endpoints. See [config.PprofConfig].
	// When nil (section omitted), pprof endpoints are not registered.
	Pprof *config.PprofConfig `toml:"pprof"`

	// Clock provides time operations and is injected for testability. Production
	// callers set this to [clock.New]; tests can substitute a fake clock to
	// control time progression.
	Clock clock.Clock `toml:"-"`

	// TLSConfig is the resolved [tls.Config] built from [TLSFiles.CertFile]
	// and [TLSFiles.KeyFile] at startup. This field is populated by the CLI
	// entrypoint after loading the config file and must not be set in TOML.
	TLSConfig *tls.Config `toml:"-"`

	// Test groups runtime-only overrides for integration tests. Zero in
	// production — no fields can be set from TOML.
	Test TestConfig `toml:"-"`
}

// TestConfig groups runtime-only flags and overrides used by integration
// tests. All fields are zero in production; setting any of them enables
// test-specific behavior that MUST NOT be reachable from a TOML config file.
type TestConfig struct {
	// Enabled relaxes certain security checks and trusts client-supplied
	// headers like X-Test-Time that would normally be rejected.
	Enabled bool

	// Counter overrides the distributed counter backend. Multi-node tests
	// share one in-memory counter across all nodes so replays sync in
	// microseconds rather than blocking on real Redis I/O.
	Counter counter.Counter

	// Listener is a pre-created net.Listener for the HTTP server. When set,
	// the server uses this listener instead of binding to HttpPort. Tests
	// use ephemeral ports (":0") to avoid conflicts when running in parallel.
	Listener net.Listener
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
