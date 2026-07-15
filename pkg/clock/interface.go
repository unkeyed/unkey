package clock

import "time"

// Clock is an interface for getting the current time and creating tickers.
// By abstracting time operations behind this interface, code can be written
// that works with both the system clock in production and controlled time
// in tests, enabling deterministic testing of time-dependent logic.
//
// This approach helps avoid flaky tests caused by timing dependencies and
// allows for simulating time-based scenarios without waiting for real time to pass.
type Clock interface {
	// Now returns the current time.
	// In production implementations, this returns the system time.
	// In test implementations, this returns a controlled time that
	// can be manipulated for testing purposes.
	Now() time.Time

	// NewTicker returns a Ticker that fires every d. Production
	// implementations delegate to time.NewTicker. Test implementations
	// fire when simulated time advances past the ticker period, so a
	// background goroutine driven by NewTicker observes the same notion
	// of time as the rest of the test instead of running on real time.
	NewTicker(d time.Duration) Ticker
}

// Ticker delivers ticks on a channel at a regular cadence. It mirrors
// the surface of [time.Ticker] needed by callers that work against the
// Clock interface, so production code can stay agnostic of whether the
// clock is real or simulated.
type Ticker interface {
	// C returns the channel on which the ticks are delivered. Real
	// tickers buffer one tick and drop further ticks while the channel
	// is full; test tickers send synchronously, so a slow consumer
	// holds back simulated time advance until it catches up.
	C() <-chan time.Time

	// Stop turns off the ticker. After Stop, no further ticks are
	// delivered, but the channel is not closed, matching time.Ticker.
	Stop()
}
