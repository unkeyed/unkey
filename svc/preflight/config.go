package preflight

import (
	"github.com/unkeyed/unkey/pkg/config"
)

// TLSConfig holds filesystem paths for the TLS certificate and private key
// used by the webhook HTTPS server.
type TLSConfig struct {
	// CertFile is the path to a PEM-encoded TLS certificate.
	CertFile string `toml:"cert_file" config:"required,nonempty"`

	// KeyFile is the path to a PEM-encoded TLS private key.
	KeyFile string `toml:"key_file" config:"required,nonempty"`
}

// InjectConfig controls the container image injected into mutated pods by the
// admission webhook.
type InjectConfig struct {
	// Image is the container image reference for the inject binary.
	Image string `toml:"image" config:"default=inject:latest"`

	// ImagePullPolicy is the Kubernetes image pull policy applied to the
	// injected init container.
	ImagePullPolicy string `toml:"image_pull_policy" config:"default=IfNotPresent,oneof=Always|IfNotPresent|Never"`
}

// RegistryConfig configures container registry behavior for the preflight
// webhook, including insecure registries and alias mappings.
type RegistryConfig struct {
	// InsecureRegistries is a list of registry hostnames that should be
	// contacted over plain HTTP instead of HTTPS.
	InsecureRegistries []string `toml:"insecure_registries"`

	// Aliases is a list of registry alias mappings in "from=to" format.
	// The webhook rewrites image references matching the left-hand side to
	// the right-hand side before pulling.
	Aliases []string `toml:"aliases"`
}

// Config holds the complete configuration for the preflight admission webhook
// server. It is designed to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[preflight.Config]("/etc/unkey/preflight.toml")
//
// Environment variables are expanded in file values using ${VAR} or
// ${VAR:-default} syntax before parsing.
type Config struct {
	// HttpPort is the TCP port the webhook HTTPS server binds to.
	HttpPort int `toml:"http_port" config:"default=8443,min=1,max=65535"`

	// KraneEndpoint is the URL of the Krane secrets provider service.
	KraneEndpoint string `toml:"krane_endpoint" config:"default=http://krane.unkey.svc.cluster.local:8070"`

	// DepotToken is an optional Depot API token for fetching on-demand
	// container registry pull tokens.
	DepotToken string `toml:"depot_token"`

	// TLS provides filesystem paths for HTTPS certificate and key.
	// See [TLSConfig].
	TLS TLSConfig `toml:"tls"`

	// Inject controls the container image injected into mutated pods.
	// See [InjectConfig].
	Inject InjectConfig `toml:"inject"`

	// Registry configures container registry behavior. See [RegistryConfig].
	Registry RegistryConfig `toml:"registry"`

	// Logging configures log sampling. See [config.LoggingConfig].
	Logging config.LoggingConfig `toml:"logging"`
}

// Validate implements [config.Validator] so that [config.Load] calls it
// automatically after tag-level validation. All constraints are expressed
// through struct tags, so this method is a no-op.
func (c *Config) Validate() error {
	return nil
}
