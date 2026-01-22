package ctrl

import (
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/tls"
)

// BuildBackend specifies the container image build backend system.
//
// Determines which service will be used for building container images
// from application source code. Each backend has different capabilities
// and integration requirements.
type BuildBackend string

const (
	// BuildBackendDepot uses Depot.dev for container builds.
	// Provides optimized cloud-native builds with caching and
	// integrated registry management.
	BuildBackendDepot BuildBackend = "depot"
	// BuildBackendDocker uses local Docker daemon for builds.
	// Provides on-premises builds with direct Docker integration.
	BuildBackendDocker BuildBackend = "docker"
)

// S3Config holds S3 configuration for storage backends.
//
// This configuration is used by vault, build storage, and other services
// that need to store data in S3-compatible object storage.
type S3Config struct {
	// URL is the S3 endpoint URL including protocol and region.
	// Examples: "https://s3.amazonaws.com" or "https://s3.us-west-2.amazonaws.com".
	URL string

	// Bucket is the S3 bucket name for storing objects.
	// Must exist and be accessible with the provided credentials.
	Bucket string

	// AccessKeyID is the AWS access key ID for S3 authentication.
	// Must have appropriate permissions for bucket operations.
	AccessKeyID string

	// AccessKeySecret is the AWS secret access key for S3 authentication.
	// Should be stored securely and rotated regularly.
	AccessKeySecret string

	// ExternalURL is the public-facing URL for accessing S3 objects.
	// Used when objects need to be accessed from outside the AWS network.
	// Optional - can be empty for internal-only access.
	ExternalURL string
}

// Route53Config holds AWS Route53 configuration for ACME DNS-01 challenges.
//
// This configuration enables automatic DNS record creation for wildcard
// TLS certificates through AWS Route53 DNS API.
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

// AcmeConfig holds configuration for ACME TLS certificate management.
//
// This configuration enables automatic certificate issuance and renewal
// through ACME protocol with Route53 for DNS-01 challenges.
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

// RestateConfig holds configuration for Restate workflow engine integration.
//
// This configuration enables asynchronous workflow execution through
// the Restate distributed system for deployment and certificate operations.
type RestateConfig struct {
	// URL is the Restate ingress endpoint URL for workflow invocation.
	// Used by clients to start and interact with workflow executions.
	// Example: "http://restate:8080".
	URL string

	// AdminURL is the Restate admin endpoint URL for service registration.
	// Used by the control plane to register its workflow services.
	// Example: "http://restate:9070".
	AdminURL string

	// HttpPort is the port where the control plane listens for Restate requests.
	// This is the internal Restate server port, not the main API port.
	HttpPort int

	// RegisterAs is the service URL used for self-registration with Restate.
	// Allows other Restate services to discover and invoke this control plane.
	// Example: "http://ctrl:9080".
	RegisterAs string

	// APIKey is the authentication key for Restate ingress requests.
	// If set, this key will be sent with all requests to the Restate ingress.
	APIKey string
}

// DepotConfig holds configuration for Depot.dev build service integration.
//
// This configuration enables cloud-native container builds through
// Depot's managed build infrastructure with optimized caching.
type DepotConfig struct {
	// APIUrl is the Depot API endpoint URL for build operations.
	// Example: "https://api.depot.dev".
	APIUrl string

	// ProjectRegion is the geographic region for build storage.
	// Affects build performance and data residency.
	// Options: "us-east-1", "eu-central-1". Default: "us-east-1".
	ProjectRegion string
}

// RegistryConfig holds container registry authentication configuration.
//
// This configuration provides credentials for accessing container registries
// used by build backends for pushing and pulling images.
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

// Config holds configuration for the control plane server.
//
// This comprehensive configuration structure defines all aspects of control plane
// operation including database connections, vault integration, build backends,
// ACME certificate management, and service discovery.
type Config struct {
	// InstanceID is the unique identifier for this control plane instance.
	// Used for logging, tracing, and cluster coordination.
	InstanceID string

	// Platform identifies the cloud platform where the node is running.
	// Affects integration with cloud-specific services and monitoring.
	// Examples: "aws", "gcp", "hetzner".
	Platform string

	// Image specifies the container image identifier including repository and tag.
	// Used for control plane deployment and sentinel image configuration.
	Image string

	// HttpPort defines the HTTP port for the control plane server.
	// Default: 8080. Cannot be 0.
	HttpPort int

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int

	// Region identifies the geographic region where this node is deployed.
	// Used for observability, compliance, and service routing.
	Region string

	// RegistryURL is the container registry URL for pulling images.
	// Example: "registry.depot.dev" or "https://registry.example.com".
	RegistryURL string

	// RegistryUsername is the username for container registry authentication.
	// Common values: "x-token" for token-based auth or actual username.
	RegistryUsername string

	// RegistryPassword is the password/token for container registry authentication.
	// Should be stored securely (environment variable or secret management).
	RegistryPassword string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string.
	// Used for both read and write operations to persistent storage.
	DatabasePrimary string

	// --- OpenTelemetry configuration ---

	// OtelEnabled enables sending telemetry data to collector endpoint.
	// When true, enables metrics, traces, and structured logs.
	OtelEnabled bool

	// OtelTraceSamplingRate controls the percentage of traces sampled.
	// Range: 0.0 (no traces) to 1.0 (all traces). Recommended: 0.1.
	OtelTraceSamplingRate float64

	// TLSConfig contains TLS configuration for HTTPS server.
	// When nil, server runs in HTTP mode for development.
	TLSConfig *tls.Config

	// AuthToken is the authentication token for control plane API access.
	// Used by clients and services to authenticate with this control plane.
	AuthToken string

	// SPIFFESocketPath is the path to the SPIFFE agent socket.
	// Enables mTLS authentication with SPIFFE-based identity system.
	SPIFFESocketPath string

	// Clock provides time operations for testing and scheduling.
	// Use clock.RealClock{} for production deployments.
	Clock clock.Clock

	// --- Vault Configuration ---

	// VaultMasterKeys are encryption keys for the general vault service.
	// Used for encrypting/decrypting environment variables, API keys, etc.
	VaultMasterKeys []string

	// VaultS3 configures S3 storage for the general vault.
	// Stores encrypted secrets data with the provided master keys.
	VaultS3 S3Config

	// AcmeVaultMasterKeys are encryption keys for the ACME vault service.
	// Separate vault for TLS certificate storage and ACME account data.
	AcmeVaultMasterKeys []string

	// AcmeVaultS3 configures S3 storage for the ACME vault.
	// Stores encrypted TLS certificates and ACME challenge data.
	AcmeVaultS3 S3Config

	// --- ACME Configuration ---

	// Acme configures automatic TLS certificate management.
	// Enables Let's Encrypt integration for domain certificates.
	Acme AcmeConfig

	// DefaultDomain is the fallback domain for system operations.
	// Used for sentinel deployment and automatic certificate bootstrapping.
	DefaultDomain string

	// RegionalApexDomain is the base domain for cross-region frontline communication.
	// Certs are provisioned for *.{region}.{RegionalApexDomain} for each available region.
	// Example: "unkey.cloud" results in certs for "*.us-west-2.aws.unkey.cloud", etc.
	RegionalApexDomain string

	// Restate configures workflow engine integration.
	// Enables asynchronous deployment and certificate renewal workflows.
	Restate RestateConfig

	// --- Build Configuration ---

	// BuildBackend selects the container build system.
	// Options: BuildBackendDepot or BuildBackendDocker.
	BuildBackend BuildBackend

	// BuildS3 configures storage for build artifacts and outputs.
	// Used by both Depot and Docker build backends.
	BuildS3 S3Config

	// BuildPlatform defines the target architecture for container builds.
	// Format: "linux/amd64", "linux/arm64". Only "linux" OS supported.
	BuildPlatform string

	// Depot configures Depot.dev build service integration.
	// Required when using BuildBackendDepot.
	Depot DepotConfig

	// --- Analytics Configuration ---

	// ClickhouseURL is the ClickHouse database connection string.
	// Used for analytics and operational metrics storage.
	ClickhouseURL string

	// --- Sentinel Configuration ---

	// SentinelImage is the container image used for new sentinel deployments.
	// Overrides default sentinel image with custom build or registry.
	SentinelImage string

	// AvailableRegions is a list of available regions for deployments.
	// typically in the format "region.provider", ie "us-east-1.aws", "local.dev"
	AvailableRegions []string
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
// BuildPlatform string. Should only be called after Validate() succeeds
// to ensure the platform string is valid.
//
// Returns BuildPlatform with parsed platform and architecture components.
func (c Config) GetBuildPlatform() BuildPlatform {
	parsed, _ := parseBuildPlatform(c.BuildPlatform)
	return parsed
}

// GetRegistryConfig returns the registry configuration.
//
// This method builds a RegistryConfig from the individual registry
// settings in the main Config struct. Should only be called after
// Validate() succeeds to ensure all required fields are present.
//
// Returns RegistryConfig with URL, username, and password for container registry access.
func (c Config) GetRegistryConfig() RegistryConfig {
	return RegistryConfig{
		URL:      c.RegistryURL,
		Username: c.RegistryUsername,
		Password: c.RegistryPassword,
	}
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
func (c Config) Validate() error {
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

	if err := assert.NotEmpty(c.ClickhouseURL, "ClickhouseURL is required"); err != nil {
		return err
	}

	// Validate build platform format
	_, platformErr := parseBuildPlatform(c.BuildPlatform)

	// Validate registry configuration
	registryErr := assert.All(
		assert.NotEmpty(c.RegistryURL, "registry URL is required"),
		assert.NotEmpty(c.RegistryUsername, "registry username is required"),
		assert.NotEmpty(c.RegistryPassword, "registry password is required"),
	)

	switch c.BuildBackend {
	case BuildBackendDepot:
		return assert.All(
			platformErr,
			registryErr,
			assert.NotEmpty(c.BuildPlatform, "build platform is required"),
			assert.NotEmpty(c.BuildS3.URL, "build S3 URL is required when using Depot backend"),
			assert.NotEmpty(c.BuildS3.Bucket, "build S3 bucket is required when using Depot backend"),
			assert.NotEmpty(c.BuildS3.AccessKeyID, "build S3 access key ID is required when using Depot backend"),
			assert.NotEmpty(c.BuildS3.AccessKeySecret, "build S3 access key secret is required when using Depot backend"),
			assert.NotEmpty(c.Depot.APIUrl, "Depot API URL is required when using Depot backend"),
			assert.NotEmpty(c.Depot.ProjectRegion, "Depot project region is required when using Depot backend"),
		)
	case BuildBackendDocker:
		return assert.All(
			platformErr,
			assert.NotEmpty(c.BuildPlatform, "build platform is required"),
			assert.NotEmpty(c.BuildS3.URL, "build S3 URL is required when using Docker backend"),
			assert.NotEmpty(c.BuildS3.ExternalURL, "build S3 external URL is required when using Docker backend"),
			assert.NotEmpty(c.BuildS3.Bucket, "build S3 bucket is required when using Docker backend"),
			assert.NotEmpty(c.BuildS3.AccessKeyID, "build S3 access key ID is required when using Docker backend"),
			assert.NotEmpty(c.BuildS3.AccessKeySecret, "build S3 access key secret is required when using Docker backend"),
		)
	default:
		return fmt.Errorf("build backend must be either 'depot' or 'docker', got: %s", c.BuildBackend)
	}
}
