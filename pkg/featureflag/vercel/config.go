package vercel

import (
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

const (
	defaultEndpoint     = "https://flags.vercel.com/v1/datafile"
	defaultRefresh      = time.Minute
	defaultTimeout      = 10 * time.Second
	defaultMaxStaleness = 10 * time.Minute
	maxBodyBytes        = 4 << 20
	minRefresh          = time.Second
	maxRefresh          = time.Hour
	maxTimeout          = time.Minute
	maxStalenessCeiling = 24 * time.Hour
)

// Config configures a Vercel datafile provider. Zero durations use the
// defaults: a 1m refresh interval, a 10s HTTP timeout, and 10m of maximum
// staleness.
type Config struct {
	// SDKKey is the bare server SDK key for one Vercel environment, starting
	// with vf_server_. It isn't the FLAGS connection string the dashboard reads.
	SDKKey          string        `toml:"sdk_key"`
	RefreshInterval time.Duration `toml:"refresh_interval"`
	HTTPTimeout     time.Duration `toml:"http_timeout"`
	MaxStaleness    time.Duration `toml:"max_staleness"`
}

func (c Config) withDefaults() Config {
	if c.RefreshInterval == 0 {
		c.RefreshInterval = defaultRefresh
	}
	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = defaultTimeout
	}
	if c.MaxStaleness == 0 {
		c.MaxStaleness = defaultMaxStaleness
	}
	return c
}

// Validate checks config after applying defaults.
func (c Config) Validate() error {
	c = c.withDefaults()
	return assert.All(
		assert.True(strings.HasPrefix(c.SDKKey, "vf_server_"), "vercel feature flags SDK key must start with vf_server_"),
		assert.GreaterOrEqual(c.RefreshInterval, minRefresh, "refresh interval must be at least 1s"),
		assert.LessOrEqual(c.RefreshInterval, maxRefresh, "refresh interval must be at most 1h"),
		assert.Greater(c.HTTPTimeout, 0, "HTTP timeout must be positive"),
		assert.LessOrEqual(c.HTTPTimeout, maxTimeout, "HTTP timeout must be at most 1m"),
		assert.GreaterOrEqual(c.MaxStaleness, c.RefreshInterval, "maximum staleness must be at least the refresh interval"),
		assert.LessOrEqual(c.MaxStaleness, maxStalenessCeiling, "maximum staleness must be at most 24h"),
	)
}
