package preflight

import (
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

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

	// --- Logging sampler configuration ---

	// LogSampleRate is the baseline probability (0.0-1.0) of emitting log events.
	LogSampleRate float64

	// LogSlowThreshold defines what duration qualifies as "slow" for sampling.
	LogSlowThreshold time.Duration
}

func (c *Config) Validate() error {
	if c.HttpPort == 0 {
		c.HttpPort = 8443
	}

	return assert.True(validImagePullPolicies[c.InjectImagePullPolicy], "inject-image-pull-policy must be one of: Always, IfNotPresent, Never")
}
