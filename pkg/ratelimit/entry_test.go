package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEntryMergeRowsKeepsTheLargerConsumedTime guarantees that a stale
// replica row never moves this pod's own session back, so admitted units
// that are not flushed yet still count.
func TestEntryMergeRowsKeepsTheLargerConsumedTime(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10}
	var e entry
	result, err := e.check(100, rate, 3)
	require.NoError(t, err)
	require.NoError(t, e.spend(result, "own"))

	require.NoError(t, e.mergeRows([]session{{id: "own", anchorAtNs: 100, consumedNs: 1}}))
	require.Equal(t, []session{{id: "own", anchorAtNs: 100, consumedNs: 3}}, e.all())
	require.Equal(t, int64(103), e.summary.resetAtNs)

	require.NoError(t, e.mergeRows([]session{{id: "own", anchorAtNs: 100, consumedNs: 5}}))
	require.Equal(t, int64(105), e.summary.resetAtNs)
}

// TestEntryMergeRowsRejectsInvalidRows guarantees that corrupt rows return an
// error and leave the entry unchanged.
func TestEntryMergeRowsRejectsInvalidRows(t *testing.T) {
	cases := []struct {
		name string
		rows []session
	}{
		{"empty ID", []session{{id: "", anchorAtNs: 1, consumedNs: 1}}},
		{"zero anchor", []session{{id: "a", anchorAtNs: 0, consumedNs: 1}}},
		{"changed anchor of a known session", []session{{id: "known", anchorAtNs: 51, consumedNs: 1}}},
		{"changed anchor within the rows", []session{{id: "a", anchorAtNs: 1, consumedNs: 1}, {id: "a", anchorAtNs: 2, consumedNs: 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var e entry
			require.NoError(t, e.mergeRows([]session{{id: "known", anchorAtNs: 50, consumedNs: 5}}))
			require.Error(t, e.mergeRows(tc.rows))
			require.Equal(t, []session{{id: "known", anchorAtNs: 50, consumedNs: 5}}, e.all())
			require.Equal(t, int64(55), e.summary.resetAtNs)
		})
	}
}

// TestEntrySpendRejectsAReusedSessionID guarantees that a new session never
// reuses a known ID. The upsert keeps the larger `consumed_ns`, so a session
// that starts again from zero under an old ID would be hidden.
func TestEntrySpendRejectsAReusedSessionID(t *testing.T) {
	var e entry
	require.NoError(t, e.mergeRows([]session{{id: "peer", anchorAtNs: 50, consumedNs: 1}}))
	result, err := e.check(100, Rate{Limit: 10, Duration: 10}, 1)
	require.NoError(t, err)
	require.NotZero(t, result.createNewSessionAtNs)
	require.Error(t, e.spend(result, "peer"))
	require.Error(t, e.spend(result, ""))
	require.Equal(t, []session{{id: "peer", anchorAtNs: 50, consumedNs: 1}}, e.all())
}
