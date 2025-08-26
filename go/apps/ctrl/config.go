package ctrl

import (
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

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
	DatabaseHydra     string

	// --- OpenTelemetry configuration ---

	// Enable sending otel data to the collector endpoint for metrics, traces, and logs
	OtelEnabled           bool
	OtelTraceSamplingRate float64

	// TLSConfig contains the TLS configuration for HTTPS
	TLSConfig *tls.Config

	// AuthToken is the authentication token for control plane API access
	AuthToken string

	// MetaldAddress is the full URL of the metald service for VM operations (e.g., "https://metald.example.com:8080")
	MetaldAddress string

	// SPIFFESocketPath is the path to the SPIFFE agent socket for mTLS authentication
	SPIFFESocketPath string

	Clock clock.Clock

	// --- Vault Configuration ---
	VaultMasterKeys []string
	VaultS3         *storage.S3Config
}

func (c Config) Validate() error {
	if c.VaultS3 != nil {
		err := assert.All(
			assert.NotEmpty(c.VaultS3.S3URL, "vault s3 url is empty"),
			assert.NotEmpty(c.VaultS3.S3Bucket, "vault s3 bucket is empty"),
			assert.NotEmpty(c.VaultS3.S3AccessKeyID, "vault s3 access key id is empty"),
			assert.NotEmpty(c.VaultS3.S3AccessKeySecret, "vault s3 secret access key is empty"),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
