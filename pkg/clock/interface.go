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
}
