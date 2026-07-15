package heimdall

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_Validate_CollectorsDefault(t *testing.T) {
	t.Parallel()

	cfg := &Config{Collectors: nil}
	require.NoError(t, cfg.Validate())
	require.Equal(t, []string{"cpu", "memory", "disk", "network"}, cfg.Collectors)
}

func TestConfig_Validate_CollectorsExplicitSubset(t *testing.T) {
	t.Parallel()

	// Operators should be able to roll out one collector at a time. The
	// list as written must round-trip through validation untouched (no
	// reordering, no defaulting on top of an explicit subset).
	cfg := &Config{Collectors: []string{"cpu", "memory"}}
	require.NoError(t, cfg.Validate())
	require.Equal(t, []string{"cpu", "memory"}, cfg.Collectors)
}

func TestConfig_Validate_CollectorsUnknownFails(t *testing.T) {
	t.Parallel()

	// Typo guard. "ram" is plausible-looking but not real; the validator
	// must surface it at startup, not silently no-op into "all enabled".
	cfg := &Config{Collectors: []string{"cpu", "ram"}}
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "ram")
	require.Contains(t, strings.ToLower(err.Error()), "unknown")
}
