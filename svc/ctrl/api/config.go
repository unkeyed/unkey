package api

import (
	"github.com/unkeyed/unkey/pkg/config"
)

// RestateConfig holds configuration for Restate workflow engine integration.
//
// The API is a Restate client that invokes workflows. It only needs the
// ingress URL and optional API key for authentication.
type RestateConfig struct {
	// URL is the Restate ingress endpoint URL for workflow invocation.
	// Used by clients to start and interact with workflow executions.
	URL string `toml:"url" config:"default=http://restate:8080"`

	// AdminURL is the Restate admin API endpoint for managing invocations.
	// Used for canceling invocations.
	AdminURL string `toml:"admin_url" config:"default=http://restate:9070"`

	// APIKey is the authentication key for Restate ingress requests.
	// If set, this key will be sent with all requests to the Restate ingress.
	APIKey string `toml:"api_key"`
}

// GitHubConfig holds GitHub App integration settings for webhook-triggered
// deployments.
type GitHubConfig struct {
	// WebhookSecret is the secret used to verify webhook signatures.
	// Configured in the GitHub App webhook settings.
	WebhookSecret string `toml:"webhook_secret"`
}

// Config holds the complete configuration for the control plane API server.
// It is designed to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[api.Config]("/etc/unkey/ctrl-api.toml")
//
// Environment variables are expanded in file values using ${VAR} or
// ${VAR:-default} syntax before parsing. Struct tag defaults are applied to
// any field left at its zero value after parsing, and validation runs
// automatically via [Config.Validate].
//
// TLSConfig is runtime-only and cannot be set through a config file. It is
// tagged toml:"-" and must be set programmatically after loading.
type Config struct {
	// InstanceID is the unique identifier for this control plane instance.
	// Used for logging, tracing, and cluster coordination.
	InstanceID string `toml:"instance_id"`

	// Region is the geographic region where this control plane instance runs.
	// Used for logging, tracing, and region-aware routing decisions.
	Region string `toml:"region" config:"required,nonempty"`

	// HttpPort defines the HTTP port for the control plane server.
	// Default: 7091. Cannot be 0.
	HttpPort int `toml:"http_port" config:"default=7091,min=1,max=65535"`

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int `toml:"prometheus_port"`

	// AuthToken is the authentication token for control plane API access.
	// Used by clients and services to authenticate with this control plane.
	AuthToken string `toml:"auth_token" config:"required,nonempty"`

	// AvailableRegions is a list of available regions for deployments.
	// Typically in the format "region.provider", ie "us-east-1.aws", "local.dev"
	AvailableRegions []string `toml:"available_regions"`

	// DefaultDomain is the fallback domain for system operations.
	// Used for wildcard certificate bootstrapping. When set, the API will
	// ensure a wildcard certificate exists for *.{DefaultDomain}.
	DefaultDomain string `toml:"default_domain"`

	// RegionalDomain is the base domain for cross-region communication
	// between frontline instances. Combined with AvailableRegions to create
	// per-region wildcard certificates like *.{region}.{RegionalDomain}.
	RegionalDomain string `toml:"regional_domain"`

	// CnameDomain is the base domain for custom domain CNAME targets.
	// Each custom domain gets a unique subdomain like "{random}.{CnameDomain}".
	CnameDomain string `toml:"cname_domain"`

	// Database configures MySQL connections. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	// Otel configures OpenTelemetry export. See [config.OtelConfig].
	Otel config.OtelConfig `toml:"otel"`

	// Restate configures workflow engine integration. See [RestateConfig].
	Restate RestateConfig `toml:"restate"`

	// GitHub configures GitHub App webhook integration. See [GitHubConfig].
	GitHub GitHubConfig `toml:"github"`

	// Logging configures log sampling. See [config.LoggingConfig].
	Logging config.LoggingConfig `toml:"logging"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
func (c *Config) Validate() error {

	return nil
}
