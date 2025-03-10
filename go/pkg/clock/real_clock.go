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
