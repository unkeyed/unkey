package bucket

import (
	"cmp"
	"math"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Session is one pod's record of the time it admitted for one rate limit,
// counted from `AnchorAtNs` on. It is one row of the `ratelimit_state` table,
// or one of this pod's sessions in memory.
type Session struct {
	// ID is unique per session. The pod creates it together with the session
	// and never uses it again for another session.
	ID string

	// AnchorAtNs is the fixed time from which this session's consumed time
	// counts, in Unix nanoseconds. It never changes. It can be ahead of the
	// pod's own clock when a peer had a later anchor; see [Bucket.Check].
	AnchorAtNs int64

	// ConsumedNs is the consumed time of the units this session admitted. It
	// only grows. When the caller holds two copies of the same session, for
	// example its own memory and a database row, it must keep the larger
	// value; an older value must never replace a newer one.
	ConsumedNs int64
}

// Bucket is the state of one rate limit, seen by the pod that owns its live
// session. It does not keep the sessions. It keeps only the values that
// [Bucket.Check] needs, so a decision is O(1) after an O(n log n) build.
//
// Units added to the live session raise the candidates of the sessions
// anchored at or before it, and no other candidate. So besides `reset_at`,
// the largest candidate of all, the Bucket keeps the largest candidate among
// those sessions. When the two are equal, the live session can take a request
// and move `reset_at` exactly as single-machine GCRA does.
//
// A Bucket is a value without locks. The caller serializes access.
type Bucket struct {
	// resetAtNs is `reset_at`: the largest candidate of all sessions. It is
	// zero for a rate limit without sessions.
	resetAtNs int64

	// liveSessionAnchorAtNs is the anchor of this pod's live session. It is
	// zero when the pod has no live session, and then the next admitted
	// request starts a new session. Zero is free to mean "no live session"
	// because every anchor is positive: [New] rejects other anchors and
	// [Bucket.Check] requires a positive time.
	liveSessionAnchorAtNs int64

	// maxCandidateRaisedByLiveSessionNs is the largest candidate among the
	// sessions anchored at or before the live session, the live session
	// included. These are exactly the candidates that units added to the
	// live session raise. It is zero when there is no live session.
	maxCandidateRaisedByLiveSessionNs int64

	// newestAnchorAtNs is the largest anchor of all sessions. A new session
	// is never anchored before it, so that every other session counts the new
	// session's consumed time.
	newestAnchorAtNs int64
}

// Decision is the result of [Bucket.Check]. Pass it to [Bucket.Apply] to
// commit it, and write it into the sessions as described on
// `CreateNewSessionAtNs`.
type Decision struct {
	Allowed bool

	// ConsumedNs is the consumed time of the request. It is zero if the
	// request was denied or had zero `units`, and then the caller writes
	// nothing.
	ConsumedNs int64

	// ResetAtNs is `reset_at` after the request, in Unix nanoseconds: the
	// time when every admitted unit is paid back and a request of `Limit`
	// units fits again. It is in the past when the rate limit is idle.
	ResetAtNs int64

	// Remaining is the largest number of units that one more request could
	// use at the same time. For a denied request it is how many fit now.
	Remaining int64

	// CreateNewSessionAtNs tells the caller where to write `ConsumedNs`. It
	// is only relevant when `ConsumedNs` is positive:
	//
	// If it is zero, the caller adds `ConsumedNs` to the live session.
	//
	// Otherwise the live session cannot take the request. The caller creates
	// a new session with a new ID, anchored at `CreateNewSessionAtNs`, writes
	// `ConsumedNs` into it, makes it the live session, and never writes to
	// the old live session again.
	//
	// Zero is free to mean "no new session" because [Bucket.Check] requires
	// a positive time and a new anchor is never before it.
	CreateNewSessionAtNs int64
}

// New computes the state of a rate limit from all its known sessions.
// `liveSessionID` names this pod's live session; it must be in `sessions`.
// Pass an empty `liveSessionID` if this pod has no live session.
//
// Each session ID must be in `sessions` only once. The caller merges its own
// sessions with the database rows first and keeps the larger `ConsumedNs`.
//
// New returns an error if an anchor is not positive, if a consumed time is
// negative, if a sum overflows int64, or if the live session is missing or
// duplicated. It does not change `sessions`.
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

	var consumedAtOrAfterAnchorNs int64
	for start := 0; start < len(newestFirst); {
		anchorNs := newestFirst[start].AnchorAtNs
		end := start
		// Sessions with the same anchor count toward each other, so all of
		// them are added before their candidate is computed.
		for ; end < len(newestFirst) && newestFirst[end].AnchorAtNs == anchorNs; end++ {
			var err error
			consumedAtOrAfterAnchorNs, err = addNs(consumedAtOrAfterAnchorNs, newestFirst[end].ConsumedNs)
			if err != nil {
				return Bucket{}, err
			}
		}
		candidateNs, err := addNs(anchorNs, consumedAtOrAfterAnchorNs)
		if err != nil {
			return Bucket{}, err
		}
		b.resetAtNs = max(b.resetAtNs, candidateNs)
		if b.liveSessionAnchorAtNs != 0 && anchorNs <= b.liveSessionAnchorAtNs {
			b.maxCandidateRaisedByLiveSessionNs = max(b.maxCandidateRaisedByLiveSessionNs, candidateNs)
		}
		start = end
	}
	if len(newestFirst) > 0 {
		b.newestAnchorAtNs = newestFirst[0].AnchorAtNs
	}
	return b, nil
}

// ResetAtNs returns `reset_at` in Unix nanoseconds: the time when every
// admitted unit is paid back and a request of `Limit` units fits again. It is
// zero for a rate limit without sessions.
func (b Bucket) ResetAtNs() int64 {
	return b.resetAtNs
}

// Check decides a request of `units` units at `nowNs` without changing the
// Bucket. An allowed request moves `reset_at` to
// `max(reset_at, nowNs) + consumed time of units`, the same as single-machine
// GCRA. A request fits if `reset_at` then stays at most one `rate.Duration`
// ahead of `nowNs`. More `units` than `rate.Limit` never fit.
//
// The request goes into the live session if that moves `reset_at` the same
// way. Otherwise [Decision.CreateNewSessionAtNs] tells the caller to start a
// new session. The package documentation explains when and why.
//
// Check returns an error if `rate` or `units` is invalid, if `nowNs` is not
// positive, or if the new `reset_at` does not fit into int64.
func (b Bucket) Check(nowNs int64, rate Rate, units int64) (Decision, error) {
	err := assert.All(
		rate.Validate(),
		assert.GreaterOrEqual(units, 0, "units must not be negative"),
		assert.Greater(nowNs, 0, "time must be after the Unix epoch"),
	)
	if err != nil {
		return Decision{}, err
	}

	// A denied request returns the state as it is. Nothing is written, so a
	// caller who keeps hitting a closed rate limit costs no database writes.
	unchanged := Decision{
		Allowed:              false,
		ConsumedNs:           0,
		ResetAtNs:            b.resetAtNs,
		Remaining:            rate.remaining(nowNs, b.resetAtNs),
		CreateNewSessionAtNs: 0,
	}
	// At most `Limit` units are ever free, so more `units` never fit.
	// Checking it here also keeps the 128-bit division in `unitsToNs` from
	// panicking.
	if units > rate.Limit {
		return unchanged, nil
	}
	consumedNs := rate.unitsToNs(units)
	// This is the single-machine GCRA rule. If `reset_at` is in the past,
	// every unit is free, and the request pays from now, not from the old
	// `reset_at`. Otherwise the request waits behind what is already in use.
	nextResetAtNs, err := addNs(max(b.resetAtNs, nowNs), consumedNs)
	if err != nil {
		return Decision{}, err
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
		unchanged.Allowed = true
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
	// `reset_at` stays 210: the request would be free. Single-machine GCRA
	// gives 211. A new session anchored at 155 has candidate 155 + 1 = 156,
	// and the peer session now counts it too: 150 + 60 + 1 = 211.
	liveSessionCanTakeRequest := b.liveSessionAnchorAtNs != 0 &&
		b.maxCandidateRaisedByLiveSessionNs == b.resetAtNs &&
		b.resetAtNs >= nowNs
	var createNewSessionAtNs int64
	if !liveSessionCanTakeRequest {
		// Anchored at or after the newest anchor, the new session is counted
		// by every other session. The newest anchor can be ahead of `nowNs`
		// when the clocks of two pods differ a little.
		createNewSessionAtNs = max(nowNs, b.newestAnchorAtNs)
	}
	return Decision{
		Allowed:              true,
		ConsumedNs:           consumedNs,
		ResetAtNs:            nextResetAtNs,
		Remaining:            rate.remaining(nowNs, nextResetAtNs),
		CreateNewSessionAtNs: createNewSessionAtNs,
	}, nil
}

// Apply commits a decision to the Bucket. The decision must come from
// [Bucket.Check] on this Bucket with no change in between. Apply updates only
// the Bucket; the caller must also write the decision into its sessions, as
// described on [Decision.CreateNewSessionAtNs]. Apply does nothing for a
// decision with zero `ConsumedNs`.
func (b *Bucket) Apply(decision Decision) {
	if !decision.Allowed || decision.ConsumedNs == 0 {
		return
	}
	b.resetAtNs = decision.ResetAtNs
	// A new session is anchored at or after every other session, so it
	// raises every candidate. In both cases the largest raised candidate is
	// the new `reset_at`. The old live session keeps counting in full.
	b.maxCandidateRaisedByLiveSessionNs = decision.ResetAtNs
	if decision.CreateNewSessionAtNs != 0 {
		b.liveSessionAnchorAtNs = decision.CreateNewSessionAtNs
		b.newestAnchorAtNs = decision.CreateNewSessionAtNs
	}
}

// EndLiveSession stops using the live session, for example when its
// lifetime ended. The session keeps counting in full. The next admitted
// request starts a new session. `reset_at` does not change.
func (b *Bucket) EndLiveSession() {
	b.liveSessionAnchorAtNs = 0
	b.maxCandidateRaisedByLiveSessionNs = 0
}

// addNs returns `a + b` for non-negative `a` and `b`, or an error if the sum
// does not fit into int64. The limiter must never wrap around, because a
// wrapped `reset_at` would open the rate limit.
func addNs(a, b int64) (int64, error) {
	if a > math.MaxInt64-b {
		return 0, fault.New("time overflows int64 nanoseconds")
	}
	return a + b, nil
}
