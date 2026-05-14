package clock

import (
	"slices"
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
	return &TestClock{mu: sync.RWMutex{}, now: now[0]} //nolint:exhaustruct
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
// Forward ticks fire any pending Ticker registrations whose period
// elapsed during the advance, in order. Each fire blocks until the
// consumer receives the tick from the channel, so by the time Tick
// returns every due tick has been delivered. The consumer's
// post-receive work runs asynchronously, so tests asserting on its
// effects should use a polling helper or a done channel rather than
// treating Tick's return as a barrier.
//
// Negative or zero durations only update the clock value and do not
// fire tickers.
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
	c.now = c.now.Add(d)
	target := c.now
	c.mu.Unlock()

	if d > 0 {
		c.fireTickers(target)
	}
	return target
}

// Set changes the clock to the given time and returns the new time.
// Forward jumps fire pending Tickers in the same way as [TestClock.Tick];
// backward jumps only adjust the clock value.
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
	prev := c.now
	c.now = t
	c.mu.Unlock()

	if t.After(prev) {
		c.fireTickers(t)
	}
	return t
}

// NewTicker registers a Ticker driven by simulated time. Each Tick or
// forward Set call delivers every tick whose scheduled fire time fell
// within the advance, in order, blocking on the consumer for each
// delivery. A zero or negative period panics, matching the contract of
// [time.NewTicker].
func (c *TestClock) NewTicker(d time.Duration) Ticker {
	if d <= 0 {
		panic("clock.TestClock: non-positive interval for NewTicker")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &testTicker{ //nolint:exhaustruct // stopMu and stopped take their zero values
		parent:   c,
		period:   d,
		nextFire: c.now.Add(d),
		ch:       make(chan time.Time),
		stopCh:   make(chan struct{}),
	}
	c.tickers = append(c.tickers, t)
	return t
}

func (c *TestClock) fireTickers(target time.Time) {
	c.mu.RLock()
	snapshot := slices.Clone(c.tickers)
	c.mu.RUnlock()

	for _, t := range snapshot {
		t.fire(target)
	}
}

func (c *TestClock) removeTicker(t *testTicker) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, x := range c.tickers {
		if x == t {
			c.tickers = slices.Delete(c.tickers, i, i+1)
			return
		}
	}
}

// testTicker fires synchronously: each due tick blocks until the
// consumer receives it. That keeps simulated time aligned with what
// the consumer has observed, so tests can advance the clock and rely
// on the receive having happened before Tick returns.
//
// stopCh closes when Stop is called. fire selects on (channel send |
// stopCh) so a Stop concurrent with a pending send unblocks fire
// instead of leaving Tick wedged on a consumer that has already
// detached.
type testTicker struct {
	parent   *TestClock
	period   time.Duration
	nextFire time.Time
	ch       chan time.Time
	stopCh   chan struct{}

	stopMu  sync.Mutex
	stopped bool
}

func (t *testTicker) C() <-chan time.Time { return t.ch }

func (t *testTicker) Stop() {
	t.stopMu.Lock()
	if !t.stopped {
		t.stopped = true
		close(t.stopCh)
	}
	t.stopMu.Unlock()
	t.parent.removeTicker(t)
}

func (t *testTicker) fire(target time.Time) {
	for {
		t.stopMu.Lock()
		if t.stopped || t.nextFire.After(target) {
			t.stopMu.Unlock()
			return
		}
		when := t.nextFire
		t.nextFire = t.nextFire.Add(t.period)
		t.stopMu.Unlock()

		select {
		case t.ch <- when:
		case <-t.stopCh:
			return
		}
	}
}
