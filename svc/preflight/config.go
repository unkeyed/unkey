package preflight

import "github.com/unkeyed/unkey/pkg/assert"

type Config struct {
	HttpPort              int
	TLSCertFile           string
	TLSKeyFile            string
	InjectImage           string
	InjectImagePullPolicy string
	KraneEndpoint         string
	DepotToken            string
}

func (c *Config) Validate() error {
	if c.HttpPort == 0 {
		c.HttpPort = 8443
	}

	return assert.All(
		assert.NotEmpty(c.TLSCertFile, "tls-cert-file is required"),
		assert.NotEmpty(c.TLSKeyFile, "tls-key-file is required"),
		assert.NotEmpty(c.InjectImage, "inject-image is required"),
		assert.NotEmpty(c.KraneEndpoint, "krane-endpoint is required"),
	)
}
