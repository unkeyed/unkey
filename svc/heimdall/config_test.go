package heimdall

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
)

func TestConfig_Validate_CollectorsDefault(t *testing.T) {
	t.Parallel()

	cfg := &Config{CollectionInterval: time.Second, Collectors: nil}
	require.NoError(t, cfg.Validate())
	require.Equal(t, []string{"cpu", "memory", "disk", "network"}, cfg.Collectors)
}

func TestConfig_Validate_CollectorsExplicitSubset(t *testing.T) {
	t.Parallel()

	// Operators should be able to roll out one collector at a time. The
	// list as written must round-trip through validation untouched (no
	// reordering, no defaulting on top of an explicit subset).
	cfg := &Config{CollectionInterval: time.Second, Collectors: []string{"cpu", "memory"}}
	require.NoError(t, cfg.Validate())
	require.Equal(t, []string{"cpu", "memory"}, cfg.Collectors)
}

func TestConfig_Validate_CollectorsUnknownFails(t *testing.T) {
	t.Parallel()

	// Typo guard. "ram" is plausible-looking but not real; the validator
	// must surface it at startup, not silently no-op into "all enabled".
	cfg := &Config{CollectionInterval: time.Second, Collectors: []string{"cpu", "ram"}}
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "ram")
	require.Contains(t, strings.ToLower(err.Error()), "unknown")
}

func TestConfigCollectionInterval(t *testing.T) {
	for _, interval := range []time.Duration{-time.Second, 0, time.Second} {
		t.Run(interval.String(), func(t *testing.T) {
			cfg, err := config.Load[Config](fmt.Sprintf(`
region = "test"
platform = "test"
clickhouse_url = "http://localhost:8123"
node_name = "test"
collection_interval = %q
`, interval.String()))
			if interval < 0 {
				require.ErrorContains(t, err, "collection_interval must be positive")
				return
			}
			require.NoError(t, err)
			if interval == 0 {
				require.Equal(t, 5*time.Second, cfg.CollectionInterval)
			} else {
				require.Equal(t, interval, cfg.CollectionInterval)
			}
		})
	}
}
