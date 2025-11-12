package clock

import (
	"sync"
	"time"
)

// TestClock implements the Clock interface with a controlled time value.
// It allows tests to manually set and advance time to create deterministic
// test scenarios for time-dependent code.
type TestClock struct {
	mu  sync.RWMutex
	now time.Time
}

// NewTestClock creates a new TestClock instance.
// If a specific time is provided, the clock will be initialized to that time.
// Otherwise, it will be initialized to the current system time.
//
// Example:
//
//	// Create a test clock with the current time
//	clock := clock.NewTestClock()
//
//	// Create a test clock with a specific time
//	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
//	clock := clock.NewTestClock(fixedTime)
func NewTestClock(now ...time.Time) *TestClock {
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	return &TestClock{mu: sync.RWMutex{}, now: now[0]}
}

// Ensure TestClock implements the Clock interface
var _ Clock = (*TestClock)(nil)

// Now returns the current time as maintained by the TestClock.
// Unlike RealClock, this time is controlled programmatically rather
// than being tied to the system clock.
func (c *TestClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

// Tick advances the clock by the given duration and returns the new time.
// This method is particularly useful for testing time-dependent behavior
// without waiting for real time to pass.
//
// Example:
//
//	clock := clock.NewTestClock()
//
//	// Fast-forward 1 hour
//	newTime := clock.Tick(time.Hour)
//
//	// Fast-forward 30 days
//	newTime = clock.Tick(30 * 24 * time.Hour)
func (c *TestClock) Tick(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)

	return c.now
}

// Set changes the clock to the given time and returns the new time.
// This allows tests to jump to specific points in time for testing
// time-dependent behavior.
//
// Example:
//
//	clock := clock.NewTestClock()
//
//	// Set to a specific date and time
//	newYear := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
//	clock.Set(newYear)
func (c *TestClock) Set(t time.Time) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t

	return c.now
}
