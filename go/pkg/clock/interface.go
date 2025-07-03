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

	// NewTicker returns a new Ticker containing a channel that will send
	// the current time on the channel after each tick.
	// In production implementations, this creates a real ticker.
	// In test implementations, this creates a controllable ticker.
	NewTicker(d time.Duration) Ticker
}

// Ticker represents a ticker that sends the current time at regular intervals.
// This interface abstracts the standard library's time.Ticker to enable
// deterministic testing of ticker-based code.
type Ticker interface {
	// C returns the channel on which the ticks are delivered.
	C() <-chan time.Time

	// Stop turns off a ticker. After Stop, no more ticks will be sent.
	Stop()
}
