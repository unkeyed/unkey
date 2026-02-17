package sentinel

import (
	"fmt"
	"slices"

	"github.com/unkeyed/unkey/pkg/config"
)

// ClickHouseConfig configures connections to ClickHouse for analytics storage.
// When URL is empty, a no-op analytics backend is used.
type ClickHouseConfig struct {
	// URL is the ClickHouse connection string.
	// Example: "clickhouse://default:password@clickhouse:9000?secure=false&skip_verify=true"
	URL string `toml:"url"`
}

// Config holds the complete configuration for the Sentinel server. It is
// designed to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[sentinel.Config]("/etc/unkey/sentinel.toml")
//
// Environment variables are expanded in file values using ${VAR} or
// ${VAR:-default} syntax before parsing. Struct tag defaults are applied to
// any field left at its zero value after parsing, and validation runs
// automatically via [Config.Validate].
type Config struct {
	// SentinelID identifies this particular sentinel instance. Used in log
	// attribution and request tracing.
	SentinelID string `toml:"sentinel_id"`

	// WorkspaceID is the workspace this sentinel serves.
	WorkspaceID string `toml:"workspace_id" config:"required,nonempty"`

	// EnvironmentID identifies which environment this sentinel serves.
	// A single environment may have multiple deployments, and this sentinel
	// handles all of them based on the deployment ID passed in each request.
	EnvironmentID string `toml:"environment_id" config:"required,nonempty"`

	// Region is the geographic region identifier (e.g. "us-east-1.aws").
	// Included in structured logs and used for routing decisions.
	Region string `toml:"region" config:"required,oneof=local.dev|us-east-1.aws|us-east-2.aws|us-west-1.aws|us-west-2.aws|eu-central-1.aws"`

	// HttpPort is the TCP port the sentinel server binds to.
	HttpPort int `toml:"http_port" config:"default=8080,min=1,max=65535"`

	// PrometheusPort starts a Prometheus /metrics HTTP endpoint on the
	// specified port. Set to 0 (the default) to disable the endpoint entirely.
	PrometheusPort int `toml:"prometheus_port"`

	// Database configures MySQL connections. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	// ClickHouse configures analytics storage. See [ClickHouseConfig].
	ClickHouse ClickHouseConfig `toml:"clickhouse"`

	// Otel configures OpenTelemetry export. See [config.OtelConfig].
	Otel config.OtelConfig `toml:"otel"`

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
	validRegions := []string{
		"local.dev",
		"us-east-1.aws",
		"us-east-2.aws",
		"us-west-1.aws",
		"us-west-2.aws",
		"eu-central-1.aws",
	}

	if !slices.Contains(validRegions, c.Region) {
		return fmt.Errorf("invalid region: %s, must be one of %v", c.Region, validRegions)
	}

	return nil
}
