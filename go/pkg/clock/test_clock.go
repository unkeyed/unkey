package clock

import "time"

type TestClock struct {
	now time.Time
}

func NewTestClock(now ...time.Time) *TestClock {
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	return &TestClock{now: now[0]}
}

// nolint:exhaustruct
var _ Clock = &TestClock{}

func (c *TestClock) Now() time.Time {
	return c.now
}

// Tick advances the clock by the given duration and returns the new time.
func (c *TestClock) Tick(d time.Duration) time.Time {
	c.now = c.now.Add(d)
	return c.now
}

// Set sets the clock to the given time and returns the new time.
func (c *TestClock) Set(t time.Time) time.Time {
	c.now = t
	return c.now
}
