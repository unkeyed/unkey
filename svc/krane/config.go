package krane

import (
	"net/netip"

	"github.com/unkeyed/unkey/pkg/assert"
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

// K8sConfig tunes the client-go REST config used to talk to the cluster.
//
// Client-go defaults (QPS=5, Burst=10) trigger multi-second client-side
// throttling on the pod watch hot path, so krane raises them. Exposed as
// config so operators can adjust per environment without a rebuild.
type K8sConfig struct {
	// QPS is the steady-state request rate for the k8s client.
	QPS int `toml:"qps" config:"default=100,min=1"`

	// Burst is the maximum burst size for the k8s client.
	Burst int `toml:"burst" config:"default=200,min=1"`
}

// PrivateNetworkConfig controls private network discovery and customer Pod DNS.
type PrivateNetworkConfig struct {
	// Enabled starts the discovery reconciler and applies ResolverIP to new
	// customer Pods.
	Enabled bool `toml:"enabled"`

	// ResolverIP is the private IPv4 address of this region's undns Service,
	// without a port.
	ResolverIP string `toml:"resolver_ip"`

	// LeaseNamespace holds the Lease that elects one discovery reconciler per
	// cluster.
	LeaseNamespace string `toml:"lease_namespace" config:"default=kube-system,min=1,max=63"`
}

// ClusterConfig identifies the infrastructure cell where krane is running.
type ClusterConfig struct {
	// CellID uniquely identifies the cell within the Unkey network.
	CellID string `toml:"cell_id" config:"required,nonempty"`

	// Region identifies the geographic region where the cell is deployed.
	Region string `toml:"region" config:"required,nonempty"`

	// Platform identifies the infrastructure provider (e.g. "aws", "gcp", "local").
	Platform string `toml:"platform" config:"required,nonempty"`
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

	// Cluster identifies the infrastructure cell managed by this krane agent.
	Cluster ClusterConfig `toml:"cluster"`

	// RPCPort is the TCP port for the gRPC server.
	RPCPort int `toml:"rpc_port" config:"default=8070,min=1,max=65535"`

	// Registry configures container image registry access. See [RegistryConfig].
	Registry *RegistryConfig `toml:"registry"`

	// Vault configures the secrets decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Control configures the upstream control plane. See [config.ControlConfig].
	Control config.ControlConfig `toml:"control"`

	// StorageClassName is the Kubernetes StorageClass used for ephemeral EBS volumes.
	// Defaults to "ebs-csi-gp3" (prod). Set to "standard" for local Minikube development.
	StorageClassName string `toml:"storage_class_name" config:"default=ebs-csi-gp3"`

	// DisableGvisor puts user workloads on the node's default runtime instead
	// of the gVisor sandbox. Local development sets it because minikube's
	// gvisor addon installs no working runsc. The zero value keeps the
	// sandbox, so a config that never mentions it still isolates untrusted
	// code
	DisableGvisor bool `toml:"disable_gvisor"`

	// K8s tunes the client-go REST config. See [K8sConfig].
	K8s K8sConfig `toml:"k8s"`

	PrivateNetwork PrivateNetworkConfig `toml:"private_network"`

	Observability config.Observability `toml:"observability"`

	// Clock provides time operations and is injected for testability. Production
	// callers set this to [clock.New]; tests can substitute a fake clock.
	Clock clock.Clock `toml:"-"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
func (c *Config) Validate() error {
	if !c.PrivateNetwork.Enabled {
		return nil
	}

	address, err := netip.ParseAddr(c.PrivateNetwork.ResolverIP)
	return assert.All(
		assert.True(err == nil, "private_network.resolver_ip must be an IP address, not IP:port"),
		assert.True(address.Is4() && address.IsPrivate(), "private_network.resolver_ip must be a private IPv4 address"),
	)
}
