package globalcounter

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEntry_ApplySignedDeltasAdvanceTheRevision(t *testing.T) {
	deleteAfter := time.Unix(1_700_000_000, 0)
	now := deleteAfter.Add(-time.Minute)
	var e entry
	var err error
	for i, delta := range []int64{30, -44, 0} {
		e, err = e.apply(delta, deleteAfter, now)
		require.NoError(t, err)
		require.Equal(t, uint64(i+1), e.revision, "even a zero delta is a new snapshot")
	}
	require.Equal(t, entry{count: -14, revision: 3, deleteAfter: deleteAfter, touchedAt: now}, e)
}

func TestEntry_ApplyReplacesDeletionTimeWithoutResettingCount(t *testing.T) {
	first := time.Unix(1_700_000_000, 0)
	later := first.Add(time.Hour)
	var e entry
	for i, deleteAfter := range []time.Time{first, later, first, {}, later} {
		var err error
		e, err = e.apply(3, deleteAfter, first)
		require.NoError(t, err)
		require.Equal(t, entry{count: int64(3 * (i + 1)), revision: uint64(i + 1), deleteAfter: deleteAfter, touchedAt: first}, e)
	}
}

func TestEntry_ForgettableAtWaitsForTheLaterOfDeletionAndLastChange(t *testing.T) {
	deleteAfter := time.Unix(1_700_000_000, 0)
	for _, tt := range []struct {
		name string
		e    entry
		want time.Time
	}{
		{name: "never deleted", e: entry{touchedAt: deleteAfter}, want: time.Time{}},
		{name: "changed before its deletion time", e: entry{deleteAfter: deleteAfter, touchedAt: deleteAfter.Add(-time.Hour)}, want: deleteAfter},
		{name: "changed after its deletion time", e: entry{deleteAfter: deleteAfter, touchedAt: deleteAfter.Add(time.Hour)}, want: deleteAfter.Add(time.Hour)},
	} {
		require.Equal(t, tt.want, tt.e.forgettableAt(), tt.name)
	}
}

func TestEntry_ApplyRejectsCountOverflow(t *testing.T) {
	for _, tt := range []struct {
		name     string
		boundary int64
		delta    int64
	}{
		{name: "upper", boundary: math.MaxInt64, delta: 1},
		{name: "lower", boundary: math.MinInt64, delta: -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := entry{count: tt.boundary, revision: 1}

			rejected, err := e.apply(tt.delta, time.Time{}, time.Time{})
			require.Error(t, err)
			require.Zero(t, rejected)

			recovered, err := e.apply(-tt.delta, time.Time{}, time.Time{})
			require.NoError(t, err)
			require.Equal(t, entry{count: tt.boundary - tt.delta, revision: 2}, recovered)
		})
	}
}

func TestEntry_ApplyRejectsRevisionOverflow(t *testing.T) {
	e := entry{count: 17, revision: math.MaxUint64 - 1}

	last, err := e.apply(-3, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Equal(t, entry{count: 14, revision: math.MaxUint64}, last)

	rejected, err := last.apply(-5, time.Time{}, time.Time{})
	require.Error(t, err)
	require.Zero(t, rejected)
}

func TestTotalOfRejectsOverflowInBothDirections(t *testing.T) {
	for _, tt := range []struct {
		name  string
		local int64
		peers int64
	}{
		{name: "upper", local: 1, peers: math.MaxInt64},
		{name: "lower", local: -1, peers: math.MinInt64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := totalOf(tt.peers, tt.local)
			require.Error(t, err)

			total, err := totalOf(tt.peers, -tt.local)
			require.NoError(t, err)
			require.Equal(t, tt.peers-tt.local, total)
		})
	}
}

func TestAddToTotalRejectsOverflowInBothDirections(t *testing.T) {
	_, err := addToTotal(math.MaxInt64, 1)
	require.Error(t, err)
	_, err = addToTotal(math.MinInt64, -1)
	require.Error(t, err)

	total, err := addToTotal(math.MaxInt64, -1)
	require.NoError(t, err)
	require.Equal(t, int64(math.MaxInt64-1), total)
}
