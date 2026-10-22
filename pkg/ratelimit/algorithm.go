package ratelimit

import (
	"cmp"
	"maps"
	"math"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// session is one row of `ratelimit_state`, or one of this pod's sessions in
// memory.
type session struct {
	// id is never reused for another session.
	id string

	// anchorAtNs never changes.
	anchorAtNs int64

	// consumedNs only grows.
	consumedNs int64
}

// resetTimes holds `reset_at` and the two other times that `decide` needs
// from the sessions, so a decision is O(1).
type resetTimes struct {
	// resetAtNs is the largest candidate of all sessions.
	resetAtNs int64

	// maxCandidateRaisedByLiveSessionNs is the largest candidate among the
	// sessions anchored at or before the live session, or zero without one:
	//
	//	max over sessions r with r.anchor ≤ live.anchor of
	//	    (r.anchor + sum of consumed of sessions anchored at or after r)
	//
	// Units `c` added to the live session raise only these candidates, so
	//
	//	reset_at' = max(maxCandidateRaisedByLiveSession + c, reset_at)
	//
	// That follows the GCRA rule `max(reset_at, now) + c` exactly when
	// `maxCandidateRaisedByLiveSession = reset_at ≥ now`. Otherwise the
	// request needs a new session.
	maxCandidateRaisedByLiveSessionNs int64

	// newestAnchorAtNs is the largest anchor of all sessions.
	newestAnchorAtNs int64
}

// decision is the result of `decide`.
type decision struct {
	allowed bool

	// consumedNs is zero for a denied request or zero `units`; then nothing is
	// written.
	consumedNs int64

	resetAtNs int64

	// remaining is how many units one more request could use right now.
	remaining int64

	// createNewSessionAtNs is zero if the request goes into the live session.
	// Otherwise the caller must create a fresh session with a new ID, anchored
	// here, and never write to the old live session again. Writing to the old
	// one would charge too little.
	//
	//	createNewSessionAt = max(now, newestAnchorAt)
	//
	// At or after the newest anchor, every existing candidate counts the new
	// `c`. At or after now, the new session's own candidate is at least
	// `now + c`. Since `newestAnchorAt ≤ reset_at`:
	//
	//	reset_at' = max(reset_at + c, max(now, newestAnchorAt) + c)
	//	          = max(reset_at, now) + c
	//
	// An earlier anchor charges too little, a later one too much. Zero is free
	// as a marker because `decide` requires a positive time.
	createNewSessionAtNs int64
}

// computeResetTimes computes the reset times of all sessions in O(n log n):
//
//	reset_at = max over sessions r of
//	           (r.anchor + sum of consumed of sessions anchored at or after r)
//
// For example:
//
//	session   anchor     consumed   candidate
//	B         12:00:03   9s         12:00:03 + 9s             = 12:00:12
//	A         12:00:00   12s        12:00:00 + 9s + 12s       = 12:00:21   <- reset_at
//	C         11:58:40   3s         11:58:40 + 9s + 12s + 3s  = 11:59:04
//
// A candidate is when the limit would be free if it had no pause since that
// anchor. `liveID` is "" if the pod has no live session. computeResetTimes
// returns an error for a non-positive anchor, a negative consumed time, an
// int64 overflow, or a `liveID` that is not in `sessions`.
//
// It only reads its arguments, so it is safe to call concurrently.
func computeResetTimes(sessions map[string]session, liveID string) (resetTimes, error) {
	live, hasLive := sessions[liveID]
	if liveID != "" && !hasLive {
		return resetTimes{}, fault.New("live session must be in the sessions")
	}
	// Newest first, so the sum of later sessions is a running total.
	newestFirst := slices.SortedFunc(maps.Values(sessions), func(left, right session) int {
		return cmp.Compare(right.anchorAtNs, left.anchorAtNs)
	})

	var result resetTimes
	var consumedAtOrAfterAnchorNs int64
	for _, s := range newestFirst {
		err := assert.All(
			assert.Greater(s.anchorAtNs, 0, "session anchor must be positive"),
			assert.GreaterOrEqual(s.consumedNs, 0, "session consumed time must not be negative"),
		)
		if err != nil {
			return resetTimes{}, err
		}
		consumedAtOrAfterAnchorNs, err = checkedAdd(consumedAtOrAfterAnchorNs, s.consumedNs)
		if err != nil {
			return resetTimes{}, err
		}
		// Equal anchors need no special case: the last of them gets the full
		// sum, and only the largest candidate matters.
		candidateNs, err := checkedAdd(s.anchorAtNs, consumedAtOrAfterAnchorNs)
		if err != nil {
			return resetTimes{}, err
		}
		result.resetAtNs = max(result.resetAtNs, candidateNs)
		if hasLive && s.anchorAtNs <= live.anchorAtNs {
			result.maxCandidateRaisedByLiveSessionNs = max(result.maxCandidateRaisedByLiveSessionNs, candidateNs)
		}
	}
	if len(newestFirst) > 0 {
		result.newestAnchorAtNs = newestFirst[0].anchorAtNs
	}
	return result, nil
}

// decide applies the GCRA rule to a request of `units` at `nowNs`. It is pure,
// so a caller can decide several rate limits before applying any. It
// returns an error for invalid input or an int64 overflow. It is safe to call
// concurrently.
func decide(current resetTimes, nowNs int64, rate Rate, units int64) (decision, error) {
	err := assert.All(
		rate.Validate(),
		assert.GreaterOrEqual(units, 0, "units must not be negative"),
		assert.Greater(nowNs, 0, "time must be after the Unix epoch"),
	)
	if err != nil {
		return decision{}, err
	}

	unchanged := decision{
		allowed:              false, //faultline:equivalent the zero value; exhaustruct requires the field
		consumedNs:           0,     //faultline:equivalent the zero value; exhaustruct requires the field
		resetAtNs:            current.resetAtNs,
		remaining:            rate.remaining(nowNs, current.resetAtNs),
		createNewSessionAtNs: 0, //faultline:equivalent the zero value; exhaustruct requires the field
	}
	// More than `Limit` never fits. Checked first so `unitsToNs` cannot panic.
	if units > rate.Limit {
		return unchanged, nil
	}
	consumedNs := rate.unitsToNs(units)
	nextResetAtNs, err := checkedAdd(max(current.resetAtNs, nowNs), consumedNs)
	if err != nil {
		return decision{}, err
	}
	if nextResetAtNs-nowNs > rate.Duration.Nanoseconds() {
		return unchanged, nil
	}
	// Zero units only ask whether the limit is open; they write nothing.
	if units == 0 {
		unchanged.allowed = true
		return unchanged, nil
	}

	// See `maxCandidateRaisedByLiveSessionNs`. Example, 60 units per 60 ns,
	// now = 155:
	//
	//	live session  anchor 100, consumed 1    candidate 100 + 1 + 60 = 161
	//	peer session  anchor 150, consumed 60   candidate 150 + 60     = 210  <- reset_at
	//
	// In the live session, 1 unit leaves `reset_at` at 210; it would be free.
	// In a new session at 155, the peer counts it: 150 + 60 + 1 = 211.
	// Without a live session the first condition fails: the maximum is zero,
	// and `reset_at` is only zero without sessions, where `reset_at < now`.
	liveSessionCanTakeRequest := current.maxCandidateRaisedByLiveSessionNs == current.resetAtNs &&
		current.resetAtNs >= nowNs
	var createNewSessionAtNs int64
	if !liveSessionCanTakeRequest {
		// The newest anchor can be ahead of now when pod clocks differ.
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

// resetTimesAfter returns the reset times after an allowed decision with positive
// `consumedNs` was applied. The decision must come from `decide` on `current`.
// It only reads its arguments, so it is safe to call concurrently.
func resetTimesAfter(current resetTimes, applied decision) resetTimes {
	// Either way, the largest raised candidate is the new `reset_at`.
	next := resetTimes{
		resetAtNs:                         applied.resetAtNs,
		maxCandidateRaisedByLiveSessionNs: applied.resetAtNs,
		newestAnchorAtNs:                  current.newestAnchorAtNs,
	}
	if applied.createNewSessionAtNs != 0 {
		next.newestAnchorAtNs = applied.createNewSessionAtNs
	}
	return next
}

// checkedAdd returns `a + b` for non-negative `a` and `b`, or an error on
// overflow. A wrapped `reset_at` would open the rate limit. It is safe to call
// concurrently.
func checkedAdd(a, b int64) (int64, error) {
	if a > math.MaxInt64-b {
		return 0, fault.New("sum overflows int64") //faultline:equivalent callers discard the value when there is an error
	}
	return a + b, nil
}
