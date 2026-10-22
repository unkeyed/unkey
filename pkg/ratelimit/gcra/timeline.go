package gcra

import (
	"cmp"
	"math"
	"slices"

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

// Timeline is the state of one limit, seen by the pod that owns its live
// session. It does not keep the sessions. It keeps only the values that
// [Timeline.Check] needs, so a decision is O(1) after an O(n log n) build.
//
// The candidate of a session r is r's anchor plus the consumed time of every
// session anchored at or after r. reset_at is the largest candidate.
//
// A request that goes into the live session raises the candidates of the
// sessions anchored at or before it, and of no other session. That is why the
// Timeline keeps two maxima: one for the sessions at or before the live anchor
// and one for the sessions after it.
//
// A Timeline is a value without locks. The caller serializes access.
type Timeline struct {
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

// Decision is the result of [Timeline.Check]. Pass it to [Timeline.Apply] to
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

	// StartsSession reports that the request must go into a new session
	// anchored at SessionAnchorAtNs. The caller gives that session a new ID and
	// does not use the old live session again.
	StartsSession     bool
	SessionAnchorAtNs int64
}

// NewTimeline computes the state of a limit from all its known sessions.
// liveSessionID names this pod's live session; it must be in sessions. Pass
// an empty liveSessionID if this pod has no live session.
//
// Each session ID must be in sessions only once. The caller merges its own
// sessions with the database rows first and keeps the larger ConsumedNs.
//
// NewTimeline returns an error if a value is negative, if a sum overflows
// int64, or if the live session is missing or duplicated. It does not change
// sessions.
func NewTimeline(sessions []Session, liveSessionID string) (Timeline, error) {
	var timeline Timeline
	newestFirst := slices.Clone(sessions)
	slices.SortFunc(newestFirst, func(a, b Session) int {
		return cmp.Compare(b.AnchorAtNs, a.AnchorAtNs)
	})

	liveCount := 0
	for _, session := range newestFirst {
		if session.AnchorAtNs < 0 || session.ConsumedNs < 0 {
			return Timeline{}, fault.New("session values must not be negative")
		}
		if liveSessionID != "" && session.ID == liveSessionID {
			liveCount++
			timeline.liveAnchorNs = session.AnchorAtNs
		}
	}
	if liveSessionID != "" && liveCount != 1 {
		return Timeline{}, fault.New("live session must be in sessions exactly once")
	}
	timeline.hasLive = liveSessionID != ""

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
				return Timeline{}, err
			}
		}
		candidateNs, err := addNs(anchorNs, usedNs)
		if err != nil {
			return Timeline{}, err
		}
		if timeline.hasLive && anchorNs > timeline.liveAnchorNs {
			timeline.newerMaxNs = max(timeline.newerMaxNs, candidateNs)
		} else {
			timeline.olderMaxNs = max(timeline.olderMaxNs, candidateNs)
		}
		start = end
	}
	if len(newestFirst) > 0 {
		timeline.newestAnchorNs = newestFirst[0].AnchorAtNs
	}
	return timeline, nil
}

// ResetAtNs returns reset_at in Unix nanoseconds: the time when every unit is
// free again. It is zero for a limit without sessions.
func (t Timeline) ResetAtNs() int64 {
	return max(t.olderMaxNs, t.newerMaxNs)
}

// Check decides a request of cost units at nowNs without changing the
// Timeline. An allowed request moves reset_at to max(reset_at, now) + cost,
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
func (t Timeline) Check(nowNs int64, rate Rate, cost int64) (Decision, error) {
	if err := rate.Validate(); err != nil {
		return Decision{}, err
	}
	if cost < 0 {
		return Decision{}, fault.New("cost must not be negative")
	}
	if nowNs < 0 {
		return Decision{}, fault.New("time must not be before the Unix epoch")
	}

	resetAtNs := t.ResetAtNs()
	unchanged := Decision{
		Allowed:           false,
		CostNs:            0,
		ResetAtNs:         resetAtNs,
		Remaining:         rate.remaining(nowNs, resetAtNs),
		StartsSession:     false,
		SessionAnchorAtNs: 0,
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

	startsSession := !t.hasLive || t.olderMaxNs < max(t.newerMaxNs, nowNs)
	var sessionAnchorAtNs int64
	if startsSession {
		sessionAnchorAtNs = max(nowNs, t.newestAnchorNs)
	}
	return Decision{
		Allowed:           true,
		CostNs:            costNs,
		ResetAtNs:         nextResetAtNs,
		Remaining:         rate.remaining(nowNs, nextResetAtNs),
		StartsSession:     startsSession,
		SessionAnchorAtNs: sessionAnchorAtNs,
	}, nil
}

// Apply commits a decision. The decision must come from [Timeline.Check] on
// this Timeline with no change in between. Apply does nothing for a decision
// that uses no time.
func (t *Timeline) Apply(decision Decision) {
	if !decision.Allowed || decision.CostNs == 0 {
		return
	}
	// In both cases the request raises exactly the candidates at or before the
	// live anchor, and the largest of them becomes the new reset_at.
	t.olderMaxNs = decision.ResetAtNs
	if decision.StartsSession {
		t.hasLive = true
		t.liveAnchorNs = decision.SessionAnchorAtNs
		t.newerMaxNs = 0
		t.newestAnchorNs = decision.SessionAnchorAtNs
	}
}

// EndLiveSession freezes the live session, for example when its lifetime
// ended. The next admitted request starts a new session. reset_at does not
// change.
func (t *Timeline) EndLiveSession() {
	t.olderMaxNs = t.ResetAtNs()
	t.newerMaxNs = 0
	t.hasLive = false
	t.liveAnchorNs = 0
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
