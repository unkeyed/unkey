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
	// pod's own clock when a peer had a later anchor; see [Sessions.Check].
	AnchorAtNs int64

	// ConsumedNs is the consumed time of the units this session admitted. It
	// only grows.
	ConsumedNs int64
}

// Sessions holds all known sessions of one rate limit, as seen by one pod,
// and caches the values that a decision needs. Sessions change only through
// its methods, so the cached values always match the sessions:
// [Sessions.MergeRows] recomputes them in one pass, and [Sessions.Spend]
// updates them in O(1). A decision is O(1).
//
// The zero value holds no sessions and is ready to use. A Sessions has no
// locks; the caller serializes access.
type Sessions struct {
	// live is this pod's live session: the session that takes its admitted
	// units. Its `ID` is empty when the pod has no live session, and then
	// the next admitted request starts a new session.
	live Session

	// others are all other sessions: the sessions of peers and this pod's
	// frozen sessions. They are in no particular order.
	others []Session

	// resetAtNs is `reset_at`: the largest candidate of all sessions. It is
	// zero without sessions.
	resetAtNs int64

	// maxCandidateRaisedByLiveSessionNs is the largest candidate among the
	// sessions anchored at or before the live session, the live session
	// included. These are exactly the candidates that units added to the
	// live session raise. It is zero without a live session.
	maxCandidateRaisedByLiveSessionNs int64

	// newestAnchorAtNs is the largest anchor of all sessions. A new session
	// is never anchored before it, so that every other session counts the new
	// session's consumed time.
	newestAnchorAtNs int64
}

// Decision is the result of [Sessions.Check]. Pass it to [Sessions.Spend] to
// commit it.
type Decision struct {
	Allowed bool

	// ConsumedNs is the consumed time of the request. It is zero if the
	// request was denied or had zero `units`, and then [Sessions.Spend]
	// changes nothing.
	ConsumedNs int64

	// ResetAtNs is `reset_at` after the request, in Unix nanoseconds: the
	// time when every admitted unit is paid back and a request of `Limit`
	// units fits again. It is in the past when the rate limit is idle.
	ResetAtNs int64

	// Remaining is the largest number of units that one more request could
	// use at the same time. For a denied request it is how many fit now.
	Remaining int64

	// CreateNewSessionAtNs is zero when the request goes into the live
	// session. Otherwise the live session cannot take the request, and
	// [Sessions.Spend] creates a new live session anchored at
	// `CreateNewSessionAtNs`. The old live session is frozen and keeps
	// counting in full.
	//
	// Zero is free to mean "no new session" because [Sessions.Check] requires
	// a positive time and a new anchor is never before it.
	CreateNewSessionAtNs int64
}

// MergeRows adds rows read from the database. For a session that is already
// known, it keeps the larger `ConsumedNs`, so a stale replica never moves a
// session back. Then it recomputes the cached values in one pass, O(n log n)
// for n sessions.
//
// MergeRows never removes a session. It returns an error, and changes
// nothing, if a row has an anchor that is not positive, a negative consumed
// time, or a known ID with a different anchor, or if a sum overflows int64.
func (s *Sessions) MergeRows(rows []Session) error {
	next := *s
	next.others = slices.Clone(s.others)

	indexByID := make(map[string]int, len(next.others))
	for index, session := range next.others {
		indexByID[session.ID] = index
	}
	for _, row := range rows {
		err := assert.All(
			assert.NotEmpty(row.ID, "session ID must not be empty"),
			assert.Greater(row.AnchorAtNs, 0, "session anchor must be positive"),
			assert.GreaterOrEqual(row.ConsumedNs, 0, "session consumed time must not be negative"),
		)
		if err != nil {
			return err
		}
		known := &next.live
		if row.ID != next.live.ID {
			index, ok := indexByID[row.ID]
			if !ok {
				indexByID[row.ID] = len(next.others)
				next.others = append(next.others, row)
				continue
			}
			known = &next.others[index]
		}
		err = assert.Equal(known.AnchorAtNs, row.AnchorAtNs, "the anchor of a session never changes")
		if err != nil {
			return err
		}
		known.ConsumedNs = max(known.ConsumedNs, row.ConsumedNs)
	}

	err := next.recompute()
	if err != nil {
		return err
	}
	*s = next
	return nil
}

// ResetAtNs returns `reset_at` in Unix nanoseconds: the time when every
// admitted unit is paid back and a request of `Limit` units fits again. It is
// zero without sessions.
func (s *Sessions) ResetAtNs() int64 {
	return s.resetAtNs
}

// All returns a copy of all sessions, the live session included, in no
// particular order.
func (s *Sessions) All() []Session {
	all := slices.Clone(s.others)
	if s.live.ID != "" {
		all = append(all, s.live)
	}
	return all
}

// Check decides a request of `units` units at `nowNs` without changing
// anything. An allowed request moves `reset_at` to
// `max(reset_at, nowNs) + consumed time of units`, the same as single-machine
// GCRA. A request fits if `reset_at` then stays at most one `rate.Duration`
// ahead of `nowNs`. More `units` than `rate.Limit` never fit.
//
// Check is separate from [Sessions.Spend] so that a caller can check several
// rate limits and spend on all of them or on none.
//
// Check returns an error if `rate` or `units` is invalid, if `nowNs` is not
// positive, or if the new `reset_at` does not fit into int64.
func (s *Sessions) Check(nowNs int64, rate Rate, units int64) (Decision, error) {
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
		ResetAtNs:            s.resetAtNs,
		Remaining:            rate.remaining(nowNs, s.resetAtNs),
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
	nextResetAtNs, err := addNs(max(s.resetAtNs, nowNs), consumedNs)
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
	liveSessionCanTakeRequest := s.live.ID != "" &&
		s.maxCandidateRaisedByLiveSessionNs == s.resetAtNs &&
		s.resetAtNs >= nowNs
	var createNewSessionAtNs int64
	if !liveSessionCanTakeRequest {
		// Anchored at or after the newest anchor, the new session is counted
		// by every other session. The newest anchor can be ahead of `nowNs`
		// when the clocks of two pods differ a little.
		createNewSessionAtNs = max(nowNs, s.newestAnchorAtNs)
	}
	return Decision{
		Allowed:              true,
		ConsumedNs:           consumedNs,
		ResetAtNs:            nextResetAtNs,
		Remaining:            rate.remaining(nowNs, nextResetAtNs),
		CreateNewSessionAtNs: createNewSessionAtNs,
	}, nil
}

// Spend commits a decision: it adds `decision.ConsumedNs` to the live
// session, or creates a new live session with `newSessionID` when
// `decision.CreateNewSessionAtNs` is set. The decision must come from
// [Sessions.Check] on these sessions with no change in between. Spend does
// nothing for a decision with zero `ConsumedNs`.
//
// `newSessionID` must be a new, unique ID; it is only used when a new
// session starts. Spend returns an error, and changes nothing, if a new
// session is needed and `newSessionID` is empty or already known.
func (s *Sessions) Spend(decision Decision, newSessionID string) error {
	if !decision.Allowed || decision.ConsumedNs == 0 {
		return nil
	}
	if decision.CreateNewSessionAtNs == 0 {
		s.live.ConsumedNs += decision.ConsumedNs
	} else {
		err := assert.All(
			assert.NotEmpty(newSessionID, "new session ID must not be empty"),
			assert.False(s.isKnown(newSessionID), "new session ID must not be known"),
		)
		if err != nil {
			return err
		}
		s.EndLiveSession()
		s.live = Session{
			ID:         newSessionID,
			AnchorAtNs: decision.CreateNewSessionAtNs,
			ConsumedNs: decision.ConsumedNs,
		}
		s.newestAnchorAtNs = decision.CreateNewSessionAtNs
	}
	// The request raised every candidate at or before the live session. A new
	// session is anchored at or after every other session, so there it raised
	// every candidate. Either way, the largest raised candidate is the new
	// `reset_at`.
	s.resetAtNs = decision.ResetAtNs
	s.maxCandidateRaisedByLiveSessionNs = decision.ResetAtNs
	return nil
}

// EndLiveSession freezes the live session, for example when its lifetime
// ended. It keeps counting in full, and the next admitted request starts a
// new session. `reset_at` does not change.
func (s *Sessions) EndLiveSession() {
	if s.live.ID == "" {
		return
	}
	s.others = append(s.others, s.live)
	s.live = Session{ID: "", AnchorAtNs: 0, ConsumedNs: 0}
	s.maxCandidateRaisedByLiveSessionNs = 0
}

// recompute sets the cached values from the sessions, with the formula from
// the package documentation.
func (s *Sessions) recompute() error {
	// The candidate of a session needs the sum of everything anchored at or
	// after it. Walking from the newest anchor to the oldest builds that sum
	// as a running total, so all candidates take one pass.
	newestFirst := s.All()
	slices.SortFunc(newestFirst, func(left, right Session) int {
		return cmp.Compare(right.AnchorAtNs, left.AnchorAtNs)
	})

	s.resetAtNs = 0
	s.maxCandidateRaisedByLiveSessionNs = 0
	s.newestAnchorAtNs = 0
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
				return err
			}
		}
		candidateNs, err := addNs(anchorNs, consumedAtOrAfterAnchorNs)
		if err != nil {
			return err
		}
		s.resetAtNs = max(s.resetAtNs, candidateNs)
		if s.live.ID != "" && anchorNs <= s.live.AnchorAtNs {
			s.maxCandidateRaisedByLiveSessionNs = max(s.maxCandidateRaisedByLiveSessionNs, candidateNs)
		}
		start = end
	}
	if len(newestFirst) > 0 {
		s.newestAnchorAtNs = newestFirst[0].AnchorAtNs
	}
	return nil
}

// isKnown reports whether a session with `id` exists. It is O(n), but it
// only runs when a new session starts.
func (s *Sessions) isKnown(id string) bool {
	return id == s.live.ID || slices.ContainsFunc(s.others, func(session Session) bool {
		return session.ID == id
	})
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
