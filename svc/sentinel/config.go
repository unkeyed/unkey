package sentinel

import (
	"fmt"

	"github.com/unkeyed/unkey/pkg/config"
)

// ClickHouseConfig configures connections to ClickHouse for analytics storage.
// When URL is empty, a no-op analytics backend is used.
type ClickHouseConfig struct {
	// URL is the ClickHouse connection string.
	// Example: "clickhouse://default:password@clickhouse:9000?secure=false&skip_verify=true"
	URL string `toml:"url"`
}

// RedisConfig configures the Redis connection used for rate limiting
// and usage limiting in sentinel middleware policies.
type RedisConfig struct {
	// URL is the Redis connection string.
	// Example: "redis://default:password@redis:6379"
	// When empty, the middleware engine (KeyAuth, rate limiting) is disabled.
	URL string `toml:"url"`
}

// Config holds the complete configuration for the Sentinel server. It is
// designed to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[sentinel.Config]("/etc/unkey/sentinel.toml")
//
// Environment variables are expanded in file values using ${VAR}
// syntax before parsing. Struct tag defaults are applied to
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

	// Platform identifies the underlying cloud platform this sentinel is running on.
	Platform string `toml:"platform" config:"required,nonempty"`

	// Region is the geographic region identifier (e.g. "us-east-1").
	// Included in structured logs and used for routing decisions.
	Region string `toml:"region" config:"required,nonempty"`

	// RegionID is the stable database identifier for this region.
	// When set, Sentinel skips region-name lookup queries in the router path.
	//
	// TODO: figure out how/if we can delete Platform and Region fields after we have RegionID in place everywhere.
	// We may want to keep them for human-friendly logging and metrics, but they are redundant with RegionID.
	RegionID string `toml:"region_id"`

	// HttpPort is the TCP port the sentinel server binds to.
	HttpPort int `toml:"http_port" config:"default=8080,min=1,max=65535"`

	// Observability configures tracing, logging, and metrics. See [config.Observability].
	Observability config.Observability `toml:"observability"`

	// DatabaseURL is the MySQL read-only replica connection string.
	DatabaseURL string `toml:"database_url"`

	// ClickHouse configures analytics storage. See [ClickHouseConfig].
	ClickHouse ClickHouseConfig `toml:"clickhouse"`

	// Redis configures the Redis connection for rate limiting and usage limiting.
	// Required when sentinel middleware policies use KeyAuth with auto-applied rate limits.
	Redis RedisConfig `toml:"redis"`

	// Gossip configures distributed cache invalidation. See [config.GossipConfig].
	// When nil (section omitted), gossip is disabled and invalidation is local-only.
	Gossip *config.GossipConfig `toml:"gossip"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}

	return nil
}
