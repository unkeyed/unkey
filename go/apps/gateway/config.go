package gateway

import (
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

type Config struct {
	// GatewayID is the unique identifier for this instance of the Gateway server
	GatewayID string

	// DeploymentID is the deployment this gateway belongs to (e.g., d-abc123)
	DeploymentID string

	// WorkspaceID is the workspace this gateway belongs to
	WorkspaceID string

	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the Gateway server to listen on (default: 8080)
	HttpPort int

	// HttpsPort defines the HTTPS port for the Gateway server to listen on (default: 8443)
	HttpsPort int

	// Region identifies the geographic region where this node is deployed
	Region string

	// EnableTLS specifies whether TLS should be enabled for the Gateway server
	// In production, gateways typically use mTLS with SPIRE for internal communication
	EnableTLS bool

	// CtrlAddr is the address for the control plane to connect to
	CtrlAddr string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations (partition_001)
	DatabasePrimary string

	// DatabaseReadonlyReplica is an optional read-replica database connection string for read operations (partition_001)
	DatabaseReadonlyReplica string

	// --- Key Service Database configuration ---

	// MainDatabasePrimary is the primary database connection string for the keys service (non-partitioned)
	MainDatabasePrimary string

	// MainDatabaseReadonlyReplica is an optional read-replica database connection string for the keys service
	MainDatabaseReadonlyReplica string

	// --- ClickHouse configuration ---

	// ClickhouseURL is the ClickHouse database connection string
	ClickhouseURL string

	// RedisURL is the Redis connection string for the keys service
	RedisURL string

	// --- SPIRE/mTLS configuration ---

	// SpireSocketPath is the path to the SPIRE agent socket for workload attestation
	// Default: /run/spire/sockets/agent.sock
	SpireSocketPath string

	// EnableSpire enables SPIRE-based mTLS for ingress communication
	EnableSpire bool

	// --- OpenTelemetry configuration ---

	// OtelEnabled specifies whether OpenTelemetry tracing is enabled
	OtelEnabled bool

	// OtelTraceSamplingRate specifies the sampling rate for OpenTelemetry traces (0.0 - 1.0)
	OtelTraceSamplingRate float64

	// PrometheusPort specifies the port for Prometheus metrics
	PrometheusPort int

	// --- Vault Configuration ---
	VaultMasterKeys []string
	VaultS3         *storage.S3Config
}

func (c Config) Validate() error {
	err := assert.All(
		assert.NotEmpty(c.DeploymentID, "deployment id is required"),
		assert.NotEmpty(c.WorkspaceID, "workspace id is required"),
	)
	if err != nil {
		return err
	}

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
