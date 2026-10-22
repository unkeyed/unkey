package bucket

import (
	"cmp"
	"math"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Session is what one session admitted for one limit: one row of the
// ratelimit_state table, or one of this pod's sessions in memory.
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
// session anchored at or after r. reset_at is the largest candidate.
//
// A request that goes into the live session raises the candidates of the
// sessions anchored at or before it, and of no other session. That is why the
// Bucket keeps two maxima: one for the sessions at or before the live anchor
// and one for the sessions after it.
//
// A Bucket is a value without locks. The caller serializes access.
type Bucket struct {
	// hasLive reports whether this pod has a live session that can take
	// requests. Without one, the next admitted request starts a new session.
	hasLive bool

	// liveAnchorNs is the anchor of the live session, if hasLive.
	liveAnchorNs int64

	// olderMaxNs is the largest candidate of the sessions anchored at or
	// before the live anchor, the live session included. Without a live
	// session it covers every session, so it is reset_at.
	olderMaxNs int64

	// newerMaxNs is the largest candidate of the sessions anchored after the
	// live anchor. It is zero if there are none.
	newerMaxNs int64

	// newestAnchorNs is the largest anchor of all sessions. A new session never
	// starts before it, so the new session counts toward every other session.
	newestAnchorNs int64
}

// Decision is the result of [Bucket.Check]. Pass it to [Bucket.Apply] to
// commit it.
type Decision struct {
	Allowed bool

	// CostNs is the time the request uses. It is zero if the request was
	// denied or had cost zero, and then the request changes nothing.
	CostNs int64

	// ResetAtNs is reset_at after the request, in Unix nanoseconds: the time
	// when every unit is free again. It can be in the past when the limit is
	// idle.
	ResetAtNs int64

	// Remaining is how many more units fit after the request.
	Remaining int64

	// CreateNewSessionAtNs is zero when the request goes into the live
	// session or changes nothing. Otherwise the request cannot go into the
	// live session: the caller creates a new session with a new ID, anchored
	// at CreateNewSessionAtNs, writes CostNs into it, and does not use the old
	// live session again. Zero is free to mean "no new session" because
	// [Bucket.Check] requires a positive time and the anchor is never before it.
	CreateNewSessionAtNs int64
}

// New computes the state of a limit from all its known sessions.
// liveSessionID names this pod's live session; it must be in sessions. Pass
// an empty liveSessionID if this pod has no live session.
//
// Each session ID must be in sessions only once. The caller merges its own
// sessions with the database rows first and keeps the larger ConsumedNs.
//
// New returns an error if a value is negative, if a sum overflows
// int64, or if the live session is missing or duplicated. It does not change
// sessions.
func New(sessions []Session, liveSessionID string) (Bucket, error) {
	var b Bucket
	newestFirst := slices.Clone(sessions)
	slices.SortFunc(newestFirst, func(left, right Session) int {
		return cmp.Compare(right.AnchorAtNs, left.AnchorAtNs)
	})

	liveCount := 0
	for _, session := range newestFirst {
		if session.AnchorAtNs < 0 || session.ConsumedNs < 0 {
			return Bucket{}, fault.New("session values must not be negative")
		}
		if liveSessionID != "" && session.ID == liveSessionID {
			liveCount++
			b.liveAnchorNs = session.AnchorAtNs
		}
	}
	if liveSessionID != "" && liveCount != 1 {
		return Bucket{}, fault.New("live session must be in sessions exactly once")
	}
	b.hasLive = liveSessionID != ""

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
		if b.hasLive && anchorNs > b.liveAnchorNs {
			b.newerMaxNs = max(b.newerMaxNs, candidateNs)
		} else {
			b.olderMaxNs = max(b.olderMaxNs, candidateNs)
		}
		start = end
	}
	if len(newestFirst) > 0 {
		b.newestAnchorNs = newestFirst[0].AnchorAtNs
	}
	return b, nil
}

// ResetAtNs returns reset_at in Unix nanoseconds: the time when every unit is
// free again. It is zero for a limit without sessions.
func (b Bucket) ResetAtNs() int64 {
	return max(b.olderMaxNs, b.newerMaxNs)
}

// Check decides a request of cost units at nowNs without changing the
// Bucket. An allowed request moves reset_at to max(reset_at, now) + cost,
// the same as single-machine GCRA. A request fits if reset_at then stays at
// most one Duration ahead of now. A cost above the limit never fits.
//
// The request goes into the live session if that gives the same reset_at.
// That is the case when the largest candidate is at or before the live anchor
// and not in the past. Otherwise the request starts a new session, anchored at
// now or at the newest anchor, whichever is later. This covers a pause (reset_at
// in the past) and a peer session that started after the live one.
//
// Check returns an error if the rate or the cost is invalid, if nowNs is
// negative, or if reset_at overflows int64.
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
	unchanged := Decision{
		Allowed:              false,
		CostNs:               0,
		ResetAtNs:            resetAtNs,
		Remaining:            rate.remaining(nowNs, resetAtNs),
		CreateNewSessionAtNs: 0,
	}
	if cost > rate.Limit {
		return unchanged, nil
	}
	costNs := rate.costNs(cost)
	nextResetAtNs, err := addNs(max(resetAtNs, nowNs), costNs)
	if err != nil {
		return Decision{}, err
	}
	if nextResetAtNs-nowNs > int64(rate.Duration) {
		return unchanged, nil
	}
	if costNs == 0 {
		unchanged.Allowed = true
		return unchanged, nil
	}

	var createNewSessionAtNs int64
	if !b.hasLive || b.olderMaxNs < max(b.newerMaxNs, nowNs) {
		createNewSessionAtNs = max(nowNs, b.newestAnchorNs)
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
	// live anchor, and the largest of them becomes the new reset_at.
	b.olderMaxNs = decision.ResetAtNs
	if decision.CreateNewSessionAtNs != 0 {
		b.hasLive = true
		b.liveAnchorNs = decision.CreateNewSessionAtNs
		b.newerMaxNs = 0
		b.newestAnchorNs = decision.CreateNewSessionAtNs
	}
}

// EndLiveSession freezes the live session, for example when its lifetime
// ended. The next admitted request starts a new session. reset_at does not
// change.
func (b *Bucket) EndLiveSession() {
	b.olderMaxNs = b.ResetAtNs()
	b.newerMaxNs = 0
	b.hasLive = false
	b.liveAnchorNs = 0
}

// addNs returns a + b for non-negative a and b, or an error if the sum does
// not fit into int64. The limiter must never wrap around, because a wrapped
// reset_at would open the limit.
func addNs(a, b int64) (int64, error) {
	if a > math.MaxInt64-b {
		return 0, fault.New("time overflows int64 nanoseconds")
	}
	return a + b, nil
}
