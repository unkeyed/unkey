package frontline

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/config"
)

// ClickHouseConfig configures connections to ClickHouse for analytics storage.
// When URL is empty, a no-op analytics backend is used.
type ClickHouseConfig struct {
	// URL is the ClickHouse connection string.
	URL string `toml:"url"`

	// BatchSize is the maximum number of items to collect before flushing to ClickHouse.
	// Applies to all event buffers (frontline requests, key verifications).
	BatchSize int `toml:"batch_size" config:"default=5000,min=1"`

	// BufferSize is the capacity of the channel buffer holding incoming items.
	// When full, new items are silently dropped.
	BufferSize int `toml:"buffer_size" config:"default=10000,min=1"`

	// Consumers is the number of goroutines that drain each buffer.
	Consumers int `toml:"consumers" config:"default=1,min=1"`
}

// RedisConfig configures the Redis connection used for rate limiting and
// usage limiting in the policy engine.
type RedisConfig struct {
	// URL is the Redis connection string. When empty, an in-memory counter
	// is used as fallback (rate limits are not shared across replicas).
	URL string `toml:"url"`
}

// Config holds the complete configuration for the frontline server. It is
// designed to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[frontline.Config]("/etc/unkey/frontline.toml")
//
// InstanceID and Image are runtime-only fields set programmatically after
// loading and tagged toml:"-".
type Config struct {
	// InstanceID is the unique identifier for this instance of the frontline
	// server.
	InstanceID string `toml:"instance_id"`

	// Image is the container image identifier including repository and tag.
	// Set at runtime; not read from the config file.
	Image string `toml:"-"`

	// HttpPort is the TCP port the plain-HTTP listener binds to. It serves
	// ACME HTTP-01 challenges (Let's Encrypt) and 308-redirects everything
	// else to https://.
	HttpPort int `toml:"http_port" config:"default=7070,min=1,max=65535"`

	// HttpsPort is the TCP port the HTTPS frontline server binds to. It
	// terminates TLS, runs the policy engine, and forwards customer traffic
	// to a deployment instance (or to a peer frontline in another region).
	HttpsPort int `toml:"https_port" config:"default=7443,min=1,max=65535"`

	// Platform identifies the cloud provider
	// ie: aws, gcp, local
	Platform string `toml:"platform" config:"required"`

	// Region identifies the geographic region where this node is deployed.
	// Used for observability, latency optimization, and cross-region routing.
	Region string `toml:"region" config:"required"`

	// ApexDomain is the apex domain for region routing. Cross-region requests
	// are forwarded to frontline.{region}.{ApexDomain}.
	ApexDomain string `toml:"apex_domain" config:"default=unkey.cloud"`

	// MaxHops is the maximum number of frontline hops allowed before rejecting
	// the request. Prevents infinite routing loops.
	MaxHops int `toml:"max_hops" config:"default=10"`

	// CtrlAddr is the address of the control plane service.
	CtrlAddr string `toml:"ctrl_addr" config:"default=localhost:8080"`

	// PrometheusPort starts a Prometheus /metrics HTTP endpoint on the
	// specified port. Set to 0 to disable.
	PrometheusPort int `toml:"prometheus_port"`

	// TLS provides filesystem paths for HTTPS certificate and key.
	// TLS is enabled by default even if omitted
	// See [config.TLS].
	TLS *config.TLS `toml:"tls"`

	// Database configures the MySQL primary + readonly replica. The
	// routing/cert lookups read from the readonly replica; the policy
	// engine uses the primary for credit decrements during key
	// verification. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	// ClickHouse configures analytics storage for request-level events.
	// See [ClickHouseConfig].
	ClickHouse ClickHouseConfig `toml:"clickhouse"`

	// Redis configures the Redis connection for distributed rate limiting
	// and usage limiting. Optional — falls back to in-memory when empty.
	Redis RedisConfig `toml:"redis"`

	// RequestTimeout is the maximum duration for proxied requests before the
	// context is cancelled and a 504 is returned.
	RequestTimeout time.Duration `toml:"request_timeout" config:"default=15m"`

	Observability config.Observability `toml:"observability"`

	// Vault configures the encryption/decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Gossip configures distributed cache invalidation. See [config.GossipConfig].
	// When nil (section omitted), gossip is disabled and invalidation is local-only.
	Gossip *config.GossipConfig `toml:"gossip"`

	// Pprof configures Go pprof profiling endpoints at /_unkey/internal/pprof/*.
	// When nil or credentials are empty, pprof is disabled.
	Pprof *config.PprofConfig `toml:"pprof"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
//
// Currently validates that TLS is either fully configured (both cert and key)
// or explicitly disabled — partial TLS configuration is an error.
func (c *Config) Validate() error {
	if c.TLS != nil && !c.TLS.Disabled && (c.TLS.CertFile == "") != (c.TLS.KeyFile == "") {
		return fmt.Errorf("both tls.cert_file and tls.key_file must be provided together when TLS is not disabled")
	}
	return nil
}
