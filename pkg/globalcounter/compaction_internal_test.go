package globalcounter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLockLimits(t *testing.T) {
	for _, tt := range []struct {
		name      string
		remaining int
		ready     int32
		parked    int32
		ok        bool
	}{
		{name: "a whole transaction", remaining: compactionRowsPerTxMax, ready: compactionRowsPerTxMax - compactionParkedRowsMax, parked: compactionParkedRowsMax, ok: true},
		{name: "room for one ready row", remaining: compactionParkedRowsMax + 1, ready: 1, parked: compactionParkedRowsMax, ok: true},
		{name: "only the parked share is left", remaining: compactionParkedRowsMax, ok: false},
		{name: "nothing left", remaining: 0, ok: false},
		{name: "overdrawn", remaining: -3, ok: false},
	} {
		ready, parked, ok := lockLimits(tt.remaining)
		require.Equal(t, tt.ok, ok, tt.name)
		require.Equal(t, tt.ready, ready, tt.name)
		require.Equal(t, tt.parked, parked, tt.name)
		if ok {
			require.LessOrEqual(t, int(ready+parked), tt.remaining, tt.name+": the limits never exceed the budget")
		}
	}
}

func TestCompactionWaitMs(t *testing.T) {
	const minute = int64(60_000)
	for _, tt := range []struct {
		level uint8
		want  int64
	}{
		{level: 1, want: 20 * minute},
		{level: 2, want: 40 * minute},
		{level: 5, want: 320 * minute},
		{level: 10, want: 10 * minute << 10},
		{level: 15, want: 10 * minute << 15},
		{level: 16, want: 365 * 24 * 60 * minute},
		{level: 100, want: 365 * 24 * 60 * minute},
		{level: 255, want: 365 * 24 * 60 * minute},
	} {
		require.Equal(t, tt.want, compactionWaitMs(tt.level), "level %d", tt.level)
	}
}
