package ratelimit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

func newTestLimiter(t *testing.T) (*limiter, *clock.TestClock) {
	t.Helper()
	testClock := clock.NewTestClock(time.Unix(1_700_000_000, 0))
	return newTestLimiterOn(t, testClock, newMemoryStore(testClock), 1), testClock
}

// newTestLimiterOn returns a limiter on a shared clock and store, as one of
// several pods.
func newTestLimiterOn(t *testing.T, testClock *clock.TestClock, store Store, seed uint64) *limiter {
	t.Helper()
	l, err := newLimiter(Config{Clock: testClock, Store: store, Random: rand.New(rand.NewPCG(seed, 2))})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, l.Close()) })
	return l
}

func fingerprint(name string) [32]byte {
	return sha256.Sum256([]byte(name))
}

// afterMs returns `Duration × ln(from / to)` rounded up to whole
// milliseconds, the time a level takes to fall from `from` to `to`.
func afterMs(duration time.Duration, from, to float64) time.Duration {
	return time.Duration(math.Ceil(float64(duration.Milliseconds())*math.Log(from/to))) * time.Millisecond
}

// TestLimiterBurstFromIdleIsTheLimit guarantees that an idle limit admits
// `Limit` units at one instant, one by one or in one request, and not one
// more.
func TestLimiterBurstFromIdleIsTheLimit(t *testing.T) {
	ctx := context.Background()
	rate := Rate{Limit: 100, Duration: time.Minute}

	t.Run("one unit at a time", func(t *testing.T) {
		l, testClock := newTestLimiter(t)
		request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 1}
		for i := range rate.Limit {
			result, err := l.Allow(ctx, request)
			require.NoError(t, err)
			require.True(t, result.Allowed, "unit %d", i+1)
			require.Equal(t, rate.Limit-i-1, result.Remaining)
		}
		result, err := l.Allow(ctx, request)
		require.NoError(t, err)
		require.False(t, result.Allowed)
		require.Zero(t, result.Remaining)
		require.Equal(t, testClock.Now().Add(afterMs(rate.Duration, 100, 99)), result.ResetAt)
	})

	t.Run("the whole limit in one request", func(t *testing.T) {
		l, _ := newTestLimiter(t)
		tooMany := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: rate.Limit + 1}
		result, err := l.Allow(ctx, tooMany)
		require.NoError(t, err)
		require.False(t, result.Allowed)
		require.Equal(t, rate.Limit, result.Remaining)

		all := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: rate.Limit}
		result, err = l.Allow(ctx, all)
		require.NoError(t, err)
		require.True(t, result.Allowed)
		require.Zero(t, result.Remaining)
	})
}

// TestLimiterHoldsASteadyStreamToTheRate sends ten times the rate for a
// while and counts what it admits once the level has settled. A unit fits
// each time the level falls from `Limit` to `Limit − 1`, every
// `Duration × ln(Limit / (Limit − 1))`, which is `Limit / Duration` less
// about half a unit per `Duration`.
func TestLimiterHoldsASteadyStreamToTheRate(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	rate := Rate{Limit: 1_000, Duration: 10 * time.Second}
	request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 1}
	const step = time.Millisecond
	const warmUp = 5 * 10 * time.Second
	const measured = 100 * time.Second

	allowed := 0
	for elapsed := time.Duration(0); elapsed < warmUp+measured; elapsed += step {
		result, err := l.Allow(ctx, request)
		require.NoError(t, err)
		if result.Allowed && elapsed >= warmUp {
			allowed++
		}
		testClock.Tick(step)
	}
	want := measured.Seconds() * float64(rate.Limit) / rate.Duration.Seconds()
	require.InEpsilon(t, want, float64(allowed), 0.001, "admitted %d, the rate allows %.0f", allowed, want)
}

// TestLimiterRecoversRemainingFully guarantees that a full limit reports
// `remaining = Limit` again once its level is below `min(1, 0.01 × Limit)`,
// `Duration × ln(Limit)` later from 100 units up and `Duration × ln(100)`
// later below, and not before.
func TestLimiterRecoversRemainingFully(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		rate      Rate
		recovered time.Duration
	}{
		{Rate{Limit: 1, Duration: time.Minute}, afterMs(time.Minute, 100, 1)},
		{Rate{Limit: 10, Duration: time.Minute}, afterMs(time.Minute, 100, 1)},
		{Rate{Limit: 100, Duration: time.Minute}, afterMs(time.Minute, 100, 1)},
		{Rate{Limit: 1_000, Duration: time.Minute}, afterMs(time.Minute, 1_000, 1)},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d per %s", tc.rate.Limit, tc.rate.Duration), func(t *testing.T) {
			l, testClock := newTestLimiter(t)
			start := testClock.Now()
			result, err := l.Allow(ctx, Request{Fingerprint: fingerprint("a"), Rate: tc.rate, Units: tc.rate.Limit})
			require.NoError(t, err)
			require.True(t, result.Allowed)

			check := Request{Fingerprint: fingerprint("a"), Rate: tc.rate, Units: 0}
			testClock.Set(start.Add(tc.recovered - time.Millisecond))
			result, err = l.Allow(ctx, check)
			require.NoError(t, err)
			require.Less(t, result.Remaining, tc.rate.Limit, "the level is just above the cut")

			testClock.Set(start.Add(tc.recovered))
			result, err = l.Allow(ctx, check)
			require.NoError(t, err)
			require.Equal(t, Result{Allowed: true, Remaining: tc.rate.Limit, ResetAt: testClock.Now()}, result)
		})
	}
}

// TestLimiterOverAdmitsAtMostOnePercent is the regression test for the cut at
// one unit, which let a limit of 1 admit 2 at once and a limit of 10 reach a
// level of almost 11. From any level, a burst of single units over a few
// milliseconds must leave the level at most `Limit + min(1, 0.01 × Limit)`.
func TestLimiterOverAdmitsAtMostOnePercent(t *testing.T) {
	ctx := context.Background()
	for _, rate := range []Rate{{Limit: 1, Duration: time.Second}, {Limit: 10, Duration: time.Second}, {Limit: 1_000, Duration: time.Minute}} {
		cut := min(1, 0.01*float64(rate.Limit))
		for _, startLevel := range []float64{0, 0.009, 0.0999, 0.5, 0.9, 0.999, 1.5} {
			t.Run(fmt.Sprintf("%d per %s from %g", rate.Limit, rate.Duration, startLevel), func(t *testing.T) {
				l, testClock := newTestLimiter(t)
				var admitted []timedRequest
				if startLevel >= float64(rate.Limit) {
					t.Skip("one request cannot reach the level")
				}
				if startLevel > 0 {
					first := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: rate.Limit}
					result, err := l.Allow(ctx, first)
					require.NoError(t, err)
					require.True(t, result.Allowed)
					admitted = append(admitted, timedRequest{atMs: testClock.Now().UnixMilli(), units: first.Units, podID: l.podID})
					testClock.Tick(afterMs(rate.Duration, float64(rate.Limit), startLevel))
				}
				one := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 1}
				for range 3 {
					for range rate.Limit + 2 {
						result, err := l.Allow(ctx, one)
						require.NoError(t, err)
						if result.Allowed {
							admitted = append(admitted, timedRequest{atMs: testClock.Now().UnixMilli(), units: 1, podID: l.podID})
						}
					}
					level := referenceLevel(admitted, testClock.Now().UnixMilli(), rate.Duration)
					require.LessOrEqual(t, level, float64(rate.Limit)+cut, "the level after the burst")
					testClock.Tick(time.Millisecond)
				}
			})
		}
	}

	t.Run("a limit of 1 admits exactly 1 at once", func(t *testing.T) {
		l, _ := newTestLimiter(t)
		one := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 1, Duration: time.Second}, Units: 1}
		result, err := l.Allow(ctx, one)
		require.NoError(t, err)
		require.True(t, result.Allowed)
		result, err = l.Allow(ctx, one)
		require.NoError(t, err)
		require.False(t, result.Allowed)
	})
}

// TestLimiterPaysBackAMultiPodOverAdmission replays the example of the
// package documentation: two pods each admit 75 of 100 per minute before
// they see each other. Once both rows are visible, the level of 150 must
// fall to 99 before the next unit fits, `60 s × ln(150 / 99) ≈ 24.9 s`.
func TestLimiterPaysBackAMultiPodOverAdmission(t *testing.T) {
	ctx := context.Background()
	testClock := clock.NewTestClock(time.Unix(1_700_000_000, 0))
	store := newMemoryStore(testClock)
	podA := newTestLimiterOn(t, testClock, store, 1)
	podB := newTestLimiterOn(t, testClock, store, 2)
	rate := Rate{Limit: 100, Duration: time.Minute}
	request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 75}
	start := testClock.Now()

	for _, pod := range []*limiter{podA, podB} {
		result, err := pod.Allow(ctx, request)
		require.NoError(t, err)
		require.True(t, result.Allowed, "neither pod has seen the other")
	}
	require.NoError(t, podA.flush(ctx))
	require.NoError(t, podB.flush(ctx))

	nextUnitAt := start.Add(afterMs(rate.Duration, 150, 99))
	require.InDelta(t, 24.93*float64(time.Second), float64(nextUnitAt.Sub(start)), float64(10*time.Millisecond))
	one := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 1}
	for _, at := range []time.Duration{rowsStaleFor, 10 * time.Second, 20 * time.Second, nextUnitAt.Sub(start) - time.Millisecond} {
		testClock.Set(start.Add(at))
		for _, pod := range []*limiter{podA, podB} {
			result, err := pod.Allow(ctx, one)
			require.NoError(t, err)
			require.False(t, result.Allowed, "at %s", at)
			require.Zero(t, result.Remaining)
			require.Equal(t, nextUnitAt, result.ResetAt)
		}
	}

	testClock.Set(nextUnitAt)
	result, err := podA.Allow(ctx, one)
	require.NoError(t, err)
	require.True(t, result.Allowed)
}

func TestLimiterAllowManySpendsNothingIfOneRequestIsDenied(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	open := Request{Fingerprint: fingerprint("open"), Rate: rate, Units: 5}
	full := Request{Fingerprint: fingerprint("full"), Rate: Rate{Limit: 1, Duration: 10 * time.Second}, Units: 1}
	_, err := l.Allow(ctx, full)
	require.NoError(t, err)

	results, err := l.AllowMany(ctx, []Request{open, full})
	require.NoError(t, err)
	require.Equal(t, []Result{
		{Allowed: true, Remaining: 10, ResetAt: testClock.Now()},
		{Allowed: false, Remaining: 0, ResetAt: testClock.Now().Add(afterMs(10*time.Second, 100, 1))},
	}, results)

	open.Units = 10
	result, err := l.Allow(ctx, open)
	require.NoError(t, err)
	require.True(t, result.Allowed)
}

// TestLimiterAllowManyLocksASharedShardOnce uses two fingerprints in one shard.
func TestLimiterAllowManyLocksASharedShardOnce(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	results, err := l.AllowMany(ctx, []Request{
		{Fingerprint: [32]byte{7, 1}, Rate: rate, Units: 2},
		{Fingerprint: [32]byte{7, 2}, Rate: rate, Units: 0},
	})
	require.NoError(t, err)
	require.Equal(t, []Result{
		{Allowed: true, Remaining: 8, ResetAt: testClock.Now()},
		{Allowed: true, Remaining: 10, ResetAt: testClock.Now()},
	}, results)
}

func TestLimiterRejectsInvalidRequests(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLimiter(t)
	valid := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: time.Second}, Units: 1}
	_, err := l.Allow(ctx, valid)
	require.NoError(t, err)

	negativeUnits := valid
	negativeUnits.Units = -1
	invalidRate := valid
	invalidRate.Rate.Limit = 0
	tooLong := valid
	tooLong.Fingerprint = fingerprint("b")
	tooLong.Rate.Duration = maxDuration + time.Millisecond
	otherDuration := valid
	otherDuration.Rate.Duration = time.Minute
	for name, requests := range map[string][]Request{
		"duplicate fingerprints":             {valid, valid},
		"negative units":                     {negativeUnits},
		"invalid rate":                       {invalidRate},
		"a duration over 30 days":            {tooLong},
		"a fingerprint without the duration": {otherDuration},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := l.AllowMany(ctx, requests)
			require.Error(t, err)
		})
	}

	require.NoError(t, l.Close())
	_, err = l.Allow(ctx, valid)
	require.Error(t, err)
}

func TestLimiterPodIDsFollowTheSeed(t *testing.T) {
	podID := func(random *rand.Rand) string {
		testClock := clock.NewTestClock()
		l, err := newLimiter(Config{Clock: testClock, Store: newMemoryStore(testClock), Random: random})
		require.NoError(t, err)
		return l.podID
	}
	require.Regexp(t, "^[a-zA-Z0-9]{16}$", podID(nil))
	require.Equal(t, podID(rand.New(rand.NewPCG(1, 2))), podID(rand.New(rand.NewPCG(1, 2))))
	require.NotEqual(t, podID(nil), podID(nil))

	testClock := clock.NewTestClock()
	_, err := New(Config{Clock: nil, Store: newMemoryStore(testClock), Random: nil})
	require.Error(t, err)
	_, err = New(Config{Clock: testClock, Store: nil, Random: nil})
	require.Error(t, err)
}

// TestLimiterAdmitsExactlyTheLimitUnderConcurrency sends two shared limits in
// shards 1 and 2, each with a new limit in shard 3 that both share. A missing
// lock on any shard over-admits or crashes on a concurrent map write.
func TestLimiterAdmitsExactlyTheLimitUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLimiter(t)
	rate := Rate{Limit: 250, Duration: time.Hour}

	var allowed [2]atomic.Int64
	var wg sync.WaitGroup
	for worker := range 100 {
		shared := worker % 2
		wg.Go(func() {
			for i := range 10 {
				results, err := l.AllowMany(ctx, []Request{
					{Fingerprint: [32]byte{byte(shared + 1)}, Rate: rate, Units: 1},
					{Fingerprint: [32]byte{3, byte(worker), byte(i)}, Rate: rate, Units: 1},
				})
				require.NoError(t, err)
				if results[0].Allowed {
					allowed[shared].Add(1)
				}
			}
		})
	}
	wg.Wait()
	require.Equal(t, int64(250), allowed[0].Load())
	require.Equal(t, int64(250), allowed[1].Load())
}
