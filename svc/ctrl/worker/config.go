package worker

import (
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
)

// Route53Config holds AWS Route53 configuration for ACME DNS-01 challenges.
//
// This configuration enables automatic DNS record creation for wildcard
// TLS certificates through AWS Route53 DNS API.
type Route53Config struct {
	// Enabled determines whether Route53 DNS-01 challenges are used.
	// When true, wildcard certificates can be automatically obtained.
	Enabled bool `toml:"enabled"`

	// AccessKeyID is the AWS access key ID for Route53 API access.
	AccessKeyID string `toml:"access_key_id"`

	// SecretAccessKey is the AWS secret access key for Route53 API access.
	SecretAccessKey string `toml:"secret_access_key"`

	// Region is the AWS region where Route53 hosted zones are located.
	// Example: "us-east-1", "us-west-2".
	Region string `toml:"region" config:"default=us-east-1"`

	// HostedZoneID overrides automatic zone discovery.
	// Required when domains have complex CNAME setups that confuse
	// automatic zone lookup (e.g., wildcard CNAMEs to load balancers).
	HostedZoneID string `toml:"hosted_zone_id"`
}

// AcmeConfig holds configuration for ACME TLS certificate management.
//
// This configuration enables automatic certificate issuance and renewal
// through ACME protocol with support for multiple DNS providers.
type AcmeConfig struct {
	// Enabled determines whether ACME certificate management is active.
	// When true, certificates are automatically obtained and renewed.
	Enabled bool `toml:"enabled"`

	// EmailDomain is the domain used for ACME account emails.
	// Used for Let's Encrypt account registration and recovery.
	// Example: "unkey.com" creates "admin@unkey.com" for ACME account.
	EmailDomain string `toml:"email_domain" config:"default=unkey.com"`

	// Route53 configures DNS-01 challenges through AWS Route53 API.
	// Enables wildcard certificates for domains hosted on Route53.
	Route53 Route53Config `toml:"route53"`
}

// RestateConfig holds configuration for Restate workflow engine integration.
//
// This configuration enables asynchronous workflow execution through
// the Restate distributed system for deployment and certificate operations.
type RestateConfig struct {
	// AdminURL is the Restate admin endpoint URL for service registration.
	// Used by the worker to register its workflow services.
	// Example: "http://restate:9070".
	AdminURL string `toml:"admin_url" config:"default=http://restate:9070"`

	// APIKey is the optional authentication key for Restate admin API requests.
	// If set, this key will be sent with all requests to the Restate admin API.
	APIKey string `toml:"api_key"`

	// HttpPort is the port where the worker listens for Restate requests.
	// This is the internal Restate server port, not the health check port.
	HttpPort int `toml:"http_port" config:"default=9080,min=1,max=65535"`

	// RegisterAs is the service URL used for self-registration with Restate.
	// Allows Restate to discover and invoke this worker's services.
	// Example: "http://worker:9080".
	RegisterAs string `toml:"register_as"`
}

// DepotConfig holds configuration for Depot.dev build service integration.
//
// This configuration enables cloud-native container builds through
// Depot's managed build infrastructure with optimized caching.
type DepotConfig struct {
	// APIUrl is the Depot API endpoint URL for build operations.
	// Example: "https://api.depot.dev".
	APIUrl string `toml:"api_url"`

	// ProjectRegion is the geographic region for build storage.
	// Affects build performance and data residency.
	// Options: "us-east-1", "eu-central-1". Default: "us-east-1".
	ProjectRegion string `toml:"project_region" config:"default=us-east-1"`
}

// RegistryConfig holds container registry authentication configuration.
//
// This configuration provides credentials for accessing container registries
// used by build backends for pushing and pulling images.
type RegistryConfig struct {
	// URL is the container registry endpoint URL.
	// Example: "registry.depot.dev" or "https://registry.example.com".
	URL string `toml:"url"`

	// Username is the registry authentication username.
	// Common values: "x-token" for token-based auth, or actual username.
	Username string `toml:"username"`

	// Password is the registry password or authentication token.
	// Should be stored securely and rotated regularly.
	Password string `toml:"password"`
}

// ClickHouseConfig holds ClickHouse connection configuration.
type ClickHouseConfig struct {
	// URL is the ClickHouse database connection string.
	// Used for analytics and operational metrics storage.
	URL string `toml:"url"`

	// AdminURL is the connection string for the ClickHouse admin user.
	// Used by ClickhouseUserService to create/configure workspace users.
	// Optional - if not set, ClickhouseUserService will not be enabled.
	AdminURL string `toml:"admin_url"`
}

// BuildPlatform represents parsed container build platform specification.
//
// Contains the validated platform string separated into OS and architecture
// components for build backend integration.
type BuildPlatform struct {
	// Platform is the original build platform string.
	// Example: "linux/amd64".
	Platform string

	// Architecture is the CPU architecture component.
	// Example: "amd64", "arm64".
	Architecture string
}

// GitHubConfig holds configuration for GitHub App integration.
type GitHubConfig struct {
	// AppID is the GitHub App ID for authentication.
	AppID int64 `toml:"app_id"`

	// PrivateKeyPEM is the GitHub App private key in PEM format.
	PrivateKeyPEM string `toml:"private_key_pem"`

	// AllowUnauthenticatedDeployments controls whether deployments can skip
	// GitHub authentication. Set to true only for local development.
	// Production should keep this false to require GitHub App authentication.
	AllowUnauthenticatedDeployments bool `toml:"allow_unauthenticated_deployments"`
}

// HeartbeatConfig holds Checkly heartbeat URLs for health monitoring.
type HeartbeatConfig struct {
	// CertRenewalURL is the Checkly heartbeat URL for certificate renewal.
	// When set, a heartbeat is sent after successful certificate renewal runs.
	// Optional - if empty, no heartbeat is sent.
	CertRenewalURL string `toml:"cert_renewal_url"`

	// QuotaCheckURL is the Checkly heartbeat URL for quota checks.
	// When set, a heartbeat is sent after successful quota check runs.
	// Optional - if empty, no heartbeat is sent.
	QuotaCheckURL string `toml:"quota_check_url"`

	// KeyRefillURL is the Checkly heartbeat URL for key refill runs.
	// When set, a heartbeat is sent after successful key refill runs.
	// Optional - if empty, no heartbeat is sent.
	KeyRefillURL string `toml:"key_refill_url"`
}

// SlackConfig holds Slack webhook URLs for notifications.
type SlackConfig struct {
	// QuotaCheckWebhookURL is the Slack webhook URL for quota exceeded notifications.
	// When set, Slack notifications are sent when workspaces exceed their quota.
	// Optional - if empty, no Slack notifications are sent.
	QuotaCheckWebhookURL string `toml:"quota_check_webhook_url"`
}

// Config holds the complete configuration for the Restate worker service.
// It is designed to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[worker.Config]("/etc/unkey/unkey.toml")
//
// Environment variables are expanded in file values using ${VAR} or
// ${VAR:-default} syntax before parsing. Struct tag defaults are applied to
// any field left at its zero value after parsing, and validation runs
// automatically via [Config.Validate].
//
// Clock is runtime-only and cannot be set through a config file. It is
// tagged toml:"-" and must be set programmatically after loading.
type Config struct {
	// InstanceID is the unique identifier for this worker instance.
	// Used for logging, tracing, and cluster coordination.
	InstanceID string `toml:"instance_id"`

	// Region is the geographic region where this worker instance is running.
	// Used for logging and tracing context.
	Region string `toml:"region"`

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int `toml:"prometheus_port"`

	// DefaultDomain is the fallback domain for system operations.
	// Used for sentinel deployment and automatic certificate bootstrapping.
	DefaultDomain string `toml:"default_domain" config:"default=unkey.app"`

	// BuildPlatformStr defines the target architecture for container builds.
	// Format: "linux/amd64", "linux/arm64". Only "linux" OS supported.
	BuildPlatformStr string `toml:"build_platform" config:"default=linux/amd64"`

	// SentinelImage is the container image used for new sentinel deployments.
	// Overrides default sentinel image with custom build or registry.
	SentinelImage string `toml:"sentinel_image" config:"default=ghcr.io/unkeyed/unkey:local"`

	// AvailableRegions is a list of available regions for deployments.
	// typically in the format "region.provider", ie "us-east-1.aws", "local.dev"
	AvailableRegions []string `toml:"available_regions"`

	// CnameDomain is the base domain for custom domain CNAME targets.
	// Each custom domain gets a unique subdomain like "{random}.{CnameDomain}".
	CnameDomain string `toml:"cname_domain" config:"required,nonempty"`

	// Database configures MySQL connections. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	// Vault configures the encryption/decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Otel configures OpenTelemetry export. See [config.OtelConfig].
	Otel config.OtelConfig `toml:"otel"`

	// Logging configures log sampling. See [config.LoggingConfig].
	Logging config.LoggingConfig `toml:"logging"`

	// Acme configures automatic TLS certificate management.
	// Enables Let's Encrypt integration for domain certificates.
	Acme AcmeConfig `toml:"acme"`

	// Restate configures workflow engine integration.
	// Enables asynchronous deployment and certificate renewal workflows.
	Restate RestateConfig `toml:"restate"`

	// Depot configures Depot.dev build service integration.
	Depot DepotConfig `toml:"depot"`

	// Registry configures container registry authentication.
	Registry RegistryConfig `toml:"registry"`

	// ClickHouse configures ClickHouse connections.
	ClickHouse ClickHouseConfig `toml:"clickhouse"`

	// GitHub configures GitHub App integration for webhook-triggered deployments.
	GitHub *GitHubConfig `toml:"github"`

	// Heartbeat configures Checkly heartbeat URLs for health monitoring.
	Heartbeat HeartbeatConfig `toml:"heartbeat"`

	// Slack configures Slack webhook URLs for notifications.
	Slack SlackConfig `toml:"slack"`

	// Clock provides time operations for testing and scheduling.
	// Use clock.New() for production deployments.
	Clock clock.Clock `toml:"-"`
}

// parseBuildPlatform validates and parses a build platform string.
//
// This function validates that the build platform follows the expected
// format "linux/{architecture}" and parses it into components.
// Only "linux" OS is currently supported.
//
// Returns BuildPlatform with parsed components or error if format is invalid
// or OS is not supported.
func parseBuildPlatform(buildPlatform string) (BuildPlatform, error) {
	buildPlatform = strings.TrimPrefix(buildPlatform, "/")
	parts := strings.Split(buildPlatform, "/")

	if err := assert.All(
		assert.Equal(len(parts), 2, fmt.Sprintf("invalid build platform format: %s (expected format: linux/amd64)", buildPlatform)),
		assert.Equal(parts[0], "linux", fmt.Sprintf("unsupported OS: %s (only linux is supported)", parts[0])),
	); err != nil {
		return BuildPlatform{}, err
	}

	return BuildPlatform{
		Platform:     buildPlatform,
		Architecture: parts[1],
	}, nil
}

// GetBuildPlatform returns the parsed build platform.
//
// This method returns the parsed BuildPlatform from the configured
// BuildPlatformStr string.
//
// Returns BuildPlatform with parsed platform and architecture components,
// or an error if the platform string is invalid.
func (c Config) GetBuildPlatform() (BuildPlatform, error) {
	return parseBuildPlatform(c.BuildPlatformStr)
}

// GetRegistryConfig returns the registry configuration.
//
// This method returns the RegistryConfig from the main Config struct.
// Should only be called after Validate() succeeds to ensure all required
// fields are present.
//
// Returns RegistryConfig with URL, username, and password for container registry access.
func (c Config) GetRegistryConfig() RegistryConfig {
	return c.Registry
}

// GetDepotConfig returns the depot configuration.
//
// This method returns the DepotConfig from the main Config struct.
// Should only be called after Validate() succeeds to ensure
// depot configuration is complete and valid.
//
// Returns the DepotConfig containing API URL and project region.
func (c Config) GetDepotConfig() DepotConfig {
	return c.Depot
}

// Validate checks the configuration for required fields and logical consistency.
//
// This method performs comprehensive validation of all configuration sections
// including build backend, ACME providers, database connections, and
// required credentials. It ensures that conditional configuration
// (like ACME providers) has all necessary dependencies.
//
// Returns an error if required fields are missing, invalid, or inconsistent.
// Provides detailed error messages to help identify configuration issues.
func (c *Config) Validate() error {
	// Validate Route53 configuration if enabled
	if c.Acme.Enabled && c.Acme.Route53.Enabled {
		if err := assert.All(
			assert.NotEmpty(c.Acme.Route53.AccessKeyID, "route53 access key ID is required when route53 is enabled"),
			assert.NotEmpty(c.Acme.Route53.SecretAccessKey, "route53 secret access key is required when route53 is enabled"),
			assert.NotEmpty(c.Acme.Route53.Region, "route53 region is required when route53 is enabled"),
		); err != nil {
			return err
		}
	}

	// Validate build platform format (only if configured)
	if c.BuildPlatformStr != "" {
		if _, err := parseBuildPlatform(c.BuildPlatformStr); err != nil {
			return err
		}
	}

	// Validate build configuration (Depot backend) - only if registry password is provided
	// The registry password is the depot token, which is required for builds.
	// URL and username may be hardcoded in k8s manifests, password comes from secrets.
	if c.Registry.Password != "" {
		if err := assert.All(
			assert.NotEmpty(c.Registry.URL, "registry URL is required when registry password is configured"),
			assert.NotEmpty(c.Registry.Username, "registry username is required when registry password is configured"),
			assert.NotEmpty(c.Depot.APIUrl, "Depot API URL is required when registry password is configured"),
			assert.NotEmpty(c.Depot.ProjectRegion, "Depot project region is required when registry password is configured"),
		); err != nil {
			return err
		}
	}

	return nil
}
