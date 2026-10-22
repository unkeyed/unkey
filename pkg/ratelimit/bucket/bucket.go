package bucket

import (
	"cmp"
	"math"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Session is what one session admitted for one limit: one row of the
// `ratelimit_state` table, or one of this pod's sessions in memory.
type Session struct {
	ID string

	// AnchorAtNs is when the session started, in Unix nanoseconds. It never
	// changes.
	AnchorAtNs int64

	// ConsumedNs is the time the session's admitted requests used. It only
	// grows.
	ConsumedNs int64
}

// Bucket is the state of one limit, seen by the pod that owns its live
// session. It does not keep the sessions. It keeps only the values that
// [Bucket.Check] needs, so a decision is O(1) after an O(n log n) build.
//
// The candidate of a session r is r's anchor plus the consumed time of every
// session anchored at or after r. `reset_at` is the largest candidate.
//
// A request that goes into the live session raises the candidates of the
// sessions anchored at or before it, and of no other session. That is why the
// Bucket keeps two maxima: one for the sessions at or before the live anchor
// and one for the sessions after it.
//
// A Bucket is a value without locks. The caller serializes access.
type Bucket struct {
	// liveSessionAnchorAtNs is the anchor of this pod's live session, the
	// session that takes its admitted requests. It is zero when the pod has
	// no live session, and then the next admitted request starts a new
	// session. Zero is free to mean "no live session" because every anchor is
	// positive: [New] rejects other anchors and [Bucket.Check] requires a
	// positive time.
	liveSessionAnchorAtNs int64

	// maxCandidateAtOrBeforeLiveSessionNs is the largest candidate of the
	// sessions anchored at or before the live session, the live session
	// included. A request in the live session raises exactly these
	// candidates. Without a live session it covers every session, so it is
	// `reset_at`.
	maxCandidateAtOrBeforeLiveSessionNs int64

	// maxCandidateAfterLiveSessionNs is the largest candidate of the sessions
	// anchored after the live session. A request in the live session does not
	// raise these candidates. It is zero if there are none.
	maxCandidateAfterLiveSessionNs int64

	// newestAnchorAtNs is the largest anchor of all sessions. A new session
	// never starts before it, so the new session counts toward every other
	// session.
	newestAnchorAtNs int64
}

// Decision is the result of [Bucket.Check]. Pass it to [Bucket.Apply] to
// commit it.
type Decision struct {
	Allowed bool

	// CostNs is the time the request uses. It is zero if the request was
	// denied or had `cost` zero, and then the request changes nothing.
	CostNs int64

	// ResetAtNs is `reset_at` after the request, in Unix nanoseconds: the time
	// when every unit is free again. It can be in the past when the limit is
	// idle.
	ResetAtNs int64

	// Remaining is how many more units fit after the request.
	Remaining int64

	// CreateNewSessionAtNs is zero when the request goes into the live
	// session or changes nothing. Otherwise the request cannot go into the
	// live session: the caller creates a new session with a new ID, anchored
	// at `CreateNewSessionAtNs`, writes `CostNs` into it, and does not use the old
	// live session again. Zero is free to mean "no new session" because
	// [Bucket.Check] requires a positive time and the anchor is never before it.
	CreateNewSessionAtNs int64
}

// New computes the state of a limit from all its known sessions.
// `liveSessionID` names this pod's live session; it must be in `sessions`.
// Pass an empty `liveSessionID` if this pod has no live session.
//
// Each session ID must be in `sessions` only once. The caller merges its own
// sessions with the database rows first and keeps the larger `ConsumedNs`.
//
// New returns an error if an anchor is not positive, if a consumed time is
// negative, if a sum overflows int64, or if the live session is missing or
// duplicated. It does not change
// `sessions`.
func New(sessions []Session, liveSessionID string) (Bucket, error) {
	var b Bucket
	// The candidate of a session needs the sum of everything anchored at or
	// after it. Walking from the newest anchor to the oldest builds that sum
	// as a running total, so all candidates take one pass.
	newestFirst := slices.Clone(sessions)
	slices.SortFunc(newestFirst, func(left, right Session) int {
		return cmp.Compare(right.AnchorAtNs, left.AnchorAtNs)
	})

	liveCount := 0
	for _, session := range newestFirst {
		err := assert.All(
			assert.Greater(session.AnchorAtNs, 0, "session anchor must be positive"),
			assert.GreaterOrEqual(session.ConsumedNs, 0, "session consumed time must not be negative"),
		)
		if err != nil {
			return Bucket{}, err
		}
		if liveSessionID != "" && session.ID == liveSessionID {
			liveCount++
			b.liveSessionAnchorAtNs = session.AnchorAtNs
		}
	}
	if liveSessionID != "" && liveCount != 1 {
		return Bucket{}, fault.New("live session must be in sessions exactly once")
	}

	// `usedNs` is the consumed time of every session seen so far, which is
	// every session anchored at or after the current one.
	var usedNs int64
	for start := 0; start < len(newestFirst); {
		anchorNs := newestFirst[start].AnchorAtNs
		end := start
		// Sessions with the same anchor count toward each other, so all of
		// them are added before their candidate is computed.
		for ; end < len(newestFirst) && newestFirst[end].AnchorAtNs == anchorNs; end++ {
			var err error
			usedNs, err = addNs(usedNs, newestFirst[end].ConsumedNs)
			if err != nil {
				return Bucket{}, err
			}
		}
		candidateNs, err := addNs(anchorNs, usedNs)
		if err != nil {
			return Bucket{}, err
		}
		// [Bucket.Check] must know which side of the live session holds the
		// largest candidate, because a cost added to the live session raises
		// only the candidates at or before it. So the maxima are kept apart.
		if b.liveSessionAnchorAtNs != 0 && anchorNs > b.liveSessionAnchorAtNs {
			b.maxCandidateAfterLiveSessionNs = max(b.maxCandidateAfterLiveSessionNs, candidateNs)
		} else {
			b.maxCandidateAtOrBeforeLiveSessionNs = max(b.maxCandidateAtOrBeforeLiveSessionNs, candidateNs)
		}
		start = end
	}
	// A new session anchored at or after the newest anchor counts toward
	// every session, so [Bucket.Check] never anchors one earlier.
	if len(newestFirst) > 0 {
		b.newestAnchorAtNs = newestFirst[0].AnchorAtNs
	}
	return b, nil
}

// ResetAtNs returns `reset_at` in Unix nanoseconds: the time when every unit is
// free again. It is zero for a limit without sessions.
func (b Bucket) ResetAtNs() int64 {
	// The two maxima together cover every session, so the larger of them is
	// the largest candidate of all.
	return max(b.maxCandidateAtOrBeforeLiveSessionNs, b.maxCandidateAfterLiveSessionNs)
}

// Check decides a request of `cost` units at `nowNs` without changing the
// Bucket. An allowed request moves `reset_at` to
// `max(reset_at, nowNs) + cost`, the same as single-machine GCRA. A request
// fits if `reset_at` then stays at most one `rate.Duration` ahead of `nowNs`.
// A `cost` above `rate.Limit` never fits.
//
// The request goes into the live session if that gives the same `reset_at`.
// That is the case when the largest candidate is at or before the live anchor
// and not in the past. Otherwise the request starts a new session, anchored at
// `nowNs` or at the newest anchor, whichever is later. This covers a pause
// (`reset_at` in the past) and a peer session that started after the live
// one.
//
// Check returns an error if `rate` or `cost` is invalid, if `nowNs` is not
// positive, or if `reset_at` overflows int64.
func (b Bucket) Check(nowNs int64, rate Rate, cost int64) (Decision, error) {
	err := assert.All(
		rate.Validate(),
		assert.GreaterOrEqual(cost, 0, "cost must not be negative"),
		assert.Greater(nowNs, 0, "time must be after the Unix epoch"),
	)
	if err != nil {
		return Decision{}, err
	}

	resetAtNs := b.ResetAtNs()
	// A denied request returns the state as it is. Nothing is written, so a
	// caller who keeps hitting a closed limit costs no database writes.
	unchanged := Decision{
		Allowed:              false,
		CostNs:               0,
		ResetAtNs:            resetAtNs,
		Remaining:            rate.remaining(nowNs, resetAtNs),
		CreateNewSessionAtNs: 0,
	}
	// The bucket never holds more than `Limit` units, so a larger `cost`
	// never fits. Checking it here also keeps the 128-bit division in
	// `costNs` from panicking.
	if cost > rate.Limit {
		return unchanged, nil
	}
	costNs := rate.costNs(cost)
	// This is the single-machine GCRA rule. If `reset_at` is in the past the
	// bucket is full, and the request starts paying from now, not from the
	// old `reset_at`. Otherwise the request waits behind what is already in
	// use.
	nextResetAtNs, err := addNs(max(resetAtNs, nowNs), costNs)
	if err != nil {
		return Decision{}, err
	}
	// The burst tolerance is one full `Duration`: `reset_at` may run at most
	// that far ahead of now. When pods over-admitted, `reset_at` can already
	// be further ahead, and every request is denied until time catches up.
	if nextResetAtNs-nowNs > int64(rate.Duration) {
		return unchanged, nil
	}
	// A request of `cost` zero only asks whether the limit is open. It spends
	// nothing, so it must not start a session or write a row.
	if costNs == 0 {
		unchanged.Allowed = true
		return unchanged, nil
	}

	// The request may go into the live session only if that moves `reset_at`
	// to exactly `nextResetAtNs`. A cost added to the live session raises the
	// candidates at or before it, so this holds when the largest candidate is
	// on that side and is not in the past.
	//
	// Example, 60 units per 60 ns, `nowNs` = 155:
	//
	//	live session  anchor 100, consumed 1    candidate 100 + 1 + 60 = 161
	//	peer session  anchor 150, consumed 60   candidate 150 + 60     = 210  <- reset_at
	//
	// Adding a cost of 1 to the live session makes its candidate 162, and
	// `reset_at` stays 210: the request would be free. Single-machine GCRA
	// gives 211. A new session anchored at 155 has candidate 155 + 1 = 156,
	// and the peer session now counts it too: 150 + 60 + 1 = 211.
	var createNewSessionAtNs int64
	liveSessionHoldsResetAt := b.maxCandidateAtOrBeforeLiveSessionNs >= max(b.maxCandidateAfterLiveSessionNs, nowNs)
	if b.liveSessionAnchorAtNs == 0 || !liveSessionHoldsResetAt {
		// At or after the newest anchor, the new session is counted by every
		// other session. The newest anchor can be ahead of `nowNs` when the
		// clocks of two pods differ a little.
		createNewSessionAtNs = max(nowNs, b.newestAnchorAtNs)
	}
	return Decision{
		Allowed:              true,
		CostNs:               costNs,
		ResetAtNs:            nextResetAtNs,
		Remaining:            rate.remaining(nowNs, nextResetAtNs),
		CreateNewSessionAtNs: createNewSessionAtNs,
	}, nil
}

// Apply commits a decision. The decision must come from [Bucket.Check] on
// this Bucket with no change in between. Apply does nothing for a decision
// that uses no time.
func (b *Bucket) Apply(decision Decision) {
	if !decision.Allowed || decision.CostNs == 0 {
		return
	}
	// In both cases the request raises exactly the candidates at or before the
	// live anchor, and the largest of them becomes the new `reset_at`.
	b.maxCandidateAtOrBeforeLiveSessionNs = decision.ResetAtNs
	// A new session is anchored at or after every other session, so no
	// session lies after it anymore. The old live session becomes one of the
	// older sessions and keeps counting in full.
	if decision.CreateNewSessionAtNs != 0 {
		b.liveSessionAnchorAtNs = decision.CreateNewSessionAtNs
		b.maxCandidateAfterLiveSessionNs = 0
		b.newestAnchorAtNs = decision.CreateNewSessionAtNs
	}
}

// EndLiveSession freezes the live session, for example when its lifetime
// ended. The next admitted request starts a new session. `reset_at` does not
// change.
func (b *Bucket) EndLiveSession() {
	// Without a live session, every session counts as "at or before", so the
	// largest candidate of all moves to that side. The next admitted request
	// then finds no live session and starts a new one.
	b.maxCandidateAtOrBeforeLiveSessionNs = b.ResetAtNs()
	b.maxCandidateAfterLiveSessionNs = 0
	b.liveSessionAnchorAtNs = 0
}

// addNs returns `a + b` for non-negative `a` and `b`, or an error if the sum
// does not fit into int64. The limiter must never wrap around, because a
// wrapped `reset_at` would open the limit.
func addNs(a, b int64) (int64, error) {
	if a > math.MaxInt64-b {
		return 0, fault.New("time overflows int64 nanoseconds")
	}
	return a + b, nil
}
