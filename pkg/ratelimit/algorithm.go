package ratelimit

import (
	"cmp"
	"math"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// session is one pod's record of the time it admitted for one rate limit,
// counted from `anchorAtNs` on. It is one row of the `ratelimit_state` table,
// or one of this pod's sessions in memory.
type session struct {
	// id is unique per session. The pod creates it together with the session
	// and never uses it again for another session.
	id string

	// anchorAtNs is the fixed time from which this session's consumed time
	// counts, in Unix nanoseconds. It never changes. It can be ahead of the
	// pod's own clock when a peer had a later anchor; see `decide`.
	anchorAtNs int64

	// consumedNs is the consumed time of the units this session admitted. It
	// only grows.
	consumedNs int64
}

// summary holds the values that `decide` needs, computed from all sessions
// of one rate limit by `summarize`. With it, a decision is O(1) and does not
// walk the sessions.
type summary struct {
	// resetAtNs is `reset_at`: the largest candidate of all sessions. It is
	// zero without sessions.
	resetAtNs int64

	// maxCandidateRaisedByLiveSessionNs is the largest candidate among the
	// sessions anchored at or before the live session, the live session
	// included. It is zero without a live session.
	//
	//	maxCandidateRaisedByLiveSession = max over sessions r with r.anchor ≤ live.anchor of
	//	        (r.anchor + sum of consumed of sessions anchored at or after r)
	//
	// Adding `c` to the live session then gives
	//
	//	reset_at' = max(maxCandidateRaisedByLiveSession + c, reset_at)
	//
	// and the GCRA rule requires `reset_at' = max(reset_at, now) + c`.
	// The two are equal exactly when
	// `maxCandidateRaisedByLiveSession = reset_at ≥ now`.
	//
	// `decide` needs it to know whether the live session can take a request.
	// Units added to the live session raise only these candidates, not the
	// candidates of sessions anchored after it. If this maximum equals
	// `resetAtNs`, adding the units moves `reset_at` exactly as the GCRA rule
	// requires. If it is smaller, a newer peer session holds
	// `reset_at`, adding the units would not move `reset_at` far enough, and
	// the request must start a new session instead.
	maxCandidateRaisedByLiveSessionNs int64

	// newestAnchorAtNs is the largest anchor of all sessions. A new session
	// is never anchored before it, so that every other session counts the new
	// session's consumed time.
	newestAnchorAtNs int64
}

// decision is the result of `decide`.
type decision struct {
	allowed bool

	// consumedNs is the consumed time of the request. It is zero if the
	// request was denied or had zero `units`, and then nothing is written.
	consumedNs int64

	// resetAtNs is `reset_at` after the request, in Unix nanoseconds: the
	// time when every admitted unit is paid back and a request of `Limit`
	// units fits again. It is in the past when the rate limit is idle.
	resetAtNs int64

	// remaining is the largest number of units that one more request could
	// use at the same time. For a denied request it is how many fit now.
	remaining int64

	// createNewSessionAtNs is zero when the request goes into the live
	// session. Otherwise the live session cannot take the request: the
	// caller freezes it and starts a new live session anchored at
	// `createNewSessionAtNs`. The frozen session keeps counting in full.
	//
	// Zero is free to mean "no new session" because `decide` requires a
	// positive time and a new anchor is never before it.
	createNewSessionAtNs int64
}

// summarize computes the summary of all sessions of one rate limit.
// `reset_at` is the largest candidate, and the candidate of a session r is
// r's anchor plus the consumed time of every session anchored at or after r:
//
//	reset_at = max over sessions r of
//	           (r.anchor + sum of consumed of sessions anchored at or after r)
//
// For example, with three sessions sorted newest first:
//
//	session   anchor     consumed   candidate
//	B         12:00:03   9s         12:00:03 + 9s             = 12:00:12
//	A         12:00:00   12s        12:00:00 + 9s + 12s       = 12:00:21   <- reset_at
//	C         11:58:40   3s         11:58:40 + 9s + 12s + 3s  = 11:59:04
//
// A candidate answers one question: if the rate limit had no pause from this
// anchor on, when would it be free? The largest candidate is right because
// `decide` applies the GCRA rule `max(reset_at, now) + consumed time` with
// every admitted request. `FuzzDecideFollowsGCRARule` checks this for random
// histories of sessions and requests.
//
// `otherSessions` are the sessions of peers and this pod's frozen sessions.
// `liveSession` is this pod's live session, or nil if it has none.
// summarize is O(n log n) for n sessions and runs once per refresh.
//
// summarize returns an error if an anchor is not positive, if a consumed time
// is negative, if a sum overflows int64, or if the live session is also in
// `otherSessions`. It does not change `otherSessions`.
func summarize(otherSessions []session, liveSession *session) (summary, error) {
	// The candidate of a session needs the sum of everything anchored at or
	// after it. Walking from the newest anchor to the oldest builds that sum
	// as a running total, so all candidates take one pass.
	newestFirst := slices.Clone(otherSessions)
	if liveSession != nil {
		newestFirst = append(newestFirst, *liveSession)
	}
	slices.SortFunc(newestFirst, func(left, right session) int {
		return cmp.Compare(right.anchorAtNs, left.anchorAtNs)
	})

	for _, s := range newestFirst {
		err := assert.All(
			assert.Greater(s.anchorAtNs, 0, "session anchor must be positive"),
			assert.GreaterOrEqual(s.consumedNs, 0, "session consumed time must not be negative"),
		)
		if err != nil {
			return summary{}, err
		}
	}
	if liveSession != nil && slices.ContainsFunc(otherSessions, func(s session) bool { return s.id == liveSession.id }) {
		return summary{}, fault.New("live session must not also be in the other sessions")
	}

	var result summary
	var consumedAtOrAfterAnchorNs int64
	for _, s := range newestFirst {
		var err error
		consumedAtOrAfterAnchorNs, err = addNs(consumedAtOrAfterAnchorNs, s.consumedNs)
		if err != nil {
			return summary{}, err
		}
		// Sessions with the same anchor count toward each other. They need no
		// special case: the first of them gets a candidate with only part of
		// their sum, but the last gets the full sum, and only the largest
		// candidate matters.
		candidateNs, err := addNs(s.anchorAtNs, consumedAtOrAfterAnchorNs)
		if err != nil {
			return summary{}, err
		}
		result.resetAtNs = max(result.resetAtNs, candidateNs)
		if liveSession != nil && s.anchorAtNs <= liveSession.anchorAtNs {
			result.maxCandidateRaisedByLiveSessionNs = max(result.maxCandidateRaisedByLiveSessionNs, candidateNs)
		}
	}
	if len(newestFirst) > 0 {
		result.newestAnchorAtNs = newestFirst[0].anchorAtNs
	}
	return result, nil
}

// decide decides a request of `units` units at `nowNs`. An allowed request
// moves `reset_at` to `max(reset_at, nowNs) + consumed time of units`, the
// GCRA rule. A request fits if `reset_at` then stays at most one
// `rate.Duration` ahead of `nowNs`. More `units` than `rate.Limit` never fit.
//
// decide is pure, so a caller can decide several rate limits and spend on
// all of them or on none.
//
// decide returns an error if `rate` or `units` is invalid, if `nowNs` is not
// positive, or if the new `reset_at` does not fit into int64.
func decide(current summary, hasLiveSession bool, nowNs int64, rate Rate, units int64) (decision, error) {
	err := assert.All(
		rate.Validate(),
		assert.GreaterOrEqual(units, 0, "units must not be negative"),
		assert.Greater(nowNs, 0, "time must be after the Unix epoch"),
	)
	if err != nil {
		return decision{}, err
	}

	// A denied request returns the state as it is. Nothing is written, so a
	// caller who keeps hitting a closed rate limit costs no database writes.
	unchanged := decision{
		allowed:              false, //faultline:equivalent the zero value; exhaustruct requires the field
		consumedNs:           0,     //faultline:equivalent the zero value; exhaustruct requires the field
		resetAtNs:            current.resetAtNs,
		remaining:            rate.remaining(nowNs, current.resetAtNs),
		createNewSessionAtNs: 0, //faultline:equivalent the zero value; exhaustruct requires the field
	}
	// At most `Limit` units are ever free, so more `units` never fit.
	// Checking it here also keeps the 128-bit division in `unitsToNs` from
	// panicking.
	if units > rate.Limit {
		return unchanged, nil
	}
	consumedNs := rate.unitsToNs(units)
	// This is the GCRA rule. If `reset_at` is in the past,
	// every unit is free, and the request pays from now, not from the old
	// `reset_at`. Otherwise the request waits behind what is already in use.
	nextResetAtNs, err := addNs(max(current.resetAtNs, nowNs), consumedNs)
	if err != nil {
		return decision{}, err
	}
	// The burst tolerance is one full `Duration`: `reset_at` may run at most
	// that far ahead of now. When pods over-admitted, `reset_at` can already
	// be further ahead, and every request is denied until time catches up.
	if nextResetAtNs-nowNs > rate.Duration.Nanoseconds() {
		return unchanged, nil
	}
	// A request of zero `units` only asks whether the rate limit is open. It
	// spends nothing, so it must not start a session or write a row.
	if units == 0 {
		unchanged.allowed = true
		return unchanged, nil
	}

	// Adding `consumedNs` to the live session gives
	// `max(maxCandidateRaisedByLiveSessionNs + consumedNs, resetAtNs)`. That
	// equals `nextResetAtNs` exactly when a candidate raised by the live
	// session already equals `reset_at` (a tie is enough) and `reset_at` is
	// not in the past.
	//
	// Example, 60 units per 60 ns, `nowNs` = 155:
	//
	//	live session  anchor 100, consumed 1    candidate 100 + 1 + 60 = 161
	//	peer session  anchor 150, consumed 60   candidate 150 + 60     = 210  <- reset_at
	//
	// Adding 1 unit to the live session makes its candidate 162, and
	// `reset_at` stays 210: the request would be free. The GCRA rule
	// gives 211. A new session anchored at 155 has candidate 155 + 1 = 156,
	// and the peer session now counts it too: 150 + 60 + 1 = 211.
	liveSessionCanTakeRequest := hasLiveSession &&
		current.maxCandidateRaisedByLiveSessionNs == current.resetAtNs &&
		current.resetAtNs >= nowNs
	var createNewSessionAtNs int64
	if !liveSessionCanTakeRequest {
		// Anchored at or after the newest anchor, the new session is counted
		// by every other session. The newest anchor can be ahead of `nowNs`
		// when the clocks of two pods differ a little.
		createNewSessionAtNs = max(nowNs, current.newestAnchorAtNs)
	}
	return decision{
		allowed:              true,
		consumedNs:           consumedNs,
		resetAtNs:            nextResetAtNs,
		remaining:            rate.remaining(nowNs, nextResetAtNs),
		createNewSessionAtNs: createNewSessionAtNs,
	}, nil
}

// summaryAfter returns the summary after an allowed decision with a positive
// `consumedNs` was written into the sessions. It is O(1). The decision must
// come from `decide` on `current`.
func summaryAfter(current summary, spent decision) summary {
	// The request raised every candidate at or before the live session. A new
	// session is anchored at or after every other session, so there it raised
	// every candidate. Either way, the largest raised candidate is the new
	// `reset_at`.
	next := summary{
		resetAtNs:                         spent.resetAtNs,
		maxCandidateRaisedByLiveSessionNs: spent.resetAtNs,
		newestAnchorAtNs:                  current.newestAnchorAtNs,
	}
	if spent.createNewSessionAtNs != 0 {
		next.newestAnchorAtNs = spent.createNewSessionAtNs
	}
	return next
}

// summaryWithoutLiveSession returns the summary after the live session was
// frozen. It keeps counting, so `reset_at` does not change, but no candidate
// is raised by a live session anymore.
func summaryWithoutLiveSession(current summary) summary {
	current.maxCandidateRaisedByLiveSessionNs = 0
	return current
}

// addNs returns `a + b` for non-negative `a` and `b`, or an error if the sum
// does not fit into int64. The limiter must never wrap around, because a
// wrapped `reset_at` would open the rate limit.
func addNs(a, b int64) (int64, error) {
	if a > math.MaxInt64-b {
		return 0, fault.New("time overflows int64 nanoseconds") //faultline:equivalent callers discard the value when there is an error
	}
	return a + b, nil
}
