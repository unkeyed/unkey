---
title: clock
description: "provides a flexible interface for time-related operations,"
---

Package clock provides a flexible interface for time-related operations, allowing for consistent time handling in both production and test environments.

The key benefit of this package is that it enables deterministic testing of time-dependent code by providing mock implementations that can be controlled in tests, while using the system clock in production.

The package offers: - A standard interface for getting the current time - A production implementation that uses the system clock - A test implementation that allows precise control over time

Example:

	// In production code:
	func ProcessWithExpiry(clock clock.Clock, data []Item) []Item {
	    now := clock.Now()
	    result := make([]Item, 0)
	    for _, item := range data {
	        if item.ExpiresAt.After(now) {
	            result = append(result, item)
	        }
	    }
	    return result
	}

	// In production:
	processor := &Processor{clock: clock.New()}

	// In tests:
	func TestProcessWithExpiry(t *testing.T) {
	    clock := clock.NewTestClock()

	    // Set a fixed time for deterministic testing
	    fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	    clock.Set(fixedTime)

	    // Create test data that expires at different times
	    data := []Item{
	        {ID: "1", ExpiresAt: fixedTime.Add(-time.Hour)},  // expired
	        {ID: "2", ExpiresAt: fixedTime.Add(time.Hour)},   // not expired
	    }

	    result := ProcessWithExpiry(clock, data)

	    // Should only contain the non-expired item
	    assert.Len(t, result, 1)
	    assert.Equal(t, "2", result[0].ID)

	    // Advance time past the expiration of the second item
	    clock.Tick(2 * time.Hour)

	    // Now all items should be expired
	    result = ProcessWithExpiry(clock, data)
	    assert.Empty(t, result)
	}

## Types

### type CachedClock

```go
type CachedClock struct {
	nanos      atomic.Int64
	resolution time.Duration
	ticker     *time.Ticker
	done       chan struct{}
}
```

CachedClock implements the Clock interface using a cached atomic value to avoid the overhead of system calls from time.Now().

#### func NewCachedClock

```go
func NewCachedClock(resolution time.Duration) *CachedClock
```

NewCachedClock creates a new CachedClock that uses the system time cached every \[resolution].

Example:

	clock := clock.NewCachedClock(time.Millisecond)
	currentTime := clock.Now()

#### func (CachedClock) Close

```go
func (c *CachedClock) Close()
```

Close stops the background goroutine that updates the cached time. After calling Close, the clock will continue to return the last cached time but will no longer update. This method should be called to clean up resources when the CachedClock is no longer needed.

#### func (CachedClock) Now

```go
func (c *CachedClock) Now() time.Time
```

Now returns the current system time. This implementation returns the cached time value.

### type Clock

```go
type Clock interface {
	// Now returns the current time.
	// In production implementations, this returns the system time.
	// In test implementations, this returns a controlled time that
	// can be manipulated for testing purposes.
	Now() time.Time
}
```

Clock is an interface for getting the current time and creating tickers. By abstracting time operations behind this interface, code can be written that works with both the system clock in production and controlled time in tests, enabling deterministic testing of time-dependent logic.

This approach helps avoid flaky tests caused by timing dependencies and allows for simulating time-based scenarios without waiting for real time to pass.

### type RealClock

```go
type RealClock struct {
}
```

RealClock implements the Clock interface using the system clock. This is the implementation that should be used in production code.

#### func New

```go
func New() *RealClock
```

New creates a new RealClock that uses the system time. This is the standard clock implementation for production use.

Example:

	clock := clock.New()
	currentTime := clock.Now()

#### func (RealClock) Now

```go
func (c *RealClock) Now() time.Time
```

Now returns the current system time. This implementation simply delegates to time.Now().

### type TestClock

```go
type TestClock struct {
	mu  sync.RWMutex
	now time.Time
}
```

TestClock implements the Clock interface with a controlled time value. It allows tests to manually set and advance time to create deterministic test scenarios for time-dependent code.

#### func NewTestClock

```go
func NewTestClock(now ...time.Time) *TestClock
```

NewTestClock creates a new TestClock instance. If a specific time is provided, the clock will be initialized to that time. Otherwise, it will be initialized to the current system time.

Example:

	// Create a test clock with the current time
	clock := clock.NewTestClock()

	// Create a test clock with a specific time
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := clock.NewTestClock(fixedTime)

#### func (TestClock) Now

```go
func (c *TestClock) Now() time.Time
```

Now returns the current time as maintained by the TestClock. Unlike RealClock, this time is controlled programmatically rather than being tied to the system clock.

#### func (TestClock) Set

```go
func (c *TestClock) Set(t time.Time) time.Time
```

Set changes the clock to the given time and returns the new time. This allows tests to jump to specific points in time for testing time-dependent behavior.

Example:

	clock := clock.NewTestClock()

	// Set to a specific date and time
	newYear := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.Set(newYear)

#### func (TestClock) Tick

```go
func (c *TestClock) Tick(d time.Duration) time.Time
```

Tick advances the clock by the given duration and returns the new time. This method is particularly useful for testing time-dependent behavior without waiting for real time to pass.

Example:

	clock := clock.NewTestClock()

	// Fast-forward 1 hour
	newTime := clock.Tick(time.Hour)

	// Fast-forward 30 days
	newTime = clock.Tick(30 * 24 * time.Hour)

