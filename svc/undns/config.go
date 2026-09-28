package undns

import (
	"net/netip"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// Config configures the regional resolver. Upstream must be a regional CoreDNS IP.
type Config struct {
	ListenAddress        string        `toml:"listen_address" config:"default=0.0.0.0:5353"`
	HealthAddress        string        `toml:"health_address" config:"default=0.0.0.0:9090"`
	Upstream             string        `toml:"upstream" config:"required,nonempty"`
	TTLSeconds           uint32        `toml:"ttl_seconds" config:"default=5,min=1,max=60"`
	ForwardTimeout       time.Duration `toml:"forward_timeout" config:"default=2s,min=1,max=10000000000"`
	WatchTimeout         time.Duration `toml:"watch_timeout" config:"default=30s,min=1000000000,max=60000000000"`
	QueriesInFlight      int           `toml:"queries_in_flight" config:"default=256,min=1,max=65536"`
	ForwardsInFlight     int           `toml:"forwards_in_flight" config:"default=1024,min=1,max=65536"`
	ForwardsPerWorkspace int           `toml:"forwards_per_workspace" config:"default=256,min=1,max=65536"`
	ForwardsUnidentified int           `toml:"forwards_unidentified" config:"default=32,min=1,max=65536"`
}

// Validate checks endpoint syntax and listener conflicts beyond the config tags.
func (c Config) Validate() error {
	listen, listenErr := netip.ParseAddrPort(c.ListenAddress)
	health, healthErr := netip.ParseAddrPort(c.HealthAddress)
	upstream, upstreamErr := netip.ParseAddrPort(c.Upstream)

	return assert.All(
		assert.True(listenErr == nil && listen.Port() != 0, "listen_address must be an IP and nonzero port"),
		assert.True(healthErr == nil && health.Port() != 0, "health_address must be an IP and nonzero port"),
		assert.True(upstreamErr == nil && upstream.Port() != 0, "upstream must be an IP and nonzero port"),
		assert.True(upstreamErr == nil && upstream.Addr().IsPrivate(), "upstream must be a private IP"),
		assert.NotEqual(listen, health, "DNS and health listeners must differ"),
		assert.NotEqual(listen, upstream, "DNS listener and upstream must differ"),
		assert.True(c.ForwardsPerWorkspace <= c.ForwardsInFlight, "forwards_per_workspace must not exceed forwards_in_flight"),
		assert.True(c.ForwardsUnidentified <= c.ForwardsInFlight, "forwards_unidentified must not exceed forwards_in_flight"),
	)
}
