package undns

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
)

func TestConfigBounds(t *testing.T) {
	for _, setting := range []string{
		"ttl_seconds = 61", "queries_in_flight = -1", "queries_in_flight = 65537",
		`watch_timeout = "999ms"`, `watch_timeout = "61s"`, `forward_timeout = "-1ns"`, `forward_timeout = "11s"`,
		"forwards_in_flight = -1", "forwards_per_workspace = -1", "forwards_in_flight = 8\nforwards_per_workspace = 9",
		"forwards_unidentified = -1", "forwards_unidentified = 65537",
		"forwards_in_flight = 8\nforwards_per_workspace = 8\nforwards_unidentified = 9",
	} {
		t.Run(setting, func(t *testing.T) {
			_, err := config.LoadBytes[Config]([]byte("upstream = '10.96.0.10:53'\n" + setting))
			require.Error(t, err)
		})
	}

	_, err := config.LoadBytes[Config]([]byte("upstream = '10.96.0.10:53'\nttl_seconds = 60\nqueries_in_flight = 65536\nwatch_timeout = '1s'\nforward_timeout = '10s'"))
	require.NoError(t, err)
	cfg, err := config.LoadBytes[Config]([]byte("upstream = '10.96.0.10:53'\nforwards_in_flight = 8\nforwards_per_workspace = 8\nforwards_unidentified = 8"))
	require.NoError(t, err)
	require.Equal(t, 8, cfg.ForwardsUnidentified)
}
