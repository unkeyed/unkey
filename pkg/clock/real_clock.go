package clock

import "time"

// RealClock implements the Clock interface using the system clock.
// This is the implementation that should be used in production code.
type RealClock struct {
	// Intentionally empty - no state needed
}

// New creates a new RealClock that uses the system time.
// This is the standard clock implementation for production use.
//
// Example:
//
//	clock := clock.New()
//	currentTime := clock.Now()
func New() *RealClock {
	return &RealClock{}
}

// Ensure RealClock implements the Clock interface
var _ Clock = &RealClock{}

// Now returns the current system time.
// This implementation simply delegates to time.Now().
func (c *RealClock) Now() time.Time {
	return time.Now()
}

// NewTicker delegates to [time.NewTicker]. The returned ticker is a
// thin adapter around [time.Ticker] and shares its drop-on-slow-consumer
// semantics.
func (c *RealClock) NewTicker(d time.Duration) Ticker {
	return &realTicker{t: time.NewTicker(d)}
}

type realTicker struct {
	t *time.Ticker
}

func (r *realTicker) C() <-chan time.Time { return r.t.C }
func (r *realTicker) Stop()               { r.t.Stop() }
