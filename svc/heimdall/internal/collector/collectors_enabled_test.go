package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectorSetFrom_AllFour(t *testing.T) {
	t.Parallel()
	cs := CollectorSetFrom([]string{"cpu", "memory", "disk", "network"})
	require.True(t, cs.CPU)
	require.True(t, cs.Memory)
	require.True(t, cs.Disk)
	require.True(t, cs.Network)
}

func TestCollectorSetFrom_Subset(t *testing.T) {
	t.Parallel()
	// Most common rollout: scrape cgroup metrics first, leave the heavier
	// disk + eBPF paths off until they're validated separately.
	cs := CollectorSetFrom([]string{"cpu", "memory"})
	require.True(t, cs.CPU)
	require.True(t, cs.Memory)
	require.False(t, cs.Disk)
	require.False(t, cs.Network)
}

func TestCollectorSetFrom_Empty(t *testing.T) {
	t.Parallel()
	// Empty list = no collector enabled. Defaulting is the Config's job
	// (it fills the list with all four when empty before this is called).
	// Here we just verify the type itself doesn't add any implicit defaults.
	cs := CollectorSetFrom(nil)
	require.False(t, cs.CPU)
	require.False(t, cs.Memory)
	require.False(t, cs.Disk)
	require.False(t, cs.Network)
}

func TestCollectorSetFrom_UnknownIsNoOp(t *testing.T) {
	t.Parallel()
	// Validation upstream should reject unknown names, but defense in
	// depth: an unrecognized name here doesn't crash and doesn't enable
	// anything. ("memory" still flips on so the typo doesn't poison the
	// rest of the list.)
	cs := CollectorSetFrom([]string{"memory", "ram"})
	require.False(t, cs.CPU)
	require.True(t, cs.Memory)
	require.False(t, cs.Disk)
	require.False(t, cs.Network)
}

func TestCollectorSet_Names_PreservesCanonicalOrder(t *testing.T) {
	t.Parallel()
	// Names() output should be stable regardless of input ordering — it's
	// stamped into the per-checkpoint attributes column, and stable order
	// makes diffs across checkpoints meaningful (set membership only).
	a := CollectorSetFrom([]string{"network", "cpu"}).Names()
	b := CollectorSetFrom([]string{"cpu", "network"}).Names()
	require.Equal(t, []string{"cpu", "network"}, a)
	require.Equal(t, a, b)
}
