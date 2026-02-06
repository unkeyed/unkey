---
title: worker
description: "implements the Restate workflow worker for Unkey's control plane"
---

Package worker implements the Restate workflow worker for Unkey's control plane.

The worker is the execution engine for long-running operations in Unkey's infrastructure, handling container builds, deployments, certificate management, and routing configuration through the Restate distributed workflow engine. It provides durable execution guarantees for operations that span multiple services and may take minutes to complete.

### Architecture

The worker acts as a Restate service host, binding workflow services that handle container deployments, TLS certificate management, traffic routing, and versioning. It maintains connections to the primary database for persistent state, vault services for secrets and ACME certificates, S3-compatible storage for build artifacts, and ClickHouse for analytics.

### Configuration

Configuration is provided through \[Config], which validates settings on startup. The worker supports multiple build backends and validates their requirements in \[Config.Validate].

### Usage

The worker is started with \[Run], which blocks until the context is cancelled or a fatal error occurs:

	cfg := worker.Config{
	    InstanceID:      "worker-1",
	    HttpPort:        7092,
	    DatabasePrimary: "mysql://...",
	    // ... additional configuration
	}

	if err := worker.Run(ctx, cfg); err != nil {
	    log.Fatal(err)
	}

### Startup Sequence

\[Run] performs initialization in a specific order: configuration validation, vault services creation, database connection, build storage initialization, ACME provider setup, Restate server binding, admin registration with retry, wildcard certificate bootstrapping, health endpoint startup, and optional Prometheus metrics exposure.

### Graceful Shutdown

When the context passed to \[Run] is cancelled, the worker performs graceful shutdown by stopping the health server, closing database connections, and allowing in-flight Restate workflows to complete. The shutdown sequence is managed through a shutdown handler that reverses the startup order.

### ACME Certificate Management

When ACME is enabled in configuration, the worker automatically manages TLS certificates using Let's Encrypt. It supports HTTP-01 challenges for regular domains and DNS-01 challenges (via Route53) for wildcard certificates. On startup with a configured default domain, \[Run] calls \[bootstrapWildcardDomain] to ensure the platform's wildcard certificate can be automatically renewed.

## Functions

### func Run

```go
func Run(ctx context.Context, cfg Config) error
```

Run starts the Restate worker service with the provided configuration.

This function initializes all required services and starts the Restate server for workflow execution. It performs these major initialization steps:

 1. Validates configuration and initializes structured logging
 2. Creates vault services for secrets and ACME certificates
 3. Initializes database and build storage
 4. Creates build service (docker/depot backend)
 5. Initializes ACME caches and providers (HTTP-01, DNS-01)
 6. Starts Restate server with workflow service bindings
 7. Registers with Restate admin API for service discovery
 8. Starts health check endpoint
 9. Optionally starts Prometheus metrics server

The worker handles graceful shutdown when context is cancelled, properly closing all services and database connections.

Returns an error if configuration validation fails, service initialization fails, or during server startup. Context cancellation results in clean shutdown with nil error.


## Types

### type AcmeConfig

```go
type AcmeConfig struct {
	// Enabled determines whether ACME certificate management is active.
	// When true, certificates are automatically obtained and renewed.
	Enabled bool

	// EmailDomain is the domain used for ACME account emails.
	// Used for Let's Encrypt account registration and recovery.
	// Example: "unkey.com" creates "admin@unkey.com" for ACME account.
	EmailDomain string

	// Route53 configures DNS-01 challenges through AWS Route53 API.
	// Enables wildcard certificates for domains hosted on Route53.
	Route53 Route53Config
}
```

AcmeConfig holds configuration for ACME TLS certificate management.

This configuration enables automatic certificate issuance and renewal through ACME protocol with support for multiple DNS providers.

### type BuildPlatform

```go
type BuildPlatform struct {
	// Platform is the original build platform string.
	// Example: "linux/amd64".
	Platform string

	// Architecture is the CPU architecture component.
	// Example: "amd64", "arm64".
	Architecture string
}
```

BuildPlatform represents parsed container build platform specification.

Contains the validated platform string separated into OS and architecture components for build backend integration.

### type Config

```go
type Config struct {
	// InstanceID is the unique identifier for this worker instance.
	// Used for logging, tracing, and cluster coordination.
	InstanceID string

	// Region is the geographic region where this worker instance is running.
	// Used for logging and tracing context.
	Region string

	// OtelEnabled determines whether OpenTelemetry is enabled.
	// When true, traces and logs are sent to the configured OTLP endpoint.
	OtelEnabled bool

	// OtelTraceSamplingRate controls what percentage of traces are sampled.
	// Values range from 0.0 to 1.0, where 1.0 means all traces are sampled.
	OtelTraceSamplingRate float64

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int

	// DatabasePrimary is the primary database connection string.
	// Used for both read and write operations to persistent storage.
	DatabasePrimary string

	// VaultURL is the URL of the remote vault service for secret encryption.
	// Example: "https://vault.unkey.cloud".
	VaultURL string

	// VaultToken is the authentication token for the remote vault service.
	// Used for bearer authentication when calling vault RPCs.
	VaultToken string

	// Acme configures automatic TLS certificate management.
	// Enables Let's Encrypt integration for domain certificates.
	Acme AcmeConfig

	// DefaultDomain is the fallback domain for system operations.
	// Used for sentinel deployment and automatic certificate bootstrapping.
	DefaultDomain string

	// Restate configures workflow engine integration.
	// Enables asynchronous deployment and certificate renewal workflows.
	Restate RestateConfig

	// BuildPlatform defines the target architecture for container builds.
	// Format: "linux/amd64", "linux/arm64". Only "linux" OS supported.
	BuildPlatform string

	// Depot configures Depot.dev build service integration.
	Depot DepotConfig

	// RegistryURL is the container registry URL for pulling images.
	// Example: "registry.depot.dev" or "https://registry.example.com".
	RegistryURL string

	// RegistryUsername is the username for container registry authentication.
	// Common values: "x-token" for token-based auth or actual username.
	RegistryUsername string

	// RegistryPassword is the password/token for container registry authentication.
	// Should be stored securely (environment variable or secret management).
	RegistryPassword string

	// ClickhouseURL is the ClickHouse database connection string.
	// Used for analytics and operational metrics storage.
	ClickhouseURL string

	// ClickhouseAdminURL is the connection string for the ClickHouse admin user.
	// Used by ClickhouseUserService to create/configure workspace users.
	// The admin user requires limited permissions: CREATE/ALTER/DROP for USER,
	// QUOTA, ROW POLICY, and SETTINGS PROFILE, plus GRANT OPTION on analytics tables.
	// Optional - if not set, ClickhouseUserService will not be enabled.
	// Example: "clickhouse://unkey_user_admin:C57RqT5EPZBqCJkMxN9mEZZEzMPcw9yBlwhIizk99t7kx6uLi9rYmtWObsXzdl@clickhouse:9000/default"
	ClickhouseAdminURL string

	// SentinelImage is the container image used for new sentinel deployments.
	// Overrides default sentinel image with custom build or registry.
	SentinelImage string

	// AvailableRegions is a list of available regions for deployments.
	// typically in the format "region.provider", ie "us-east-1.aws", "local.dev"
	AvailableRegions []string

	// CnameDomain is the base domain for custom domain CNAME targets.
	// Each custom domain gets a unique subdomain like "{random}.{CnameDomain}".
	// For production: "unkey-dns.com"
	// For local: "unkey.local"
	CnameDomain string

	// GitHub configures GitHub App integration for webhook-triggered deployments.
	GitHub GitHubConfig

	// Clock provides time operations for testing and scheduling.
	// Use clock.RealClock{} for production deployments.
	Clock clock.Clock

	// CertRenewalHeartbeatURL is the Checkly heartbeat URL for certificate renewal.
	// When set, a heartbeat is sent after successful certificate renewal runs.
	// Optional - if empty, no heartbeat is sent.
	CertRenewalHeartbeatURL string

	// QuotaCheckHeartbeatURL is the Checkly heartbeat URL for quota checks.
	// When set, a heartbeat is sent after successful quota check runs.
	// Optional - if empty, no heartbeat is sent.
	QuotaCheckHeartbeatURL string

	// QuotaCheckSlackWebhookURL is the Slack webhook URL for quota exceeded notifications.
	// When set, Slack notifications are sent when workspaces exceed their quota.
	// Optional - if empty, no Slack notifications are sent.
	QuotaCheckSlackWebhookURL string
}
```

Config holds configuration for the Restate worker service.

This comprehensive configuration structure defines all aspects of worker operation including database connections, vault integration, build backends, ACME certificate management, and Restate integration.

#### func (Config) GetBuildPlatform

```go
func (c Config) GetBuildPlatform() BuildPlatform
```

GetBuildPlatform returns the parsed build platform.

This method returns the parsed BuildPlatform from the configured BuildPlatform string. Should only be called after Validate() succeeds to ensure the platform string is valid.

Returns BuildPlatform with parsed platform and architecture components.

#### func (Config) GetDepotConfig

```go
func (c Config) GetDepotConfig() DepotConfig
```

GetDepotConfig returns the depot configuration.

This method returns the DepotConfig from the main Config struct. Should only be called after Validate() succeeds to ensure depot configuration is complete and valid.

Returns the DepotConfig containing API URL and project region.

#### func (Config) GetRegistryConfig

```go
func (c Config) GetRegistryConfig() RegistryConfig
```

GetRegistryConfig returns the registry configuration.

This method builds a RegistryConfig from the individual registry settings in the main Config struct. Should only be called after Validate() succeeds to ensure all required fields are present.

Returns RegistryConfig with URL, username, and password for container registry access.

#### func (Config) Validate

```go
func (c Config) Validate() error
```

Validate checks the configuration for required fields and logical consistency.

This method performs comprehensive validation of all configuration sections including build backend, ACME providers, database connections, and required credentials. It ensures that conditional configuration (like ACME providers) has all necessary dependencies.

Returns an error if required fields are missing, invalid, or inconsistent. Provides detailed error messages to help identify configuration issues.

### type DepotConfig

```go
type DepotConfig struct {
	// APIUrl is the Depot API endpoint URL for build operations.
	// Example: "https://api.depot.dev".
	APIUrl string

	// ProjectRegion is the geographic region for build storage.
	// Affects build performance and data residency.
	// Options: "us-east-1", "eu-central-1". Default: "us-east-1".
	ProjectRegion string
}
```

DepotConfig holds configuration for Depot.dev build service integration.

This configuration enables cloud-native container builds through Depot's managed build infrastructure with optimized caching.

### type GitHubConfig

```go
type GitHubConfig struct {
	// AppID is the GitHub App ID for authentication.
	AppID int64

	// PrivateKeyPEM is the GitHub App private key in PEM format.
	PrivateKeyPEM string
}
```

GitHubConfig holds configuration for GitHub App integration.

#### func (GitHubConfig) Enabled

```go
func (c GitHubConfig) Enabled() bool
```

Enabled returns true only if ALL required GitHub App fields are configured. This ensures we never register the workflow with partial/insecure config.

### type RegistryConfig

```go
type RegistryConfig struct {
	// URL is the container registry endpoint URL.
	// Example: "registry.depot.dev" or "https://registry.example.com".
	URL string

	// Username is the registry authentication username.
	// Common values: "x-token" for token-based auth, or actual username.
	Username string

	// Password is the registry password or authentication token.
	// Should be stored securely and rotated regularly.
	Password string
}
```

RegistryConfig holds container registry authentication configuration.

This configuration provides credentials for accessing container registries used by build backends for pushing and pulling images.

### type RestateConfig

```go
type RestateConfig struct {
	// AdminURL is the Restate admin endpoint URL for service registration.
	// Used by the worker to register its workflow services.
	// Example: "http://restate:9070".
	AdminURL string

	// APIKey is the optional authentication key for Restate admin API requests.
	// If set, this key will be sent with all requests to the Restate admin API.
	APIKey string

	// HttpPort is the port where the worker listens for Restate requests.
	// This is the internal Restate server port, not the health check port.
	HttpPort int

	// RegisterAs is the service URL used for self-registration with Restate.
	// Allows Restate to discover and invoke this worker's services.
	// Example: "http://worker:9080".
	RegisterAs string
}
```

RestateConfig holds configuration for Restate workflow engine integration.

This configuration enables asynchronous workflow execution through the Restate distributed system for deployment and certificate operations.

### type Route53Config

```go
type Route53Config struct {
	// Enabled determines whether Route53 DNS-01 challenges are used.
	// When true, wildcard certificates can be automatically obtained.
	Enabled bool

	// AccessKeyID is the AWS access key ID for Route53 API access.
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key for Route53 API access.
	SecretAccessKey string

	// Region is the AWS region where Route53 hosted zones are located.
	// Example: "us-east-1", "us-west-2".
	Region string

	// HostedZoneID overrides automatic zone discovery.
	// Required when domains have complex CNAME setups that confuse
	// automatic zone lookup (e.g., wildcard CNAMEs to load balancers).
	HostedZoneID string
}
```

Route53Config holds AWS Route53 configuration for ACME DNS-01 challenges.

This configuration enables automatic DNS record creation for wildcard TLS certificates through AWS Route53 DNS API.

