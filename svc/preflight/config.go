package preflight

import "github.com/unkeyed/unkey/pkg/assert"

var validImagePullPolicies = map[string]bool{
	"Always":       true,
	"IfNotPresent": true,
	"Never":        true,
}

type Config struct {
	HttpPort              int
	TLSCertFile           string
	TLSKeyFile            string
	InjectImage           string
	InjectImagePullPolicy string
	KraneEndpoint         string
	DepotToken            string
	InsecureRegistries    []string
	RegistryAliases       []string
}

func (c *Config) Validate() error {
	if c.HttpPort == 0 {
		c.HttpPort = 8443
	}

	return assert.True(validImagePullPolicies[c.InjectImagePullPolicy], "inject-image-pull-policy must be one of: Always, IfNotPresent, Never")
}
