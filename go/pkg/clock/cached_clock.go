package clock

import (
	"sync/atomic"
	"time"
)

// This clock implementation takes inspiration from
// https://github.com/agilira/go-timecache
// but works with our clock.Clock interface

// CachedClock implements the Clock interface using a cached atomic value
// to avoid the overhead of system calls from time.Now().
type CachedClock struct {
	nanos      atomic.Int64
	resolution time.Duration
	ticker     *time.Ticker
	done       chan struct{}
}

// NewCachedClock creates a new CachedClock that uses the system time cached every [resolution].
//
// Example:
//
//	clock := clock.NewCachedClock(time.Millisecond)
//	currentTime := clock.Now()
func NewCachedClock(resolution time.Duration) *CachedClock {

	done := make(chan struct{})
	ticker := time.NewTicker(resolution)

	c := &CachedClock{
		nanos:      atomic.Int64{},
		resolution: resolution,
		ticker:     ticker,
		done:       done,
	}

	// Initialize with current time
	c.nanos.Store(time.Now().UnixNano())

	go func() {
		for {
			select {
			case <-ticker.C:
				c.nanos.Store(time.Now().UnixNano())
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return c

}

// Ensure CachedClock implements the Clock interface
// nolint:exhaustruct
var _ Clock = &CachedClock{}

// Now returns the current system time.
// This implementation returns the cached time value.
func (c *CachedClock) Now() time.Time {
	return time.Unix(0, c.nanos.Load())
}

// Close stops the background goroutine that updates the cached time.
// After calling Close, the clock will continue to return the last cached time
// but will no longer update. This method should be called to clean up resources
// when the CachedClock is no longer needed.
func (c *CachedClock) Close() {
	close(c.done)
}
