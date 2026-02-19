package frontline

import (
	"fmt"

	"github.com/unkeyed/unkey/pkg/config"
)

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

	// ChallengePort is the TCP port the HTTP challenge server binds to.
	// Used for ACME HTTP-01 challenges (Let's Encrypt).
	ChallengePort int `toml:"challenge_port" config:"default=7070,min=1,max=65535"`

	// HttpPort is the TCP port the HTTP frontline server binds to.
	// Serves general traffic over HTTPS by default.
	HttpPort int `toml:"http_port" config:"default=7443,min=1,max=65535"`

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

	// Database configures MySQL connections. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	Observability config.Observability `toml:"observability"`

	// Vault configures the encryption/decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Gossip configures distributed cache invalidation. See [config.GossipConfig].
	// When nil (section omitted), gossip is disabled and invalidation is local-only.
	Gossip *config.GossipConfig `toml:"gossip"`
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
