package frontline

import (
	"fmt"

	"github.com/unkeyed/unkey/pkg/config"
)

// TLSConfig controls TLS termination for the frontline server. When Enabled
// is true and both CertFile and KeyFile are set, the server uses static
// file-based TLS (dev mode). When only Enabled is true and a vault/cert
// manager is available, dynamic certificates are used (production mode).
type TLSConfig struct {
	// Enabled activates TLS termination. Defaults to true.
	Enabled bool `toml:"enabled" config:"default=true"`

	// CertFile is the filesystem path to a PEM-encoded TLS certificate.
	// Used together with KeyFile for static file-based TLS (dev mode).
	CertFile string `toml:"cert_file"`

	// KeyFile is the filesystem path to a PEM-encoded TLS private key.
	// Used together with CertFile for static file-based TLS (dev mode).
	KeyFile string `toml:"key_file"`
}

// Config holds the complete configuration for the frontline server. It is
// designed to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[frontline.Config]("/etc/unkey/frontline.toml")
//
// FrontlineID and Image are runtime-only fields set programmatically after
// loading and tagged toml:"-".
type Config struct {
	// FrontlineID is the unique identifier for this instance of the frontline
	// server. Set at runtime; not read from the config file.
	FrontlineID string `toml:"-"`

	// Image is the container image identifier including repository and tag.
	// Set at runtime; not read from the config file.
	Image string `toml:"-"`

	// HttpPort is the TCP port the HTTP challenge server binds to.
	HttpPort int `toml:"http_port" config:"default=7070,min=1,max=65535"`

	// HttpsPort is the TCP port the HTTPS frontline server binds to.
	HttpsPort int `toml:"https_port" config:"default=7443,min=1,max=65535"`

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

	// TLS controls TLS termination. See [TLSConfig].
	TLS TLSConfig `toml:"tls"`

	// Database configures MySQL connections. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	Observability config.Observability `toml:"observability"`

	// Vault configures the encryption/decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Gossip configures distributed cache invalidation. See [config.GossipConfig].
	// When nil (section omitted), gossip is disabled and invalidation is local-only.
	Gossip *config.GossipConfig `toml:"gossip"`

	// Logging configures log sampling. See [config.LoggingConfig].
	Logging config.LoggingConfig `toml:"logging"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
func (c *Config) Validate() error {
	if (c.TLS.CertFile == "") != (c.TLS.KeyFile == "") {
		return fmt.Errorf("both tls.cert_file and tls.key_file must be provided together")
	}
	return nil
}
