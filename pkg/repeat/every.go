package repeat

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
)

type config struct {
	clock  clock.Clock
	jitter float64
	random func() float64
	ctx    context.Context
	done   <-chan struct{}
}

// Option configures [Every].
type Option func(*config)

// WithClock drives the schedule from c instead of the system clock. Tests
// pass a [clock.TestClock] so simulated time advances drive the firing
// schedule instead of real wall time.
func WithClock(c clock.Clock) Option {
	return func(cfg *config) { cfg.clock = c }
}

// WithJitter spreads tick timing uniformly in [d*(1-fraction), d*(1+fraction)].
// Use it to avoid thundering herds when many instances start the same
// recurring task at the same time (e.g. a fleet rolling out simultaneously,
// all nodes hitting the same database every 10s in lockstep). Jitter applies
// to every tick after the immediate first call.
//
// Values outside [0, 1] are clamped: negative becomes 0, anything above 1
// becomes 1. Beyond 1.0 the lower bound on the jittered duration goes
// non-positive, which is meaningless for a periodic timer. Zero means fixed
// cadence.
//
// An optional random source replaces [rand.Float64] as the jitter draw. It
// must return values in [0, 1). Tests use it to make jitter deterministic.
func WithJitter(fraction float64, random ...func() float64) Option {
	return func(cfg *config) {
		cfg.jitter = min(max(fraction, 0), 1)
		if len(random) > 0 {
			cfg.random = random[0]
		}
	}
}

// UntilClose stops the loop when done is closed. The context passed to fn
// is cancelled at the same moment.
func UntilClose(done <-chan struct{}) Option {
	return func(cfg *config) { cfg.done = done }
}

// WithContext stops the loop when ctx is done. The context passed to fn is
// derived from ctx, so it carries ctx's values and cancellation.
func WithContext(ctx context.Context) Option {
	return func(cfg *config) { cfg.ctx = ctx }
}

// Every runs fn in a background goroutine every interval. fn runs
// sequentially, never concurrently with itself. The first invocation
// happens immediately so callers don't have to wait one full period for
// their initialization tick. If the loop is already stopped, fn does not
// run at all.
//
// The context passed to fn is cancelled when the loop stops. Every does
// not wait for a running fn to return after the loop stops.
//
// Every returns immediately and gives no stop function. Without [UntilClose]
// or [WithContext], the loop runs until the process exits.
//
// Every panics if interval is not positive.
//
// A panic in fn is recovered, counted in [metrics.PanicsTotal], and the loop
// continues.
//
// The schedule is anchored to absolute target times, so a slow fn does not
// drift the cadence. If fn runs longer than the interval, missed
// ticks are dropped and the schedule re-anchors to now plus one interval,
// mirroring how [time.Ticker] drops ticks for a slow receiver.
func Every(interval time.Duration, fn func(ctx context.Context), opts ...Option) {
	if interval <= 0 {
		panic("repeat.Every: non-positive interval")
	}

	cfg := config{
		clock:  clock.New(),
		jitter: 0, //placebo:equivalent zero value of the field
		random: rand.Float64,
		ctx:    context.Background(),
		done:   nil, //placebo:equivalent zero value of the field
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	ctx, cancel := context.WithCancel(cfg.ctx)
	if cfg.done != nil {
		select {
		case <-cfg.done:
			cancel()
		default:
		}
		go func() {
			select {
			case <-cfg.done:
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	go func() {
		defer cancel()
		run(ctx, cfg, interval, fn)
	}()
}

func run(ctx context.Context, cfg config, interval time.Duration, fn func(ctx context.Context)) {
	call := func() {
		defer func() {
			if r := recover(); r != nil {
				metrics.PanicsTotal.WithLabelValues("repeat.Every", "background").Inc()
			}
		}()
		fn(ctx)
	}

	stopped := func() bool {
		select {
		case <-ctx.Done():
			return true
		case <-cfg.done:
			return true
		default:
			return false
		}
	}

	if stopped() {
		return
	}
	call()

	next := cfg.clock.Now().Add(jitterDuration(interval, cfg.jitter, cfg.random))
	for {
		timer := cfg.clock.NewTimer(next.Sub(cfg.clock.Now()))
		select {
		case <-timer.C():
			if stopped() {
				return
			}
			call()
			next = next.Add(jitterDuration(interval, cfg.jitter, cfg.random))
			if now := cfg.clock.Now(); now.After(next) {
				next = now.Add(jitterDuration(interval, cfg.jitter, cfg.random))
			}
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

// jitterDuration returns d perturbed by a uniform random offset in
// [-fraction*d, +fraction*d], using random as the draw in [0, 1).
func jitterDuration(d time.Duration, fraction float64, random func() float64) time.Duration {
	offset := (random()*2 - 1) * fraction * float64(d)
	return d + time.Duration(offset)
}
