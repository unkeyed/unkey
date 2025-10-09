package ctrl

import (
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
}

type CloudflareConfig struct {
	// Enables DNS-01 challenges using Cloudflare
	Enabled bool

	// ApiToken is the Cloudflare API token with Zone:Read, DNS:Edit permissions
	ApiToken string
}

type AcmeConfig struct {
	// Enables ACME challenges for TLS certificates
	Enabled bool

	// Enables DNS-01 challenges using Cloudflare
	Cloudflare CloudflareConfig
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
	// RegistryUrl is the URL of the Depot container registry (e.g., "registry.depot.dev")
	RegistryUrl string
	// Username is the registry username for authentication (typically "x-token" for token-based auth)
	Username string
	// AccessToken is the Depot API access token for authentication
	AccessToken string
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

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations
	DatabasePrimary   string
	DatabasePartition string

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
	// TODO: Replace with JWT authentication when moving to private IP
	APIKey string

	// SPIFFESocketPath is the path to the SPIFFE agent socket for mTLS authentication
	SPIFFESocketPath string

	Clock clock.Clock

	// --- Vault Configuration ---
	VaultMasterKeys []string
	VaultS3         S3Config

	// --- ACME/Cloudflare Configuration ---
	Acme AcmeConfig

	DefaultDomain string

	Restate RestateConfig

	// --- Build Storage Configuration ---
	BuildBackend BuildBackend
	BuildS3      S3Config
	Depot        DepotConfig
}

func (c Config) Validate() error {
	// Validate Cloudflare configuration if enabled
	if c.Acme.Enabled && c.Acme.Cloudflare.Enabled {
		if err := assert.NotEmpty(c.Acme.Cloudflare.ApiToken, "cloudflare API token is required when cloudflare is enabled"); err != nil {
			return err
		}
	}

	// Validate build backend configuration
	switch c.BuildBackend {
	case BuildBackendDepot:
		if err := assert.NotEmpty(c.BuildS3.URL, "build S3 URL is required when using Depot backend"); err != nil {
			return err
		}
		if err := assert.NotEmpty(c.BuildS3.Bucket, "build S3 bucket is required when using Depot backend"); err != nil {
			return err
		}
		if err := assert.NotEmpty(c.BuildS3.AccessKeyID, "build S3 access key ID is required when using Depot backend"); err != nil {
			return err
		}
		if err := assert.NotEmpty(c.BuildS3.AccessKeySecret, "build S3 access key secret is required when using Depot backend"); err != nil {
			return err
		}
		if err := assert.NotEmpty(c.Depot.AccessToken, "Depot access token is required when using Depot backend"); err != nil {
			return err
		}
	case BuildBackendDocker:
		if err := assert.NotEmpty(c.BuildS3.URL, "build S3 URL is required when using Docker backend"); err != nil {
			return err
		}
		if err := assert.NotEmpty(c.BuildS3.Bucket, "build S3 bucket is required when using Docker backend"); err != nil {
			return err
		}
		if err := assert.NotEmpty(c.BuildS3.AccessKeyID, "build S3 access key ID is required when using Docker backend"); err != nil {
			return err
		}
		if err := assert.NotEmpty(c.BuildS3.AccessKeySecret, "build S3 access key secret is required when using Docker backend"); err != nil {
			return err
		}
	case "":
		return assert.NotEmpty(string(c.BuildBackend), "build backend must be specified (depot or docker)")
	default:
		return assert.NotEmpty("", "build backend must be either 'depot' or 'docker'")
	}

	return nil
}
