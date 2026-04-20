package clock

import "time"

// MonotonicClock returns timestamps whose interval deltas are driven by a
// monotonic clock but whose wall-clock value is anchored at construction.
//
// Use this where wall-clock ts is needed for display or bucketing, but the
// intervals between successive Now() calls MUST never go backward — e.g.
// billing interval math, where a wall-clock backward jump (NTP step) would
// produce negative durations.
//
// Drift: the wall anchor is never re-synced. Over long-running processes
// (weeks+), the returned timestamps diverge from real wall time by whatever
// NTP correction occurred after construction. Typical drift is sub-second
// per day; process restart re-anchors.
type MonotonicClock struct {
	// anchor carries both wall + monotonic readings (from time.Now()).
	// time.Time.Add preserves monotonic; time.Time.Sub between two Times
	// with monotonic readings uses the monotonic delta.
	anchor time.Time
}

// NewMonotonic returns a MonotonicClock anchored at the current moment.
func NewMonotonic() *MonotonicClock {
	return &MonotonicClock{anchor: time.Now()}
}

// Ensure MonotonicClock implements the Clock interface.
var _ Clock = (*MonotonicClock)(nil)

// Now returns wall_at_anchor + monotonic_elapsed_since_anchor. Consecutive
// calls never produce a regressing timestamp, even across NTP step corrections
// that move the system wall clock backward.
func (c *MonotonicClock) Now() time.Time {
	return c.anchor.Add(time.Since(c.anchor))
}
