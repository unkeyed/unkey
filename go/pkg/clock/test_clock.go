package clock

import (
	"sync"
	"time"
)

// TestClock implements the Clock interface with a controlled time value.
// It allows tests to manually set and advance time to create deterministic
// test scenarios for time-dependent code.
type TestClock struct {
	mu      sync.RWMutex
	now     time.Time
	tickers []*testTicker
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
// without waiting for real time to pass. It also triggers any tickers
// that should fire during this time advancement.
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

	// Notify all tickers about the time advancement
	for _, ticker := range c.tickers {
		ticker.checkForTick(c.now)
	}

	return c.now
}

// Set changes the clock to the given time and returns the new time.
// This allows tests to jump to specific points in time for testing
// time-dependent behavior. It also triggers any tickers that should
// fire during this time change.
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

	// Notify all tickers about the time change
	for _, ticker := range c.tickers {
		ticker.checkForTick(c.now)
	}

	return c.now
}

// NewTicker returns a new test ticker that can be manually controlled.
// The ticker will only send ticks when the clock is advanced using Tick() or Set().
// This enables deterministic testing of ticker-based code.
func (c *TestClock) NewTicker(d time.Duration) Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan time.Time, 1) // Buffered to prevent blocking
	ticker := &testTicker{
		clock:    c,
		interval: d,
		lastTick: c.now,
		ch:       ch,
		stopped:  false,
	}

	// Register this ticker with the clock
	c.tickers = append(c.tickers, ticker)

	return ticker
}

// testTicker implements a controllable ticker for testing
type testTicker struct {
	clock    *TestClock
	interval time.Duration
	lastTick time.Time
	ch       chan time.Time
	stopped  bool
	mu       sync.Mutex
}

// C returns the channel on which the ticks are delivered.
func (t *testTicker) C() <-chan time.Time {
	return t.ch
}

// Stop turns off the ticker.
func (t *testTicker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
	close(t.ch)

	// Remove this ticker from the clock's list
	t.clock.removeTicker(t)
}

// removeTicker removes a ticker from the clock's list (internal method)
func (c *TestClock) removeTicker(tickerToRemove *testTicker) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find and remove the ticker
	for i, ticker := range c.tickers {
		if ticker == tickerToRemove {
			// Remove by swapping with last element and truncating
			c.tickers[i] = c.tickers[len(c.tickers)-1]
			c.tickers = c.tickers[:len(c.tickers)-1]
			break
		}
	}
}

// checkForTick checks if enough time has passed to send a tick.
// This is called internally when the clock advances.
func (t *testTicker) checkForTick(currentTime time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return
	}

	// Check if enough time has passed since last tick
	if currentTime.Sub(t.lastTick) >= t.interval {
		// Send tick if channel has space (non-blocking)
		select {
		case t.ch <- currentTime:
			t.lastTick = currentTime
		default:
			// Channel is full, skip this tick (mimics real ticker behavior)
		}
	}
}
