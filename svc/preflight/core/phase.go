package core

import "time"

// Phase is one named step inside a probe's execution. The Runner emits
// a preflight_phase_duration_seconds histogram bucket per phase,
// letting dashboards decompose "this probe was slow" into which step
// was slow.
//
// Use one phase per meaningful external call. Splitting "push commit"
// and "poll hostname" into two phases tells an oncall immediately
// which leg is the problem, without needing to trace.
type Phase struct {
	Name     string
	Duration time.Duration
	Err      error
}
