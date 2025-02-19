package clock

import (
	"sync"
	"time"
)

type TestClock struct {
	mu  sync.RWMutex
	now time.Time
}

func NewTestClock(now ...time.Time) *TestClock {
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	return &TestClock{mu: sync.RWMutex{}, now: now[0]}
}

// nolint:exhaustruct
var _ Clock = &TestClock{}

func (c *TestClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

// Tick advances the clock by the given duration and returns the new time.
func (c *TestClock) Tick(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	return c.now
}

// Set sets the clock to the given time and returns the new time.
func (c *TestClock) Set(t time.Time) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
	return c.now
}
