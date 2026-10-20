package gcra_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/gcra"
)

var tenPerTenSeconds = gcra.Rate{Limit: 10, Duration: 10 * time.Second}

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

	require.Equal(t, 10, w.allowed(clk.Now(), tenPerTenSeconds, 11))

	denied := w.allow(clk.Now(), tenPerTenSeconds, 1)
	require.False(t, denied.Allowed)
	require.Equal(t, int64(0), denied.Remaining)
	require.Equal(t, int64(11), denied.Current)
	require.Equal(t, clk.Now().Add(10*time.Second), denied.ResetAt)

	for range 5 {
		clk.Tick(time.Second)
		require.Equal(t, 1, w.allowed(clk.Now(), tenPerTenSeconds, 3))
	}

	clk.Tick(500 * time.Millisecond)
	require.Equal(t, 0, w.allowed(clk.Now(), tenPerTenSeconds, 1))
}

func TestDecide_ReportsRemainingAndReset(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	first := w.allow(clk.Now(), tenPerTenSeconds, 3)
	require.True(t, first.Allowed)
	require.Equal(t, int64(7), first.Remaining)
	require.Equal(t, int64(3), first.Current)
	require.Equal(t, clk.Now().Add(3*time.Second), first.ResetAt)

	clk.Tick(1500 * time.Millisecond)
	check := w.allow(clk.Now(), tenPerTenSeconds, 0)
	require.True(t, check.Allowed)
	require.Equal(t, int64(8), check.Remaining)
	require.Equal(t, int64(2), check.Current)
}

func TestDecide_ZeroCostChangesNothing(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	before := gcra.State{SpentNs: 0, FreshStartNs: 0}

	decision, err := gcra.Decide(clk.Now(), tenPerTenSeconds, 0, before, before)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.False(t, decision.Clamped)
	require.Equal(t, before, decision.Writer)
	require.Equal(t, int64(10), decision.Remaining)
}

func TestDecide_CostAboveLimitIsDenied(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	denied := w.allow(clk.Now(), tenPerTenSeconds, 11)
	require.False(t, denied.Allowed)
	require.Equal(t, int64(10), denied.Remaining)
	require.Equal(t, int64(11), denied.Current)
	require.Equal(t, gcra.State{SpentNs: 0, FreshStartNs: 0}, w.state)

	denied = w.allow(clk.Now(), tenPerTenSeconds, math.MaxInt64)
	require.False(t, denied.Allowed)
	require.Equal(t, int64(math.MaxInt64), denied.Current)
}

func TestDecide_IdleClampForgetsIdleTime(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	require.Equal(t, 10, w.allowed(clk.Now(), tenPerTenSeconds, 10))

	clk.Tick(100 * time.Second)
	decision, err := gcra.Decide(clk.Now(), tenPerTenSeconds, 1, w.state, gcra.State{SpentNs: 0, FreshStartNs: 0})
	require.NoError(t, err)
	require.True(t, decision.Clamped)
	require.Equal(t, clk.Now().UnixNano()-10*time.Second.Nanoseconds(), decision.Writer.FreshStartNs)

	require.Equal(t, 10, w.allowed(clk.Now(), tenPerTenSeconds, 20), "idle time beyond one burst must not be banked")
}

func TestDecide_ConcurrentClampsCombineByMax(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	a, b := &writer{t: t}, &writer{t: t}

	require.Equal(t, 5, a.allowed(clk.Now(), tenPerTenSeconds, 5))
	require.Equal(t, 5, b.allowed(clk.Now(), tenPerTenSeconds, 5, a.state))

	clk.Tick(100 * time.Second)
	aSynced, bSynced := a.state, b.state
	require.Equal(t, 5, a.allowed(clk.Now(), tenPerTenSeconds, 5, bSynced))
	require.Equal(t, 5, b.allowed(clk.Now(), tenPerTenSeconds, 5, aSynced))
	require.Equal(t, a.state.FreshStartNs, b.state.FreshStartNs)

	after := a.allow(clk.Now(), tenPerTenSeconds, 0, b.state)
	require.Equal(t, clk.Now().Add(10*time.Second), after.ResetAt, "both clamps describe the same baseline, so it counts once")
	require.Equal(t, int64(0), after.Remaining)

	clk.Tick(time.Second)
	require.Equal(t, 1, a.allowed(clk.Now(), tenPerTenSeconds, 2, b.state))
}

func TestDecide_StaleWritersPayBackOverAdmission(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	start := clk.Now()
	a, b := &writer{t: t}, &writer{t: t}

	admitted := a.allowed(clk.Now(), tenPerTenSeconds, 10) + b.allowed(clk.Now(), tenPerTenSeconds, 10)
	require.Equal(t, 20, admitted, "writers that have not seen each other both admit a full burst")

	resetAt := a.allow(clk.Now(), tenPerTenSeconds, 0, b.state).ResetAt
	require.Equal(t, start.Add(20*time.Second), resetAt)
	firstAllowed := resetAt.Add(-10 * time.Second).Add(time.Second)

	for clk.Now().Before(firstAllowed) {
		require.Equal(t, 0, a.allowed(clk.Now(), tenPerTenSeconds, 1, b.state), "at %s", clk.Now().Sub(start))
		require.Equal(t, 0, b.allowed(clk.Now(), tenPerTenSeconds, 1, a.state), "at %s", clk.Now().Sub(start))
		clk.Tick(100 * time.Millisecond)
	}
	require.Equal(t, firstAllowed, clk.Now())

	for !clk.Now().After(start.Add(20 * time.Second)) {
		admitted += a.allowed(clk.Now(), tenPerTenSeconds, 1, b.state)
		admitted += b.allowed(clk.Now(), tenPerTenSeconds, 1, a.state)
		clk.Tick(100 * time.Millisecond)
	}
	require.Equal(t, 30, admitted, "a perfect limiter admits 10 at once and 1 per second for 20 seconds")
}

func TestDecide_LimitChangeKeepsOldPrice(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}
	twentyPerTenSeconds := gcra.Rate{Limit: 20, Duration: 10 * time.Second}

	require.Equal(t, 10, w.allowed(clk.Now(), tenPerTenSeconds, 10))
	require.Equal(t, 0, w.allowed(clk.Now(), twentyPerTenSeconds, 1), "the time already spent still fills the burst")

	clk.Tick(5 * time.Second)
	require.Equal(t, 10, w.allowed(clk.Now(), twentyPerTenSeconds, 11), "five free seconds hold ten units at the new price")
}

func TestDecide_FullBurstWhenLimitDoesNotDivideDuration(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	w := &writer{t: t}

	require.Equal(t, 3, w.allowed(clk.Now(), gcra.Rate{Limit: 3, Duration: 10 * time.Second}, 4))
}

func TestDecide_LargeRatesDoNotOverflow(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))

	for _, tc := range []struct {
		name string
		rate gcra.Rate
	}{
		{name: "1e12 per second", rate: gcra.Rate{Limit: 1_000_000_000_000, Duration: time.Second}},
		{name: "162M per minute", rate: gcra.Rate{Limit: 162_000_000, Duration: time.Minute}},
		{name: "max limit per max duration", rate: gcra.Rate{Limit: math.MaxInt64 / 2, Duration: time.Duration(math.MaxInt64 / 2)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &writer{t: t}
			first := w.allow(clk.Now(), tc.rate, 1)
			require.True(t, first.Allowed)
			require.Equal(t, tc.rate.Limit-1, first.Remaining)
			require.Equal(t, int64(1), first.Current)

			burst := w.allow(clk.Now(), tc.rate, tc.rate.Limit-1)
			require.True(t, burst.Allowed)
			require.Equal(t, int64(0), burst.Remaining)
			require.Equal(t, tc.rate.Limit, burst.Current)

			require.False(t, w.allow(clk.Now(), tc.rate, 1).Allowed)
		})
	}
}

func TestDecide_OverflowIsAnError(t *testing.T) {
	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))

	_, err := gcra.Decide(clk.Now(), tenPerTenSeconds, 1,
		gcra.State{SpentNs: math.MaxInt64 - 1, FreshStartNs: 0},
		gcra.State{SpentNs: 2, FreshStartNs: 0},
	)
	require.Error(t, err)

	_, err = gcra.Decide(clk.Now(), tenPerTenSeconds, 1,
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
