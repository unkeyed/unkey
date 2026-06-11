package api

import (
	"fmt"
	"net/url"

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
// deployments and deployment authorization.
type GitHubConfig struct {
	// WebhookSecret is the secret used to verify webhook signatures.
	// Configured in the GitHub App webhook settings.
	WebhookSecret string `toml:"webhook_secret"`

	// AppID is the GitHub App ID for authentication.
	// Required for deployment authorization (fetching branch HEAD).
	AppID int64 `toml:"app_id"`

	// PrivateKeyPEM is the GitHub App private key in PEM format.
	// Required for deployment authorization (fetching branch HEAD).
	PrivateKeyPEM string `toml:"private_key_pem"`

	// AllowUnauthenticatedDeployments controls whether deployments can skip
	// GitHub authentication. Set to true only for local development.
	// Production should keep this false to require GitHub App authentication.
	AllowUnauthenticatedDeployments bool `toml:"allow_unauthenticated_deployments"`
}

// StripeConfig holds the Stripe integration for the month-end Deploy billing
// close: the invoice.created webhook claims renewal invoices of Deploy
// workspaces (auto_advance off) and dispatches the closing flow to the
// worker via Restate. Both fields empty disables the webhook entirely.
type StripeConfig struct {
	// WebhookSecret verifies Stripe webhook signatures. Empty disables the
	// /webhooks/stripe route.
	WebhookSecret string `toml:"webhook_secret"`

	// SecretKey authenticates the auto_advance claim on draft invoices.
	// Required when WebhookSecret is set.
	SecretKey string `toml:"secret_key"`
}

// DomainConnectConfig holds Domain Connect protocol configuration for
// one-click DNS setup via supported providers.
type DomainConnectConfig struct {
	// PrivateKeyPEM is the PEM-encoded RSA private key for signing
	// Domain Connect redirect URLs. If empty, Domain Connect is disabled.
	PrivateKeyPEM string `toml:"private_key_pem"`
}

// ClickHouseConfig holds ClickHouse connection configuration. The api
// process writes container lifecycle events here when krane reports them
// via ReportInstanceEvents. When URL is empty the writer falls back to a
// noop and the dashboard's events panel will be empty.
type ClickHouseConfig struct {
	URL string `toml:"url"`
}

// Config holds the complete configuration for the control plane API server.
// It is designed to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[api.Config]("/etc/unkey/ctrl-api.toml")
//
// Environment variables are expanded in file values using ${VAR}
// syntax before parsing. Struct tag defaults are applied to
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
	// Default: 8080. Cannot be 0.
	HttpPort int `toml:"http_port" config:"default=8080,min=1,max=65535"`

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int `toml:"prometheus_port"`

	// AuthToken is the authentication token for control plane API access.
	// Used by clients and services to authenticate with this control plane.
	AuthToken string `toml:"auth_token" config:"required,nonempty"`

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

	// Observability configures tracing, logging, and metrics. See [config.Observability].
	Observability config.Observability `toml:"observability"`

	// Restate configures workflow engine integration. See [RestateConfig].
	Restate RestateConfig `toml:"restate"`

	// GitHub configures GitHub App webhook integration. See [GitHubConfig].
	GitHub GitHubConfig `toml:"github"`

	// Stripe configures the Stripe webhook for the month-end Deploy billing
	// close. See [StripeConfig].
	Stripe StripeConfig `toml:"stripe"`

	// DomainConnect configures the Domain Connect protocol for one-click DNS setup.
	// See [DomainConnectConfig].
	DomainConnect DomainConnectConfig `toml:"domain_connect"`

	// ClickHouse configures the analytics database connection used for
	// container lifecycle event ingestion.
	ClickHouse ClickHouseConfig `toml:"clickhouse"`

	// DeployGate configures the Unkey Deploy project-creation entitlement gate.
	DeployGate DeployGateConfig `toml:"deploy_gate"`
}

// DeployGateConfig configures the Unkey Deploy project-creation gate, which
// requires a workspace to have a Deploy entitlement (a synced plan or a manual
// override) before it can create projects.
type DeployGateConfig struct {
	// Enforce hard-blocks project creation for workspaces without a Deploy
	// entitlement. Default false: the gate runs in observe mode, logging when it
	// would block but allowing creation, so the signal can be validated before
	// enforcing. Flip to true (config change + redeploy) to enforce.
	Enforce bool `toml:"enforce"`
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
func (c *Config) Validate() error {
	// ClickHouse.URL is optional (empty means "skip ingestion, use noop
	// sink"), but a non-empty value must be a parseable URL with scheme
	// and host. Letting a malformed value through here means the process
	// boots, the noop sink swallows events for the lifetime of the
	// process, and the failure is invisible until the dashboard is empty.
	if c.ClickHouse.URL != "" {
		u, err := url.Parse(c.ClickHouse.URL)
		if err != nil {
			return fmt.Errorf("invalid clickhouse.url: %w", err)
		}
		if u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid clickhouse.url %q: scheme and host are required", c.ClickHouse.URL)
		}
	}

	return nil
}
