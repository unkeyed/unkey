// Package gcra holds the math of the global rate limiter: GCRA (Generic Cell
// Rate Algorithm) over the sessions of the ratelimit_state table.
//
// GCRA stores a limit as one point in time, reset_at: the time when every unit
// is free again. A request of cost units uses ⌈cost × Duration / Limit⌉
// nanoseconds and fits if reset_at, moved by that time, stays at most one
// Duration ahead of now.
//
// Many pods share a limit, so no single row holds reset_at. Each session of a
// pod has its own row with an anchor and the time it consumed, and
//
//	reset_at = max over sessions r of
//	           (r.anchor + sum of consumed of sessions anchored at or after r)
//
// # Exact cost arithmetic
//
// The cost of a request is ⌈cost × Duration / Limit⌉ nanoseconds. Do not
// replace the 128-bit math in [Rate] with plain int64 or uint64 multiplication.
// The product cost × Duration can reach Limit × Duration, and that does not fit
// into 64 bits for rates the API accepts:
//
//	rate                 Limit × Duration (ns)   fits into uint64 (max 1.8e19)
//	10,000 per 30 days   2.6e19                  no
//	400M per minute      2.4e19                  no
//
// A product that wraps around gives a small or negative cost, and the limiter
// admits too much. Dividing first, as in Duration / Limit × cost, loses
// precision: 162M per minute is 370.37 ns per unit, and 370 ns would let the
// full limit through in less than a minute. [math/big] is exact but allocates
// on every decision.
//
// [bits.Mul64] keeps the full 128-bit product and [bits.Div64] divides it, so
// the result is exact for every Limit and Duration that fit into int64, with
// no allocation. It costs about 1 ns more per decision than plain division on
// amd64. bits.Div64 panics if the divisor is zero or if the quotient does not
// fit into 64 bits; [Rate.Validate] and the rule cost ≤ Limit in
// [Timeline.Check] exclude both cases, so keep them in front of every call.
//
// The package has no I/O, no clock and no locks. The caller reads the
// sessions, passes the time, and serializes access, so the same code runs in
// the limiter and in simulations.
//
//	timeline, err := gcra.NewTimeline(sessions, liveSessionID)
//	if err != nil {
//		return err
//	}
//	decision, err := timeline.Check(nowNs, gcra.Rate{Limit: 100, Duration: time.Minute}, 1)
//	if err != nil {
//		return err
//	}
//	timeline.Apply(decision)
//	if decision.StartsSession {
//		// Write decision.CostNs into a new session anchored at
//		// decision.SessionAnchorAtNs.
//	}
package gcra
