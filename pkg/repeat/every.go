package repeat

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
)

// Every runs fn in a goroutine every d duration until the returned stop
// function is called. The first invocation happens immediately so callers
// don't have to wait one full period for their initialization tick.
//
// An optional jitter fraction in [0, 1] spreads tick timing uniformly in
// [d*(1-jitter), d*(1+jitter)]. Use it to avoid thundering herds when many
// instances start the same recurring task at the same time (e.g. a fleet
// rolling out simultaneously, all nodes hitting the same database every
// 10s in lockstep). Jitter applies to both the initial delay and every
// subsequent tick. Pass 0 (or omit entirely) for fixed-cadence behavior.
//
// Values outside [0, 1] are clamped: negative becomes 0, anything above 1
// becomes 1. Beyond 1.0 the lower bound on the jittered duration goes
// non-positive, which is meaningless for a periodic timer.
func Every(d time.Duration, fn func(), jitter ...float64) func() {
	jitterFraction := 0.0
	if len(jitter) > 0 {
		jitterFraction = jitter[0]
		if jitterFraction < 0 {
			jitterFraction = 0
		}
		if jitterFraction > 1 {
			jitterFraction = 1
		}
	}

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
		// Tick scheduling is anchored to absolute target times so a slow
		// fn does not drift the cadence. The next tick is computed from
		// the previous scheduled tick, never from "now after fn returned",
		// so a fn that takes T does not push the period to d+T. The timer
		// fires at the original target wall-clock, regardless of how long
		// fn ran.
		next := time.Now().Add(jitterDuration(d, jitterFraction))
		for {
			timer := time.NewTimer(time.Until(next))
			select {
			case <-timer.C:
				fnWithRecovery()
				next = next.Add(jitterDuration(d, jitterFraction))
				// Catch-up clamp: if fn took longer than the interval,
				// `next` is already in the past. Without this branch we
				// would create a timer with a negative duration that
				// fires immediately, then loop back, fire again, and so
				// on — effectively running fn back-to-back once for
				// every missed tick. Instead we drop missed ticks and
				// re-anchor the schedule to `now + d`, which mirrors the
				// way time.Ticker drops ticks via its single-slot
				// channel buffer when the receiver is slow.
				if now := time.Now(); now.After(next) {
					next = now.Add(jitterDuration(d, jitterFraction))
				}
			case <-done:
				timer.Stop()
				return
			}
		}
	}()

	return sync.OnceFunc(func() {
		close(done)
	})
}

// jitterDuration returns d perturbed by a uniform random offset in
// [-fraction*d, +fraction*d]. Returns d unchanged when fraction is
// non-positive. Clamps the result to a strictly positive duration so the
// caller's [time.NewTicker] or [time.Ticker.Reset] never panics on a
// pathological combination of d and fraction.
func jitterDuration(d time.Duration, fraction float64) time.Duration {
	if fraction <= 0 {
		return d
	}
	offset := (rand.Float64()*2 - 1) * fraction * float64(d)
	result := d + time.Duration(offset)
	if result <= 0 {
		return 1
	}
	return result
}
