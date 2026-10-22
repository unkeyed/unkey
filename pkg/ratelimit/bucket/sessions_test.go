package bucket

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// TestSessionsChargesRequestsWhenAPeerSessionIsNewer guarantees that a
// request is charged in full when a peer started a session after this pod's
// live session. Adding the request to the older live session would leave
// `reset_at` unchanged, because the peer's newer session holds the largest
// candidate.
func TestSessionsChargesRequestsWhenAPeerSessionIsNewer(t *testing.T) {
	rate := Rate{Limit: 60, Duration: 60}
	var sessions Sessions
	spend(t, &sessions, 100, rate, 1, "own")
	require.NoError(t, sessions.MergeRows([]Session{{ID: "peer", AnchorAtNs: 150, ConsumedNs: 60}}))
	require.Equal(t, int64(210), sessions.ResetAtNs())

	decision, err := sessions.Check(155, rate, 1)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Equal(t, int64(211), decision.ResetAtNs)
	require.Equal(t, int64(155), decision.CreateNewSessionAtNs)
}

// TestSessionsMergeRowsKeepsTheLargerConsumedTime guarantees that a stale
// replica row never moves this pod's own session back, so admitted units
// that are not flushed yet still count.
func TestSessionsMergeRowsKeepsTheLargerConsumedTime(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10}
	var sessions Sessions
	spend(t, &sessions, 100, rate, 3, "own")
	require.NoError(t, sessions.MergeRows([]Session{{ID: "own", AnchorAtNs: 100, ConsumedNs: 1}}))
	require.Equal(t, []Session{{ID: "own", AnchorAtNs: 100, ConsumedNs: 3}}, sessions.All())
	require.Equal(t, int64(103), sessions.ResetAtNs())

	require.NoError(t, sessions.MergeRows([]Session{{ID: "own", AnchorAtNs: 100, ConsumedNs: 5}}))
	require.Equal(t, int64(105), sessions.ResetAtNs())
}

// TestSessionsMergeRowsRejectsInvalidRows guarantees that corrupt rows return
// an error and leave the sessions unchanged, instead of a wrong `reset_at`. A
// zero anchor would also collide with the "no new session" value, and a sum
// beyond int64 would wrap into the past and open the rate limit.
func TestSessionsMergeRowsRejectsInvalidRows(t *testing.T) {
	cases := []struct {
		name string
		rows []Session
	}{
		{"empty ID", []Session{{ID: "", AnchorAtNs: 1, ConsumedNs: 1}}},
		{"zero anchor", []Session{{ID: "a", AnchorAtNs: 0, ConsumedNs: 1}}},
		{"negative consumed time", []Session{{ID: "a", AnchorAtNs: 1, ConsumedNs: -1}}},
		{"changed anchor of a known session", []Session{{ID: "known", AnchorAtNs: 51, ConsumedNs: 1}}},
		{"changed anchor within the rows", []Session{{ID: "a", AnchorAtNs: 1, ConsumedNs: 1}, {ID: "a", AnchorAtNs: 2, ConsumedNs: 1}}},
		{"consumed sum beyond int64", []Session{
			{ID: "a", AnchorAtNs: 1, ConsumedNs: math.MaxInt64 - 1},
			{ID: "b", AnchorAtNs: 1, ConsumedNs: 2},
		}},
		{"candidate beyond int64", []Session{{ID: "a", AnchorAtNs: math.MaxInt64 - 5, ConsumedNs: 6}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sessions Sessions
			require.NoError(t, sessions.MergeRows([]Session{{ID: "known", AnchorAtNs: 50, ConsumedNs: 5}}))
			require.Error(t, sessions.MergeRows(tc.rows))
			require.Equal(t, []Session{{ID: "known", AnchorAtNs: 50, ConsumedNs: 5}}, sessions.All())
			require.Equal(t, int64(55), sessions.ResetAtNs())
		})
	}
}

// TestSessionsCheckAtTheEdges guarantees that `Check` rejects invalid input,
// decides exactly up to the int64 limit, and returns an error instead of
// wrapping around one nanosecond later. It also guarantees that a huge
// `units` is denied before the 128-bit division, which would panic.
func TestSessionsCheckAtTheEdges(t *testing.T) {
	cases := []struct {
		name        string
		nowNs       int64
		rate        Rate
		units       int64
		wantErr     bool
		wantAllowed bool
	}{
		{"zero time", 0, Rate{Limit: 1, Duration: 2}, 1, true, false},
		{"negative units", 1, Rate{Limit: 1, Duration: 2}, -1, true, false},
		{"zero limit", 1, Rate{Limit: 0, Duration: 2}, 1, true, false},
		{"zero duration", 1, Rate{Limit: 1, Duration: 0}, 1, true, false},
		{"reset_at exactly at the int64 maximum", math.MaxInt64 - 2, Rate{Limit: 1, Duration: 2}, 1, false, true},
		{"reset_at one past the int64 maximum", math.MaxInt64 - 1, Rate{Limit: 1, Duration: 2}, 1, true, false},
		{"huge units on a huge duration", 1, Rate{Limit: 1, Duration: math.MaxInt64}, math.MaxInt64, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sessions Sessions
			decision, err := sessions.Check(tc.nowNs, tc.rate, tc.units)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantAllowed, decision.Allowed)
		})
	}
}

// TestSessionsSpendRejectsAReusedSessionID guarantees that a new session
// never reuses a known ID. The upsert keeps the larger `consumed_ns`, so a
// session that starts again from zero under an old ID would be hidden.
func TestSessionsSpendRejectsAReusedSessionID(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10}
	var sessions Sessions
	require.NoError(t, sessions.MergeRows([]Session{{ID: "peer", AnchorAtNs: 50, ConsumedNs: 1}}))
	decision, err := sessions.Check(100, rate, 1)
	require.NoError(t, err)
	require.NotZero(t, decision.CreateNewSessionAtNs)
	require.Error(t, sessions.Spend(decision, "peer"))
	require.Error(t, sessions.Spend(decision, ""))
	require.Equal(t, []Session{{ID: "peer", AnchorAtNs: 50, ConsumedNs: 1}}, sessions.All())
}

// FuzzSessionsMatchesSingleMachineGCRA guarantees the core property of the
// limiter math. For any history of peer rows, stale rows, ended sessions and
// requests, every decision equals single-machine GCRA over the same
// sessions: `reset_at` moves to `max(reset_at, now) + consumed time`, and the
// request fits if that stays within one `Duration` of now.
//
// It also guarantees that:
//   - the sessions reproduce that `reset_at` by the formula over all rows;
//   - the cached values always equal a fresh computation, so the O(1)
//     updates in `Spend` and `EndLiveSession` never drift;
//   - `Check` changes nothing;
//   - a request starts a new session exactly when the live session cannot
//     take it, and the new session is anchored at `max(now, newest anchor)`;
//   - a denied request or a request of zero `units` consumes nothing.
func FuzzSessionsMatchesSingleMachineGCRA(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		rate := Rate{
			Limit:    int64(c.Uint8()%20) + 1,
			Duration: time.Duration(c.Uint8()%200) + 1,
		}
		var sessions Sessions
		var nowNs int64 = 1_000
		peerCount := 0

		for range 100 {
			switch c.Uint8() % 5 {
			case 0:
				nowNs += int64(c.Uint8())
			case 1:
				peerCount++
				row := Session{
					ID:         fmt.Sprintf("peer_%d", peerCount),
					AnchorAtNs: nowNs + int64(c.Int8()),
					ConsumedNs: int64(c.Uint8()),
				}
				require.NoError(t, sessions.MergeRows([]Session{row}))
			case 2:
				all := sessions.All()
				if len(all) == 0 {
					continue
				}
				// A row of any known session, grown or stale. Growing this
				// pod's own sessions here stands for a write that another
				// copy of this pod flushed; a stale row must change nothing.
				row := all[int(c.Uint8())%len(all)]
				row.ConsumedNs = max(0, row.ConsumedNs+int64(c.Int8()))
				require.NoError(t, sessions.MergeRows([]Session{row}))
			case 3:
				sessions.EndLiveSession()
			case 4:
				units := int64(c.Uint8()) % (rate.Limit + 2)
				requestMatchesSingleMachine(t, &sessions, rate, nowNs, units)
			}
			requireCacheMatchesSessions(t, sessions)
		}
	})
}

// requestMatchesSingleMachine checks one request against single-machine GCRA
// and against the formula over all sessions, then spends it.
func requestMatchesSingleMachine(t *testing.T, sessions *Sessions, rate Rate, nowNs, units int64) {
	t.Helper()
	all := sessions.All()
	before := resetAtByFormula(all)
	require.Equal(t, before, sessions.ResetAtNs())

	beforeCheck := *sessions
	beforeCheck.others = slices.Clone(sessions.others)
	decision, err := sessions.Check(nowNs, rate, units)
	require.NoError(t, err)
	require.Equal(t, beforeCheck, *sessions, "Check must not change the sessions")

	consumedNs := ceilDiv(units*rate.Duration.Nanoseconds(), rate.Limit)
	after := max(before, nowNs) + consumedNs
	allowed := units <= rate.Limit && after-nowNs <= rate.Duration.Nanoseconds()
	require.Equal(t, allowed, decision.Allowed)
	if !allowed || units == 0 {
		require.Equal(t, before, decision.ResetAtNs)
		require.Zero(t, decision.ConsumedNs)
		require.Zero(t, decision.CreateNewSessionAtNs)
		require.Equal(t, remainingBySearch(rate, nowNs, before), decision.Remaining)
		require.NoError(t, sessions.Spend(decision, ""))
		require.Equal(t, all, sessions.All())
		return
	}

	require.Equal(t, after, decision.ResetAtNs)
	require.Equal(t, consumedNs, decision.ConsumedNs)
	require.Equal(t, remainingBySearch(rate, nowNs, after), decision.Remaining)

	liveCanTakeIt := false
	if sessions.live.ID != "" {
		withLive := slices.Clone(all)
		withLive[len(withLive)-1].ConsumedNs += consumedNs
		liveCanTakeIt = resetAtByFormula(withLive) == after
	}
	require.Equal(t, !liveCanTakeIt, decision.CreateNewSessionAtNs != 0, "a new session starts exactly when the live session cannot take the request")
	if decision.CreateNewSessionAtNs != 0 {
		var newestAnchorAtNs int64
		for _, session := range all {
			newestAnchorAtNs = max(newestAnchorAtNs, session.AnchorAtNs)
		}
		require.Equal(t, max(nowNs, newestAnchorAtNs), decision.CreateNewSessionAtNs)
	}

	require.NoError(t, sessions.Spend(decision, fmt.Sprintf("own_%d", len(all))))
	require.Equal(t, after, resetAtByFormula(sessions.All()))
}

// requireCacheMatchesSessions checks that the cached values equal a fresh
// computation over the same sessions.
func requireCacheMatchesSessions(t *testing.T, sessions Sessions) {
	t.Helper()
	fresh := sessions
	require.NoError(t, fresh.recompute())
	require.Equal(t, fresh.resetAtNs, sessions.resetAtNs)
	require.Equal(t, fresh.maxCandidateRaisedByLiveSessionNs, sessions.maxCandidateRaisedByLiveSessionNs)
	require.Equal(t, fresh.newestAnchorAtNs, sessions.newestAnchorAtNs)
}

// spend checks and spends one request and requires it to be allowed.
func spend(t *testing.T, sessions *Sessions, nowNs int64, rate Rate, units int64, newSessionID string) {
	t.Helper()
	decision, err := sessions.Check(nowNs, rate, units)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.NoError(t, sessions.Spend(decision, newSessionID))
}

// resetAtByFormula computes `reset_at` straight from its definition, in O(n²).
func resetAtByFormula(sessions []Session) int64 {
	var resetAtNs int64
	for _, r := range sessions {
		candidateNs := r.AnchorAtNs
		for _, s := range sessions {
			if s.AnchorAtNs >= r.AnchorAtNs {
				candidateNs += s.ConsumedNs
			}
		}
		resetAtNs = max(resetAtNs, candidateNs)
	}
	return resetAtNs
}

// remainingBySearch returns the largest number of units that fit at `nowNs`.
func remainingBySearch(rate Rate, nowNs, resetAtNs int64) int64 {
	for units := rate.Limit; units > 0; units-- {
		if max(resetAtNs, nowNs)+ceilDiv(units*rate.Duration.Nanoseconds(), rate.Limit)-nowNs <= rate.Duration.Nanoseconds() {
			return units
		}
	}
	return 0
}

func ceilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}
