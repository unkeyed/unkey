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
	URL string `toml:"url"`

	// Username is the registry authentication username (e.g. "x-token").
	Username string `toml:"username"`

	// Password is the registry authentication password or token.
	Password string `toml:"password"`
}

// ControlPlaneConfig configures the connection to the control plane that streams
// deployment and sentinel state to krane agents.
type ControlPlaneConfig struct {
	// URL is the control plane endpoint.
	URL string `toml:"url" config:"default=https://control.unkey.cloud"`

	// Bearer is the authentication token for the control plane API.
	Bearer string `toml:"bearer" config:"required,nonempty"`
}

// Config holds the complete configuration for the krane agent. It is designed
// to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[krane.Config]("/etc/unkey/krane.toml")
//
// Environment variables are expanded in file values using ${VAR} or
// ${VAR:-default} syntax before parsing. Struct tag defaults are applied to
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

	// PrometheusPort starts a Prometheus /metrics endpoint on the specified
	// port. Set to 0 to disable.
	PrometheusPort int `toml:"prometheus_port"`

	// Registry configures container image registry access. See [RegistryConfig].
	Registry RegistryConfig `toml:"registry"`

	// Vault configures the secrets decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// ControlPlane configures the upstream control plane. See [ControlPlaneConfig].
	ControlPlane ControlPlaneConfig `toml:"control_plane"`

	// Otel configures OpenTelemetry export. See [config.OtelConfig].
	Otel config.OtelConfig `toml:"otel"`

	// Logging configures log sampling. See [config.LoggingConfig].
	Logging config.LoggingConfig `toml:"logging"`

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
