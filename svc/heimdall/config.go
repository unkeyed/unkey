package heimdall

import "github.com/unkeyed/unkey/pkg/config"

// Config holds the configuration for the heimdall resource metering agent.
// It is loaded from a TOML file using [config.Load].
type Config struct {
	// Region identifies the geographic region where this node is deployed.
	Region string `toml:"region" config:"required,nonempty"`

	// Platform identifies the infrastructure provider (e.g. "aws", "gcp", "local").
	Platform string `toml:"platform" config:"required,nonempty"`

	// ClickHouseURL is the ClickHouse connection string for writing metering data.
	ClickHouseURL string `toml:"clickhouse_url" config:"required,nonempty"`

	// CollectionInterval is how often to scrape kubelet for resource usage.
	// Parsed as a Go duration string (e.g. "15s", "30s", "1m").
	CollectionInterval string `toml:"collection_interval" config:"default=15s"`

	// NodeIP is the IP address of the node this agent runs on.
	// Used to reach the kubelet API at https://<node_ip>:10250.
	// Typically injected via the Kubernetes downward API (status.hostIP).
	NodeIP string `toml:"node_ip" config:"default=localhost"`

	Observability config.Observability `toml:"observability"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone.
func (c *Config) Validate() error {
	return nil
}
