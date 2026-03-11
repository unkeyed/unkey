// Package timer provides a simple helper for measuring elapsed time in seconds,
// designed for use with Prometheus histogram observations.
package timer

import "time"

// Timer captures a start time for measuring elapsed duration.
type Timer struct {
	start time.Time
}

// New creates a Timer that starts counting from now.
func New() Timer {
	return Timer{start: time.Now()}
}

// Seconds returns the elapsed time since the timer was created, in seconds.
// This value is suitable for passing directly to prometheus Histogram.Observe().
func (t Timer) Seconds() float64 {
	return time.Since(t.start).Seconds()
}
