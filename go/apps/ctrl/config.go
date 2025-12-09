package ctrl

import (
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/tls"
)

type BuildBackend string

const (
	BuildBackendDepot  BuildBackend = "depot"
	BuildBackendDocker BuildBackend = "docker"
)

type S3Config struct {
	URL             string
	Bucket          string
	AccessKeyID     string
	AccessKeySecret string
	ExternalURL     string
}

type CloudflareConfig struct {
	// Enables DNS-01 challenges using Cloudflare
	Enabled bool

	// ApiToken is the Cloudflare API token with Zone:Read, DNS:Edit permissions
	ApiToken string
}

type Route53Config struct {
	// Enables DNS-01 challenges using AWS Route53
	Enabled bool

	// AccessKeyID is the AWS access key ID
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key
	SecretAccessKey string

	// Region is the AWS region (e.g., "us-east-1")
	Region string

	// HostedZoneID bypasses zone auto-discovery. Required when domains have CNAMEs
	// that confuse the zone lookup (e.g., wildcard CNAMEs to load balancers).
	HostedZoneID string
}

type AcmeConfig struct {
	// Enables ACME challenges for TLS certificates
	Enabled bool

	// EmailDomain is the domain used for ACME account emails (e.g., "unkey.com")
	EmailDomain string

	// Cloudflare enables DNS-01 challenges using Cloudflare
	Cloudflare CloudflareConfig

	// Route53 enables DNS-01 challenges using AWS Route53
	Route53 Route53Config
}

type RestateConfig struct {
	// RestateIngressURL is the URL of the Restate ingress endpoint for invoking workflows (e.g., "http://restate:8080")
	IngressURL string

	// AdminURL is the URL of the Restate admin endpoint for service registration (e.g., "http://restate:9070")
	AdminURL string

	// RestateHttpPort is the port where the control plane listens for Restate HTTP requests
	HttpPort int

	// RegisterAs is the url of this service, used for self-registration with the Restate platform
	// ie: http://ctrl:9080
	RegisterAs string
}

type DepotConfig struct {
	// APIUrl is the URL of the Depot API endpoint (e.g., "https://api.depot.dev")
	APIUrl string
	// Build data will be stored in the chosen region ("us-east-1","eu-central-1") (default "us-east-1")
	ProjectRegion string
}

type RegistryConfig struct {
	URL      string
	Username string
	Password string
}

type Config struct {
	// InstanceID is the unique identifier for this instance of the control plane server
	InstanceID string

	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the control plane server to listen on (default: 8080)
	HttpPort int

	// Region identifies the geographic region where this node is deployed
	Region string

	// RegistryURL is the URL of the container registry for pulling images.
	// Example: "registry.depot.dev"
	RegistryURL string

	// RegistryUsername is the username for authenticating with the container registry.
	// Example: "x-token", "depot", or any registry-specific username.
	RegistryUsername string

	// RegistryPassword is the password/token for authenticating with the container registry.
	// Should be stored securely (e.g., environment variable).
	RegistryPassword string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations
	DatabasePrimary string

	// --- OpenTelemetry configuration ---

	// Enable sending otel data to the collector endpoint for metrics, traces, and logs
	OtelEnabled           bool
	OtelTraceSamplingRate float64

	// TLSConfig contains the TLS configuration for HTTPS
	TLSConfig *tls.Config

	// AuthToken is the authentication token for control plane API access
	AuthToken string

	// KraneAddress is the full URL of the krane service for deployment operations (e.g., "https://krane.example.com:8080")
	KraneAddress string

	// APIKey is the API key for simple authentication (demo purposes only)
	APIKey string

	// SPIFFESocketPath is the path to the SPIFFE agent socket for mTLS authentication
	SPIFFESocketPath string

	Clock clock.Clock

	// --- Vault Configuration ---
	// VaultMasterKeys are the master encryption keys for the general vault
	VaultMasterKeys []string
	// VaultS3 is used for general secrets (env vars, API keys, etc.)
	VaultS3 S3Config
	// AcmeVaultMasterKeys are the master encryption keys for the ACME vault
	AcmeVaultMasterKeys []string
	// AcmeVaultS3 is used specifically for ACME/Let's Encrypt certificate storage
	AcmeVaultS3 S3Config

	// --- ACME/Cloudflare Configuration ---
	Acme AcmeConfig

	DefaultDomain string

	Restate RestateConfig

	// --- Build Storage Configuration ---
	BuildBackend BuildBackend
	BuildS3      S3Config
	// BuildPlatform defines the target platform for builds (e.g., "linux/amd64", "linux/arm64")
	BuildPlatform string
	Depot         DepotConfig

	// ClickhouseURL is the ClickHouse database connection string
	ClickhouseURL string
}

type BuildPlatform struct {
	Platform     string
	Architecture string
}

// parseBuildPlatform validates and parses a build platform string
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

// GetBuildPlatform returns the parsed build platform
// This should only be called after Validate() has been called successfully
func (c Config) GetBuildPlatform() BuildPlatform {
	parsed, _ := parseBuildPlatform(c.BuildPlatform)
	return parsed
}

// GetRegistryConfig returns the registry configuration
// This should only be called after Validate() has been called successfully
func (c Config) GetRegistryConfig() RegistryConfig {
	return RegistryConfig{
		URL:      c.RegistryURL,
		Username: c.RegistryUsername,
		Password: c.RegistryPassword,
	}
}

// GetDepotConfig returns the depot configuration
// This should only be called after Validate() has been called successfully
func (c Config) GetDepotConfig() DepotConfig {
	return c.Depot
}

func (c Config) Validate() error {
	// Validate Cloudflare configuration if enabled
	if c.Acme.Enabled && c.Acme.Cloudflare.Enabled {
		if err := assert.NotEmpty(c.Acme.Cloudflare.ApiToken, "cloudflare API token is required when cloudflare is enabled"); err != nil {
			return err
		}
	}

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
