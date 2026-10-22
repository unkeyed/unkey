package ratelimit

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// TestDecideChargesRequestsWhenAPeerSessionIsNewer guarantees that a request
// is charged in full when a peer started a session after this pod's live
// session. Adding the request to the older live session would leave
// `reset_at` unchanged, because the peer's newer session holds the largest
// candidate.
func TestDecideChargesRequestsWhenAPeerSessionIsNewer(t *testing.T) {
	current, err := summarize(
		[]session{{id: "peer", anchorAtNs: 150, consumedNs: 60}},
		session{id: "own", anchorAtNs: 100, consumedNs: 1},
	)
	require.NoError(t, err)
	require.Equal(t, int64(210), current.resetAtNs)

	result, err := decide(current, true, 155, Rate{Limit: 60, Duration: 60}, 1)
	require.NoError(t, err)
	require.True(t, result.allowed)
	require.Equal(t, int64(211), result.resetAtNs)
	require.Equal(t, int64(155), result.createNewSessionAtNs)
}

// TestSummarizeRejectsInvalidSessions guarantees that corrupt rows return an
// error instead of a wrong `reset_at`. A zero anchor would also collide with
// the "no new session" value, and a sum beyond int64 would wrap into the past
// and open the rate limit.
func TestSummarizeRejectsInvalidSessions(t *testing.T) {
	cases := []struct {
		name          string
		otherSessions []session
		liveSession   session
	}{
		{"zero anchor", []session{{id: "a", anchorAtNs: 0, consumedNs: 1}}, session{}},
		{"zero anchor of the live session", nil, session{id: "a", anchorAtNs: 0, consumedNs: 1}},
		{"negative consumed time", []session{{id: "a", anchorAtNs: 1, consumedNs: -1}}, session{}},
		{"live session also in the other sessions", []session{{id: "a", anchorAtNs: 1, consumedNs: 1}}, session{id: "a", anchorAtNs: 1, consumedNs: 1}},
		{"consumed sum beyond int64", []session{
			{id: "a", anchorAtNs: 1, consumedNs: math.MaxInt64 - 1},
			{id: "b", anchorAtNs: 1, consumedNs: 2},
		}, session{}},
		{"candidate beyond int64", []session{{id: "a", anchorAtNs: math.MaxInt64 - 5, consumedNs: 6}}, session{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := summarize(tc.otherSessions, tc.liveSession)
			require.Error(t, err)
		})
	}
}

// TestDecideAtTheEdges guarantees that `decide` rejects invalid input,
// decides exactly up to the int64 limit, and returns an error instead of
// wrapping around one nanosecond later. It also guarantees that a huge
// `units` is denied before the 128-bit division, which would panic.
func TestDecideAtTheEdges(t *testing.T) {
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
			result, err := decide(summary{}, false, tc.nowNs, tc.rate, tc.units)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantAllowed, result.allowed)
		})
	}
}

// TestDecideReplaysTheDocumentationTables guarantees that the worked
// examples in the package documentation are what the code does: the
// single-machine table for 10 per 10 seconds, and two pods that over-admitted
// and pay it back.
func TestDecideReplaysTheDocumentationTables(t *testing.T) {
	noonNs := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	secondNs := time.Second.Nanoseconds()
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	type step struct {
		atSeconds        int64
		units            int64
		wantAllowed      bool
		wantResetAtAfter int64
	}
	replay := func(t *testing.T, e *entry, steps []step) {
		t.Helper()
		for index, s := range steps {
			result, err := e.check(noonNs+s.atSeconds*secondNs, rate, s.units)
			require.NoError(t, err)
			require.Equal(t, s.wantAllowed, result.allowed, "step %d", index)
			require.NoError(t, e.spend(result, fmt.Sprintf("own_%d", index)))
			require.Equal(t, noonNs+s.wantResetAtAfter*secondNs, e.summary.resetAtNs, "step %d", index)
		}
	}

	t.Run("single machine", func(t *testing.T) {
		var e entry
		replay(t, &e, []step{
			{0, 1, true, 1},
			{0, 9, true, 10},
			{0, 1, false, 10},
			{1, 1, true, 11},
			{30, 1, true, 31},
		})
	})

	t.Run("over-admission is paid back", func(t *testing.T) {
		var e entry
		replay(t, &e, []step{{0, 10, true, 10}})
		require.NoError(t, e.mergeRows([]session{{id: "pod_b", anchorAtNs: noonNs, consumedNs: 10 * secondNs}}))
		require.Equal(t, noonNs+20*secondNs, e.summary.resetAtNs)
		replay(t, &e, []step{
			{10, 1, false, 20},
			{10, 0, true, 20},
			{11, 1, true, 21},
		})
	})
}

// FuzzDecideMatchesSingleMachineGCRA guarantees the core property of the
// limiter math. For any history of peer rows, stale rows, ended sessions and
// requests, every decision equals single-machine GCRA over the same
// sessions: `reset_at` moves to `max(reset_at, now) + consumed time`, and the
// request fits if that stays within one `Duration` of now.
//
// It also guarantees that:
//   - the sessions reproduce that `reset_at` by the formula over all rows;
//   - the summary always equals `summarize` over the sessions, so the O(1)
//     updates in `summaryAfter` and `summaryWithoutLiveSession` never drift;
//   - a request starts a new session exactly when the live session cannot
//     take it, and the new session is anchored at `max(now, newest anchor)`;
//   - a denied request or a request of zero `units` consumes nothing.
func FuzzDecideMatchesSingleMachineGCRA(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		rate := Rate{
			Limit:    int64(c.Uint8()%20) + 1,
			Duration: time.Duration(c.Uint8()%200) + 1,
		}
		var e entry
		var nowNs int64 = 1_000
		peerCount := 0

		for range 100 {
			switch c.Uint8() % 5 {
			case 0:
				nowNs += int64(c.Uint8())
			case 1:
				peerCount++
				require.NoError(t, e.mergeRows([]session{{
					id:         fmt.Sprintf("peer_%d", peerCount),
					anchorAtNs: nowNs + int64(c.Int8()),
					consumedNs: int64(c.Uint8()),
				}}))
			case 2:
				all := e.all()
				if len(all) == 0 {
					continue
				}
				// A row of any known session, grown or stale. Growing this
				// pod's own sessions here stands for a write that another
				// copy of this pod flushed; a stale row must change nothing.
				row := all[int(c.Uint8())%len(all)]
				row.consumedNs = max(0, row.consumedNs+int64(c.Int8()))
				require.NoError(t, e.mergeRows([]session{row}))
			case 3:
				e.endLiveSession()
			case 4:
				units := int64(c.Uint8()) % (rate.Limit + 2)
				requestMatchesSingleMachine(t, &e, rate, nowNs, units)
			}
			fresh, err := summarize(e.others, e.live)
			require.NoError(t, err)
			require.Equal(t, fresh, e.summary)
		}
	})
}

// requestMatchesSingleMachine checks one request against single-machine GCRA
// and against the formula over all sessions, then spends it.
func requestMatchesSingleMachine(t *testing.T, e *entry, rate Rate, nowNs, units int64) {
	t.Helper()
	all := e.all()
	before := resetAtByFormula(all)
	require.Equal(t, before, e.summary.resetAtNs)

	result, err := e.check(nowNs, rate, units)
	require.NoError(t, err)

	consumedNs := ceilDiv(units*rate.Duration.Nanoseconds(), rate.Limit)
	after := max(before, nowNs) + consumedNs
	allowed := units <= rate.Limit && after-nowNs <= rate.Duration.Nanoseconds()
	require.Equal(t, allowed, result.allowed)
	if !allowed || units == 0 {
		require.Equal(t, before, result.resetAtNs)
		require.Zero(t, result.consumedNs)
		require.Zero(t, result.createNewSessionAtNs)
		require.Equal(t, remainingBySearch(rate, nowNs, before), result.remaining)
		require.NoError(t, e.spend(result, ""))
		require.Equal(t, all, e.all())
		return
	}

	require.Equal(t, after, result.resetAtNs)
	require.Equal(t, consumedNs, result.consumedNs)
	require.Equal(t, remainingBySearch(rate, nowNs, after), result.remaining)

	liveCanTakeIt := false
	if e.live.id != "" {
		withLive := slices.Clone(all)
		withLive[len(withLive)-1].consumedNs += consumedNs
		liveCanTakeIt = resetAtByFormula(withLive) == after
	}
	require.Equal(t, !liveCanTakeIt, result.createNewSessionAtNs != 0, "a new session starts exactly when the live session cannot take the request")
	if result.createNewSessionAtNs != 0 {
		var newestAnchorAtNs int64
		for _, s := range all {
			newestAnchorAtNs = max(newestAnchorAtNs, s.anchorAtNs)
		}
		require.Equal(t, max(nowNs, newestAnchorAtNs), result.createNewSessionAtNs)
	}

	require.NoError(t, e.spend(result, fmt.Sprintf("own_%d", len(all))))
	require.Equal(t, after, resetAtByFormula(e.all()))
}

// resetAtByFormula computes `reset_at` straight from its definition, in O(n²).
func resetAtByFormula(sessions []session) int64 {
	var resetAtNs int64
	for _, r := range sessions {
		candidateNs := r.anchorAtNs
		for _, s := range sessions {
			if s.anchorAtNs >= r.anchorAtNs {
				candidateNs += s.consumedNs
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
