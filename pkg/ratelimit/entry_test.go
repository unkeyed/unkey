package ratelimit

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEntryReplaceRowsKeepsTheLargerConsumedTime guarantees that a stale
// replica row never moves this pod's own session back, so admitted units
// that are not flushed yet still count.
func TestEntryReplaceRowsKeepsTheLargerConsumedTime(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10}
	var e entry
	result, err := e.check(100, rate, 3)
	require.NoError(t, err)
	require.NoError(t, e.apply(result, "own"))

	require.NoError(t, e.replaceRows([]session{{id: "own", anchorAtNs: 100, consumedNs: 1}}, 0))
	require.Equal(t, []session{{id: "own", anchorAtNs: 100, consumedNs: 3}}, allSessions(e))
	require.Equal(t, int64(103), e.resetTimes.resetAtNs)

	require.NoError(t, e.replaceRows([]session{{id: "own", anchorAtNs: 100, consumedNs: 5}}, 0))
	require.Equal(t, int64(105), e.resetTimes.resetAtNs)

	e.endLiveSession()
	require.NoError(t, e.replaceRows([]session{{id: "peer", anchorAtNs: 50, consumedNs: 60}}, 0))
	require.NoError(t, e.replaceRows([]session{
		{id: "own", anchorAtNs: 100, consumedNs: 2},
		{id: "peer", anchorAtNs: 50, consumedNs: 2},
	}, 0))
	require.ElementsMatch(t, []session{
		{id: "own", anchorAtNs: 100, consumedNs: 5},
		{id: "peer", anchorAtNs: 50, consumedNs: 60},
	}, allSessions(e), "stale rows of frozen and peer sessions must not move them back")
}

// TestEntryReplaceRowsUpdatesEveryKnownSession guarantees that a newer row of
// any known session updates that session instead of adding it a second time,
// which would count its consumed time twice.
func TestEntryReplaceRowsUpdatesEveryKnownSession(t *testing.T) {
	var e entry
	require.NoError(t, e.replaceRows([]session{
		{id: "a", anchorAtNs: 10, consumedNs: 1},
		{id: "b", anchorAtNs: 20, consumedNs: 1},
	}, 0))
	require.NoError(t, e.replaceRows([]session{
		{id: "a", anchorAtNs: 10, consumedNs: 3},
		{id: "b", anchorAtNs: 20, consumedNs: 5},
	}, 0))
	require.ElementsMatch(t, []session{
		{id: "a", anchorAtNs: 10, consumedNs: 3},
		{id: "b", anchorAtNs: 20, consumedNs: 5},
	}, allSessions(e))
	require.Equal(t, int64(25), e.resetTimes.resetAtNs)
}

// TestEntryReplaceRowsRejectsInvalidRows guarantees that corrupt rows return an
// error and leave the entry unchanged.
func TestEntryReplaceRowsRejectsInvalidRows(t *testing.T) {
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
			require.NoError(t, e.replaceRows([]session{{id: "known", anchorAtNs: 50, consumedNs: 5}}, 0))
			require.Error(t, e.replaceRows(tc.rows, 0))
			require.Equal(t, []session{{id: "known", anchorAtNs: 50, consumedNs: 5}}, allSessions(e))
			require.Equal(t, int64(55), e.resetTimes.resetAtNs)
		})
	}
}

// TestEntryApplyRejectsAReusedSessionID guarantees that a new session never
// reuses a known ID. The upsert keeps the larger `consumed_ns`, so a session
// that starts again from zero under an old ID would be hidden.
func TestEntryApplyRejectsAReusedSessionID(t *testing.T) {
	rate := Rate{Limit: 60, Duration: 60}
	var e entry
	first, err := e.check(100, rate, 1)
	require.NoError(t, err)
	require.NoError(t, e.apply(first, "own"))
	require.NoError(t, e.replaceRows([]session{{id: "peer", anchorAtNs: 150, consumedNs: 60}}, 0))

	result, err := e.check(155, rate, 1)
	require.NoError(t, err)
	require.NotZero(t, result.createNewSessionAtNs)
	before := snapshot(e)
	require.Error(t, e.apply(result, "peer"))
	require.Error(t, e.apply(result, "own"))
	require.Error(t, e.apply(result, ""))
	require.Equal(t, before, snapshot(e), "a failed apply must not change the entry")
}

// snapshot returns a deep copy of an entry, for comparing it before and after.
func snapshot(e entry) entry {
	return entry{sessions: maps.Clone(e.sessions), liveID: e.liveID, resetTimes: e.resetTimes}
}

// sessionsByID builds the session map of an entry.
func sessionsByID(sessions ...session) map[string]session {
	byID := make(map[string]session, len(sessions))
	for _, s := range sessions {
		byID[s.id] = s
	}
	return byID
}

// allSessions returns the sessions of an entry in no particular order,
// without this pod's write state.
func allSessions(e entry) []session {
	all := make([]session, 0, len(e.sessions))
	for _, s := range e.sessions {
		all = append(all, session{id: s.id, anchorAtNs: s.anchorAtNs, consumedNs: s.consumedNs})
	}
	return all
}

// TestEntryReplaceRowsIsAllOrNothing guarantees that a merge that fails on a
// later row leaves the entry exactly as it was, even after earlier rows in
// the same call were valid. Otherwise the reset times would no longer match
// the sessions.
func TestEntryReplaceRowsIsAllOrNothing(t *testing.T) {
	var e entry
	result, err := e.check(100, Rate{Limit: 10, Duration: 10}, 1)
	require.NoError(t, err)
	require.NoError(t, e.apply(result, "own"))
	require.NoError(t, e.replaceRows([]session{{id: "peer", anchorAtNs: 50, consumedNs: 1}}, 0))
	before := snapshot(e)

	require.Error(t, e.replaceRows([]session{
		{id: "own", anchorAtNs: 100, consumedNs: 9},
		{id: "peer", anchorAtNs: 50, consumedNs: 9},
		{id: "", anchorAtNs: 1, consumedNs: 1},
	}, 0))
	require.Equal(t, before, snapshot(e))

	require.Error(t, e.replaceRows([]session{{id: "own", anchorAtNs: 101, consumedNs: 1}}, 0))
	require.Equal(t, before, snapshot(e))
}

// TestEntryReplaceRowsForgetsOnlyRowsThatCleanupMayHaveChanged guarantees
// that a read forgets peer sessions it did not return, because cleanup may
// have merged them into a returned row, which would count twice. Own sessions
// that the pod still writes stay: the read may come from a replica that has
// not seen them yet, and cleanup cannot change them before their last write.
func TestEntryReplaceRowsForgetsOnlyRowsThatCleanupMayHaveChanged(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10}
	var e entry
	result, err := e.check(100, rate, 1)
	require.NoError(t, err)
	require.NoError(t, e.apply(result, "old"))
	e.endLiveSession()
	result, err = e.check(200, rate, 1)
	require.NoError(t, err)
	require.NoError(t, e.apply(result, "live"))
	require.NoError(t, e.replaceRows([]session{{id: "peer", anchorAtNs: 50, consumedNs: 1}}, 200))

	require.NoError(t, e.replaceRows([]session{{id: "merged", anchorAtNs: 40, consumedNs: 2}}, lastWriteAtNs(100)))
	require.ElementsMatch(t, []session{
		{id: "merged", anchorAtNs: 40, consumedNs: 2},
		{id: "old", anchorAtNs: 100, consumedNs: 1},
		{id: "live", anchorAtNs: 200, consumedNs: 1},
	}, allSessions(e))
	require.Equal(t, "live", e.liveID)

	require.NoError(t, e.replaceRows(nil, lastWriteAtNs(100)+1))
	require.Equal(t, []session{{id: "live", anchorAtNs: 200, consumedNs: 1}}, allSessions(e))

	require.NoError(t, e.replaceRows(nil, lastWriteAtNs(200)+1))
	require.Empty(t, allSessions(e))
	require.Empty(t, e.liveID, "a forgotten live session must end")
	fresh, err := computeResetTimes(e.sessions, e.liveID)
	require.NoError(t, err)
	require.Equal(t, fresh, e.resetTimes)
}
