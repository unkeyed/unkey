package gcra_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/gcra"
)

// hundredPerTenSeconds prices a unit at 100ms and has a burst tolerance of
// 10 units, which is one second.
var hundredPerTenSeconds = gcra.Rate{Limit: 100, Duration: 10 * time.Second}

// writer simulates one writer: it commits every decision's state, as the
// Limiter does, and sees its peers only through what the test passes in.
type writer struct {
	t     *testing.T
	state gcra.State
}

func (w *writer) allow(now time.Time, rate gcra.Rate, cost int64, peers ...gcra.State) gcra.Result {
	w.t.Helper()
	decision, err := gcra.Decide(now, rate, cost, w.state, combine(peers...))
	require.NoError(w.t, err)
	if decision.Allowed {
		w.state = decision.Writer
	} else {
		require.Equal(w.t, w.state, decision.Writer)
	}
	return decision.Result
}

// allowed returns how many of n requests of cost 1 pass at now.
func (w *writer) allowed(now time.Time, rate gcra.Rate, n int, peers ...gcra.State) int {
	w.t.Helper()
	passed := 0
	for range n {
		if w.allow(now, rate, 1, peers...).Allowed {
			passed++
		}
	}
	return passed
}

func combine(states ...gcra.State) gcra.State {
	var combined gcra.State
	for _, s := range states {
		combined.SpentNs += s.SpentNs
		combined.FreshStartNs = max(combined.FreshStartNs, s.FreshStartNs)
	}
	return combined
}

func TestDecide_BurstThenOnePerUnit(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	require.Equal(t, 10, w.allowed(clk.Now(), hundredPerTenSeconds, 11), "the burst tolerance is a tenth of the limit")

	denied := w.allow(clk.Now(), hundredPerTenSeconds, 1)
	require.False(t, denied.Allowed)
	require.Equal(t, int64(0), denied.Remaining)
	require.Equal(t, int64(11), denied.Current)
	require.Equal(t, clk.Now().Add(time.Second), denied.ResetAt)

	for range 5 {
		clk.Tick(100 * time.Millisecond)
		require.Equal(t, 1, w.allowed(clk.Now(), hundredPerTenSeconds, 3))
	}

	clk.Tick(50 * time.Millisecond)
	require.Equal(t, 0, w.allowed(clk.Now(), hundredPerTenSeconds, 1))
}

func TestDecide_ReportsRemainingAndReset(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	first := w.allow(clk.Now(), hundredPerTenSeconds, 3)
	require.True(t, first.Allowed)
	require.Equal(t, int64(7), first.Remaining)
	require.Equal(t, int64(3), first.Current)
	require.Equal(t, clk.Now().Add(300*time.Millisecond), first.ResetAt)

	clk.Tick(150 * time.Millisecond)
	check := w.allow(clk.Now(), hundredPerTenSeconds, 0)
	require.True(t, check.Allowed)
	require.Equal(t, int64(8), check.Remaining)
	require.Equal(t, int64(2), check.Current)
}

func TestDecide_ZeroCostChangesNothing(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	before := gcra.State{SpentNs: 0, FreshStartNs: 0}

	decision, err := gcra.Decide(clk.Now(), hundredPerTenSeconds, 0, before, before)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.False(t, decision.Clamped)
	require.Equal(t, before, decision.Writer)
	require.Equal(t, int64(10), decision.Remaining)
}

func TestDecide_CostAboveLimitIsDenied(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	denied := w.allow(clk.Now(), hundredPerTenSeconds, 101)
	require.False(t, denied.Allowed)
	require.Equal(t, int64(10), denied.Remaining)
	require.Equal(t, int64(101), denied.Current)
	require.Equal(t, gcra.State{SpentNs: 0, FreshStartNs: 0}, w.state)

	denied = w.allow(clk.Now(), hundredPerTenSeconds, math.MaxInt64)
	require.False(t, denied.Allowed)
	require.Equal(t, int64(math.MaxInt64), denied.Current)
}

func TestDecide_CostAboveBurstFitsOnlyWhenNothingIsOutstanding(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	require.True(t, w.allow(clk.Now(), hundredPerTenSeconds, 1).Allowed)
	denied := w.allow(clk.Now(), hundredPerTenSeconds, 50)
	require.False(t, denied.Allowed, "one unit is outstanding")
	require.Equal(t, int64(9), denied.Remaining)

	clk.Tick(100 * time.Millisecond)
	half := w.allow(clk.Now(), hundredPerTenSeconds, 50)
	require.True(t, half.Allowed)
	require.Equal(t, int64(0), half.Remaining)
	require.Equal(t, clk.Now().Add(5*time.Second), half.ResetAt)
}

func TestDecide_FullLimitRequestIsPaidBackBeforeTheNext(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	full := w.allow(clk.Now(), hundredPerTenSeconds, 100)
	require.True(t, full.Allowed, "a request of the whole limit fits an empty bucket")
	require.Equal(t, int64(0), full.Remaining)
	require.Equal(t, int64(100), full.Current)
	require.Equal(t, clk.Now().Add(10*time.Second), full.ResetAt)

	clk.Tick(9 * time.Second)
	require.Equal(t, 0, w.allowed(clk.Now(), hundredPerTenSeconds, 1), "the debt beyond the burst tolerance is not paid yet")

	clk.Tick(100 * time.Millisecond)
	require.Equal(t, 1, w.allowed(clk.Now(), hundredPerTenSeconds, 2))
}

func TestDecide_IdleClampForgetsIdleTime(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	require.Equal(t, 10, w.allowed(clk.Now(), hundredPerTenSeconds, 10))

	clk.Tick(100 * time.Second)
	decision, err := gcra.Decide(clk.Now(), hundredPerTenSeconds, 1, w.state, gcra.State{SpentNs: 0, FreshStartNs: 0})
	require.NoError(t, err)
	require.True(t, decision.Clamped)
	require.Equal(t, clk.Now().UnixNano()-time.Second.Nanoseconds(), decision.Writer.FreshStartNs)

	require.Equal(t, 10, w.allowed(clk.Now(), hundredPerTenSeconds, 20), "idle time beyond one burst must not be banked")
}

func TestDecide_ConcurrentClampsCombineByMax(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	a, b := &writer{t: t}, &writer{t: t}

	require.Equal(t, 5, a.allowed(clk.Now(), hundredPerTenSeconds, 5))
	require.Equal(t, 5, b.allowed(clk.Now(), hundredPerTenSeconds, 5, a.state))

	clk.Tick(100 * time.Second)
	aSynced, bSynced := a.state, b.state
	require.Equal(t, 5, a.allowed(clk.Now(), hundredPerTenSeconds, 5, bSynced))
	require.Equal(t, 5, b.allowed(clk.Now(), hundredPerTenSeconds, 5, aSynced))
	require.Equal(t, a.state.FreshStartNs, b.state.FreshStartNs)

	after := a.allow(clk.Now(), hundredPerTenSeconds, 0, b.state)
	require.Equal(t, clk.Now().Add(time.Second), after.ResetAt, "both clamps describe the same baseline, so it counts once")
	require.Equal(t, int64(0), after.Remaining)

	clk.Tick(100 * time.Millisecond)
	require.Equal(t, 1, a.allowed(clk.Now(), hundredPerTenSeconds, 2, b.state))
}

func TestDecide_StaleWritersPayBackOverAdmission(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	start := clk.Now()
	a, b := &writer{t: t}, &writer{t: t}

	admitted := a.allowed(clk.Now(), hundredPerTenSeconds, 10) + b.allowed(clk.Now(), hundredPerTenSeconds, 10)
	require.Equal(t, 20, admitted, "writers that have not seen each other both admit a full burst")

	resetAt := a.allow(clk.Now(), hundredPerTenSeconds, 0, b.state).ResetAt
	require.Equal(t, start.Add(2*time.Second), resetAt)
	firstAllowed := resetAt.Add(-time.Second).Add(100 * time.Millisecond)

	for clk.Now().Before(firstAllowed) {
		require.Equal(t, 0, a.allowed(clk.Now(), hundredPerTenSeconds, 1, b.state), "at %s", clk.Now().Sub(start))
		require.Equal(t, 0, b.allowed(clk.Now(), hundredPerTenSeconds, 1, a.state), "at %s", clk.Now().Sub(start))
		clk.Tick(10 * time.Millisecond)
	}
	require.Equal(t, firstAllowed, clk.Now())

	for !clk.Now().After(start.Add(2 * time.Second)) {
		admitted += a.allowed(clk.Now(), hundredPerTenSeconds, 1, b.state)
		admitted += b.allowed(clk.Now(), hundredPerTenSeconds, 1, a.state)
		clk.Tick(10 * time.Millisecond)
	}
	require.Equal(t, 30, admitted, "a perfect limiter admits 10 at once and 1 per 100ms for 2 seconds")
}

func TestDecide_LimitChangeKeepsOldPrice(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}
	twoHundredPerTenSeconds := gcra.Rate{Limit: 200, Duration: 10 * time.Second}

	require.Equal(t, 10, w.allowed(clk.Now(), hundredPerTenSeconds, 10))
	require.Equal(t, 0, w.allowed(clk.Now(), twoHundredPerTenSeconds, 1), "the time already spent still fills the burst tolerance")

	clk.Tick(500 * time.Millisecond)
	require.Equal(t, 10, w.allowed(clk.Now(), twoHundredPerTenSeconds, 11), "half a second holds ten units at the new price")
}

func TestDecide_RoundsWhenLimitDoesNotDivideDuration(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))

	w := &writer{t: t}
	require.Equal(t, 2, w.allowed(clk.Now(), gcra.Rate{Limit: 15, Duration: 10 * time.Second}, 3), "the burst tolerance rounds up")

	w = &writer{t: t}
	require.True(t, w.allow(clk.Now(), gcra.Rate{Limit: 3, Duration: 10 * time.Second}, 3).Allowed, "a request of the whole limit still fits")
}

func TestDecide_LargeRatesDoNotOverflow(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))

	for _, tc := range []struct {
		name  string
		rate  gcra.Rate
		burst int64
	}{
		{name: "1e12 per second", rate: gcra.Rate{Limit: 1_000_000_000_000, Duration: time.Second}, burst: 100_000_000_000},
		{name: "162M per minute", rate: gcra.Rate{Limit: 162_000_000, Duration: time.Minute}, burst: 16_200_000},
		{name: "max limit per max duration", rate: gcra.Rate{Limit: math.MaxInt64 / 2, Duration: time.Duration(math.MaxInt64 / 2)}, burst: math.MaxInt64/20 + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &writer{t: t}
			first := w.allow(clk.Now(), tc.rate, 1)
			require.True(t, first.Allowed)
			require.Equal(t, tc.burst-1, first.Remaining)
			require.Equal(t, int64(1), first.Current)

			burst := w.allow(clk.Now(), tc.rate, tc.burst-1)
			require.True(t, burst.Allowed)
			require.Equal(t, int64(0), burst.Remaining)
			require.Equal(t, tc.burst, burst.Current)

			require.False(t, w.allow(clk.Now(), tc.rate, 1).Allowed)

			full := &writer{t: t}
			require.True(t, full.allow(clk.Now(), tc.rate, tc.rate.Limit).Allowed)
			require.False(t, full.allow(clk.Now(), tc.rate, 1).Allowed)
		})
	}
}

func TestDecide_OverflowIsAnError(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))

	_, err := gcra.Decide(clk.Now(), hundredPerTenSeconds, 1,
		gcra.State{SpentNs: math.MaxInt64 - 1, FreshStartNs: 0},
		gcra.State{SpentNs: 2, FreshStartNs: 0},
	)
	require.Error(t, err)

	_, err = gcra.Decide(clk.Now(), hundredPerTenSeconds, 1,
		gcra.State{SpentNs: math.MaxInt64 - clk.Now().UnixNano() + 1, FreshStartNs: clk.Now().UnixNano()},
		gcra.State{SpentNs: 0, FreshStartNs: 0},
	)
	require.Error(t, err)

	_, err = gcra.Decide(clk.Now(), gcra.Rate{Limit: math.MaxInt64/2 + 1, Duration: time.Duration(math.MaxInt64)}, 1,
		gcra.State{SpentNs: 0, FreshStartNs: 0},
		gcra.State{SpentNs: 0, FreshStartNs: 0},
	)
	require.Error(t, err)
}
