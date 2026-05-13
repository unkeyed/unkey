package repeat

import (
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
)

// Every runs the given function in a go routine every d duration until the returned function is called.
func Every(d time.Duration, fn func()) func() {
	return EveryClock(clock.New(), d, fn)
}

// EveryClock is like [Every] but drives the schedule from the supplied
// [clock.Clock]. Tests can pass a [clock.TestClock] so that simulated
// time advances drive the firing schedule, instead of decoupling
// background work from request-time progress and producing flaky
// assertions.
func EveryClock(c clock.Clock, d time.Duration, fn func()) func() {
	t := c.NewTicker(d)
	done := make(chan struct{})

	fnWithRecovery := func() {
		defer func() {
			if r := recover(); r != nil {
				metrics.PanicsTotal.WithLabelValues("repeat.Every", "background").Inc()
			}
		}()
		fn()
	}

	go func() {
		fnWithRecovery()
		for {
			select {
			case <-t.C():
				fnWithRecovery()

			case <-done:
				return
			}
		}
	}()

	return sync.OnceFunc(func() {
		t.Stop()
		close(done)
	})
}
