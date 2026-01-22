package api

import (
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

// VaultConfig holds configuration for HashiCorp Vault integration.
//
// Vault is used for secret management and encryption key storage.
// The control plane uses Vault to securely store and retrieve
// sensitive configuration data for deployed applications.
type VaultConfig struct {
	// Url is the Vault server address including protocol.
	// Example: "https://vault.example.com:8200".
	Url string

	// Token is the Vault authentication token.
	// Must have appropriate policies for secret operations.
	Token string
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

	// Image specifies the container image identifier including repository and tag.
	// Used for control plane deployment and sentinel image configuration.
	Image string

	// Region is the geographic region where this control plane instance runs.
	// Used for logging, tracing, and region-aware routing decisions.
	Region string

	// HttpPort defines the HTTP port for the control plane server.
	// Default: 8080. Cannot be 0.
	HttpPort int

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int

	// DatabasePrimary is the primary database connection string.
	// Used for both read and write operations to persistent storage.
	DatabasePrimary string

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

	// Clock provides time operations for testing and scheduling.
	// Use clock.RealClock{} for production deployments.
	Clock clock.Clock

	// Vault configures HashiCorp Vault integration for secret management.
	Vault VaultConfig

	// Restate configures workflow engine integration.
	// Enables asynchronous deployment and certificate renewal workflows.
	Restate RestateConfig

	// BuildS3 configures storage for build artifacts and outputs.
	// Used by both Depot and Docker build backends.
	BuildS3 S3Config

	// SentinelImage is the container image used for new sentinel deployments.
	// Overrides default sentinel image with custom build or registry.
	SentinelImage string

	// AvailableRegions is a list of available regions for deployments.
	// typically in the format "region.provider", ie "us-east-1.aws", "local.dev"
	AvailableRegions []string
}

// Validate checks the configuration for required fields and logical consistency.
//
// Currently this method performs no validation and always returns nil. Future
// implementations should validate required fields like DatabasePrimary, HttpPort,
// and conditional dependencies between configuration sections.
func (c Config) Validate() error {
	return nil
}
