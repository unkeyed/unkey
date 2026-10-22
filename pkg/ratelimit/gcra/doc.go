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
