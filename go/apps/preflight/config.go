package preflight

import "github.com/unkeyed/unkey/go/pkg/assert"

type Config struct {
	HttpPort                int
	TLSCertFile             string
	TLSKeyFile              string
	UnkeyEnvImage           string
	UnkeyEnvImagePullPolicy string
	KraneEndpoint           string
	AnnotationPrefix        string
	DepotToken              string
}

func (c *Config) Validate() error {
	if c.HttpPort == 0 {
		c.HttpPort = 8443
	}
	if c.AnnotationPrefix == "" {
		c.AnnotationPrefix = "unkey.com"
	}

	return assert.All(
		assert.NotEmpty(c.TLSCertFile, "tls-cert-file is required"),
		assert.NotEmpty(c.TLSKeyFile, "tls-key-file is required"),
		assert.NotEmpty(c.UnkeyEnvImage, "unkey-env-image is required"),
		assert.NotEmpty(c.KraneEndpoint, "krane-endpoint is required"),
	)
}
