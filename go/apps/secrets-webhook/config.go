package webhook

import (
	"github.com/unkeyed/unkey/go/pkg/assert"
)

// Config holds configuration for the secrets webhook server.
type Config struct {
	// HttpPort is the port for the webhook server to listen on.
	// Default is 8443. Must use HTTPS for admission webhooks.
	HttpPort int

	// TLSCertFile is the path to the TLS certificate file.
	// Required - K8s admission webhooks must use HTTPS.
	TLSCertFile string

	// TLSKeyFile is the path to the TLS private key file.
	// Required - K8s admission webhooks must use HTTPS.
	TLSKeyFile string

	// UnkeyEnvImage is the container image for the unkey-env binary.
	// Example: "registry.unkey.dev/unkey-env:latest"
	UnkeyEnvImage string

	// KraneEndpoint is the endpoint for the Krane secrets service.
	// Example: "http://krane.unkey.svc.cluster.local:7070"
	KraneEndpoint string

	// AnnotationPrefix is the prefix for pod annotations that control injection.
	// Default: "unkey.com"
	AnnotationPrefix string
}

// Validate checks the configuration for required fields and sets defaults.
func (c *Config) Validate() error {
	// Set defaults
	if c.HttpPort == 0 {
		c.HttpPort = 8443
	}
	if c.AnnotationPrefix == "" {
		c.AnnotationPrefix = "unkey.com"
	}

	// Validate required fields
	return assert.All(
		assert.NotEmpty(c.TLSCertFile, "tls-cert-file is required (K8s webhooks must use HTTPS)"),
		assert.NotEmpty(c.TLSKeyFile, "tls-key-file is required (K8s webhooks must use HTTPS)"),
		assert.NotEmpty(c.UnkeyEnvImage, "unkey-env-image is required"),
		assert.NotEmpty(c.KraneEndpoint, "krane-endpoint is required"),
	)
}
