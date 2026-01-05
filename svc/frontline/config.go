package frontline

import (
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/vault/storage"
)

type Config struct {
	// FrontlineID is the unique identifier for this instance of the Frontline server
	FrontlineID string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the Gate server to listen on (default: 7070)
	HttpPort int

	// HttpsPort defines the HTTPS port for the Gate server to listen on (default: 7443)
	HttpsPort int

	// Region identifies the geographic region where this node is deployed
	Region string

	// EnableTLS specifies whether TLS should be enabled for the Frontline server
	EnableTLS bool

	// BaseDomain is the base domain for region routing (e.g., unkey.cloud)
	// Cross-region requests are forwarded to {region}.{BaseDomain}
	BaseDomain string

	// MaxHops is the maximum number of frontline hops allowed before rejecting the request
	// This prevents infinite routing loops. Default: 3
	MaxHops int

	// -- Control Plane Configuration ---

	// CtrlAddr is the address of the control plane (e.g., control.unkey.com)
	CtrlAddr string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations
	DatabasePrimary string

	// DatabaseReadonlyReplica is an optional read-replica database connection string for read operations
	DatabaseReadonlyReplica string

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
