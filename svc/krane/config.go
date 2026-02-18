package krane

import (
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
)

// RegistryConfig holds credentials for the container image registry used when
// pulling deployment images. All fields are optional; when URL is empty, the
// default registry configured on the cluster is used.
type RegistryConfig struct {
	// URL is the container registry endpoint (e.g. "registry.depot.dev").
	URL string `toml:"url" config:"required"`

	// Username is the registry authentication username (e.g. "x-token").
	Username string `toml:"username" config:"required"`

	// Password is the registry authentication password or token.
	Password string `toml:"password" config:"required"`
}

// Config holds the complete configuration for the krane agent. It is designed
// to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[krane.Config]("/etc/unkey/krane.toml")
//
// Environment variables are expanded in file values using ${VAR}
// syntax before parsing. Struct tag defaults are applied to
// any field left at its zero value after parsing, and validation runs
// automatically via [Config.Validate].
//
// The Clock field is runtime-only and cannot be set through a config file.
type Config struct {
	// InstanceID is the unique identifier for this krane agent instance.
	InstanceID string `toml:"instance_id"`

	// Region identifies the geographic region where this node is deployed.
	Region string `toml:"region" config:"required,nonempty"`

	// RPCPort is the TCP port for the gRPC server.
	RPCPort int `toml:"rpc_port" config:"default=8070,min=1,max=65535"`

	// Registry configures container image registry access. See [RegistryConfig].
	Registry *RegistryConfig `toml:"registry"`

	// Vault configures the secrets decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Control configures the upstream control plane. See [config.ControlConfig].
	Control config.ControlConfig `toml:"control"`

	Observability config.Observability `toml:"observability"`

	// Clock provides time operations and is injected for testability. Production
	// callers set this to [clock.New]; tests can substitute a fake clock.
	Clock clock.Clock `toml:"-"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
func (c *Config) Validate() error {
	return nil
}
