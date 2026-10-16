package repeat

import (
	"context"
	"math"
	"math/rand/v2"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
)

const waitTimeout = 2 * time.Second

func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	require.Eventually(t, cond, waitTimeout, time.Millisecond)
}

func TestEvery_CallsImmediatelyThenOncePerInterval(t *testing.T) {
	const interval = 10 * time.Second
	clk := newTimerObserved()
	var calls atomic.Int32
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })

	Every(interval, func(context.Context) { calls.Add(1) }, WithClock(clk), UntilClose(done))

	for want := int32(1); want <= 4; want++ {
		d := clk.nextTimer(t)
		require.Equal(t, interval, d)
		eventually(t, func() bool { return calls.Load() == want })
		clk.Tick(d)
	}
}

// TestEvery_UntilCloseCancelsContextAndStopsLoop guarantees that closing the
// channel cancels the context handed to a running fn and that no later tick
// invokes fn again.
func TestEvery_UntilCloseCancelsContextAndStopsLoop(t *testing.T) {
	clk := clock.NewTestClock()
	done := make(chan struct{})
	started := make(chan struct{})
	cancelled := make(chan struct{})
	var calls atomic.Int32

	Every(time.Second, func(ctx context.Context) {
		if calls.Add(1) == 1 {
			close(started)
			<-ctx.Done()
			close(cancelled)
		}
	}, WithClock(clk), UntilClose(done))

	select {
	case <-started:
	case <-time.After(waitTimeout):
		t.Fatal("fn was not called")
	}
	close(done)
	select {
	case <-cancelled:
	case <-time.After(waitTimeout):
		t.Fatal("fn context was not cancelled when the channel closed")
	}

	clk.Tick(10 * time.Second)
	require.Never(t, func() bool { return calls.Load() > 1 }, 50*time.Millisecond, time.Millisecond)
}

func TestEvery_WithContextStopsLoopAndDerivesFnContext(t *testing.T) {
	type key struct{}
	clk := clock.NewTestClock()
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), key{}, "v"))
	var calls atomic.Int32
	var seen atomic.Value

	Every(time.Second, func(ctx context.Context) {
		seen.Store(ctx.Value(key{}))
		calls.Add(1)
	}, WithClock(clk), WithContext(parent))

	eventually(t, func() bool { return calls.Load() == 1 })
	require.Equal(t, "v", seen.Load())

	cancel()
	clk.Tick(10 * time.Second)
	require.Never(t, func() bool { return calls.Load() > 1 }, 50*time.Millisecond, time.Millisecond)
}

func TestEvery_AlreadyStoppedNeverCallsFn(t *testing.T) {
	done := make(chan struct{})
	close(done)
	var calls atomic.Int32

	Every(time.Millisecond, func(context.Context) { calls.Add(1) }, UntilClose(done))

	require.Never(t, func() bool { return calls.Load() > 0 }, 50*time.Millisecond, time.Millisecond)
}

func TestEvery_RecoversPanicAndKeepsTicking(t *testing.T) {
	clk := newTimerObserved()
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	counter := metrics.PanicsTotal.WithLabelValues("repeat.Every", "background")
	before := counterValue(t, counter)
	var calls atomic.Int32

	Every(time.Second, func(context.Context) {
		if calls.Add(1) == 1 {
			panic("boom")
		}
	}, WithClock(clk), UntilClose(done))

	clk.Tick(clk.nextTimer(t))
	eventually(t, func() bool { return calls.Load() == 2 })
	clk.Tick(clk.nextTimer(t))
	eventually(t, func() bool { return calls.Load() == 3 })
	require.InDelta(t, before+1, counterValue(t, counter), 0)
}

func TestEvery_AcceptsSmallestPositiveInterval(t *testing.T) {
	done := make(chan struct{})
	require.NotPanics(t, func() {
		Every(time.Nanosecond, func(context.Context) {}, WithClock(clock.NewTestClock()), UntilClose(done))
	})
	close(done)
}

// TestEvery_WithoutUntilCloseStartsOneGoroutinePerLoop guarantees that a loop
// without UntilClose does not park a second goroutine waiting on a nil channel.
func TestEvery_WithoutUntilCloseStartsOneGoroutinePerLoop(t *testing.T) {
	const loops = 20
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	before := runtime.NumGoroutine()
	var started atomic.Int32

	for range loops {
		Every(time.Hour, func(context.Context) { started.Add(1) },
			WithClock(clock.NewTestClock()), WithContext(ctx))
	}
	eventually(t, func() bool { return started.Load() == loops })

	require.Less(t, runtime.NumGoroutine()-before, loops+loops/2)
}

func TestEvery_PanicsOnNonPositiveInterval(t *testing.T) {
	require.Panics(t, func() { Every(0, func(context.Context) {}) })
	require.Panics(t, func() { Every(-time.Second, func(context.Context) {}) })
}

// TestEvery_NoCallAfterStopWhenTimerIsReady guarantees that a timer which is
// already due when the loop stops does not trigger another call. Repeating
// the scenario makes a random select choice between the ready timer and the
// cancelled context fail with overwhelming probability.
func TestEvery_NoCallAfterStopWhenTimerIsReady(t *testing.T) {
	for range 50 {
		done := make(chan struct{})
		entered := make(chan struct{})
		release := make(chan struct{})
		var calls atomic.Int32

		Every(time.Hour, func(context.Context) {
			if calls.Add(1) == 1 {
				close(entered)
				<-release
			}
		}, WithClock(clock.NewTestClock()), UntilClose(done), WithJitter(1, func() float64 { return 0 }))

		select {
		case <-entered:
		case <-time.After(waitTimeout):
			t.Fatal("fn was not called")
		}
		close(done)
		close(release)

		require.Never(t, func() bool { return calls.Load() > 1 }, 5*time.Millisecond, time.Millisecond)
	}
}

func TestEvery_UsesSystemClockByDefault(t *testing.T) {
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	var calls atomic.Int32

	Every(5*time.Millisecond, func(context.Context) { calls.Add(1) }, UntilClose(done))

	eventually(t, func() bool { return calls.Load() >= 3 })
}

func TestJitterDuration_SpreadsEvenlyAroundInterval(t *testing.T) {
	const (
		d        = 100 * time.Millisecond
		fraction = 0.5
		samples  = 10_000
	)
	lo, hi := time.Duration(math.MaxInt64), time.Duration(0)
	for range samples {
		got := jitterDuration(d, fraction, rand.Float64)
		lo, hi = min(lo, got), max(hi, got)
	}
	require.GreaterOrEqual(t, lo, d/2)
	require.LessOrEqual(t, hi, d*3/2)
	require.Less(t, lo, d*55/100, "delays must reach the lower half of the range")
	require.Greater(t, hi, d*145/100, "delays must reach the upper half of the range")
}

func TestEvery_JitterUsesGivenRandomSource(t *testing.T) {
	const interval = 100 * time.Millisecond
	clk := newTimerObserved()
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })

	Every(interval, func(context.Context) {}, WithClock(clk), UntilClose(done),
		WithJitter(0.5, func() float64 { return 0.75 }))

	require.Equal(t, interval*5/4, clk.nextTimer(t))
}

func TestEvery_Jitter(t *testing.T) {
	const interval = 100 * time.Millisecond

	start := func(t *testing.T, fn func(ctx context.Context, clk *timerObserved), opts ...Option) *timerObserved {
		t.Helper()
		clk := newTimerObserved()
		done := make(chan struct{})
		t.Cleanup(func() { close(done) })
		Every(interval, func(ctx context.Context) { fn(ctx, clk) },
			append([]Option{WithClock(clk), UntilClose(done)}, opts...)...)
		return clk
	}

	t.Run("delays stay within the configured range", func(t *testing.T) {
		clk := start(t, func(context.Context, *timerObserved) {}, WithJitter(0.5))
		seen := map[time.Duration]bool{}
		for range 50 {
			d := clk.nextTimer(t)
			require.GreaterOrEqual(t, d, interval/2)
			require.LessOrEqual(t, d, interval*3/2)
			seen[d] = true
			clk.Tick(d)
		}
		require.Greater(t, len(seen), 1, "jittered delays must vary")
	})

	t.Run("negative jitter is fixed cadence", func(t *testing.T) {
		var calls atomic.Int32
		clk := start(t, func(context.Context, *timerObserved) { calls.Add(1) }, WithJitter(-0.5))
		for range 3 {
			d := clk.nextTimer(t)
			require.Equal(t, interval, d)
			clk.Tick(d)
		}
		eventually(t, func() bool { return calls.Load() == 4 })
	})

	t.Run("jitter above one clamps to one", func(t *testing.T) {
		clk := start(t, func(context.Context, *timerObserved) {}, WithJitter(5))
		for range 50 {
			d := clk.nextTimer(t)
			require.Positive(t, d)
			require.LessOrEqual(t, d, 2*interval)
			clk.Tick(d)
		}
	})

	t.Run("slow fn does not drift the schedule", func(t *testing.T) {
		const work = 50 * time.Millisecond
		var calls atomic.Int32
		clk := start(t, func(_ context.Context, clk *timerObserved) {
			if calls.Add(1) == 2 {
				clk.Tick(work)
			}
		}, WithJitter(0.1))

		first := clk.nextTimer(t)
		clk.Tick(first)
		second := clk.nextTimer(t)
		require.LessOrEqual(t, second, interval*11/10-work,
			"the next delay must be measured from the previous target, not from when fn returned")
	})

	t.Run("catch-up drops missed ticks", func(t *testing.T) {
		const work = 10 * interval
		var calls atomic.Int32
		clk := start(t, func(_ context.Context, clk *timerObserved) {
			if calls.Add(1) == 2 {
				clk.Tick(work)
			}
		}, WithJitter(0.1))

		first := clk.nextTimer(t)
		clk.Tick(first)
		second := clk.nextTimer(t)
		require.GreaterOrEqual(t, second, interval*9/10)
		require.LessOrEqual(t, second, interval*11/10)
	})
}

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, c.Write(&m))
	return m.GetCounter().GetValue()
}

type timerObserved struct {
	*clock.TestClock
	durations chan time.Duration
}

func newTimerObserved() *timerObserved {
	return &timerObserved{
		TestClock: clock.NewTestClock(),
		durations: make(chan time.Duration, 1024),
	}
}

func (c *timerObserved) nextTimer(t *testing.T) time.Duration {
	t.Helper()
	select {
	case d := <-c.durations:
		return d
	case <-time.After(waitTimeout):
		t.Fatal("no timer was armed")
		return 0
	}
}

func (c *timerObserved) NewTimer(d time.Duration) clock.Timer {
	timer := c.TestClock.NewTimer(d)
	c.durations <- d
	return timer
}
