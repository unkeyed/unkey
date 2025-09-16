package gw

import (
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

type Config struct {
	// GatewayID is the unique identifier for this instance of the Gateway server
	GatewayID string

	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the Gateway server to listen on (default: 6060)
	HttpPort int

	// HttpsPort defines the HTTPS port for the Gateway server to listen on (default: 6061)
	HttpsPort int

	// Region identifies the geographic region where this node is deployed
	Region string

	// EnableTLS specifies whether TLS should be enabled for the Gateway server
	EnableTLS bool

	// DefaultCertDomain is the domain to use for fallback TLS certificate
	DefaultCertDomain string

	// MainDomain is the primary domain for the gateway (e.g., gateway.unkey.com)
	// Internal endpoints like /_internal/liveness are only accessible on this domain
	MainDomain string

	// CtrlAddr is the address for the control plane to connect to
	CtrlAddr string

	// --- ClickHouse configuration ---

	// ClickhouseURL is the ClickHouse database connection string
	ClickhouseURL string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations (partition_001)
	DatabasePrimary string

	// DatabaseReadonlyReplica is an optional read-replica database connection string for read operations (partition_001)
	DatabaseReadonlyReplica string

	// --- Key Service Database configuration ---

	// MainDatabasePrimary is the primary database connection string for the keys service

	// MainDatabasePrimary is the primary database connection string for the keys service (non-partitioned)
	MainDatabasePrimary string

	// MainDatabaseReadonlyReplica is an optional read-replica database connection string for the keys service
	MainDatabaseReadonlyReplica string

	// RedisURL is the Redis connection string for the keys service
	RedisURL string

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

	// --- Local Certificate Configuration ---
	// RequireLocalCert specifies whether to generate a local self-signed certificate for *.unkey.local
	RequireLocalCert bool
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
