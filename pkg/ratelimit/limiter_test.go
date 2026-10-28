package ratelimit

import (
	"context"
	"crypto/sha256"
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
	l, err := newLimiter(Config{Clock: testClock, Store: newMemoryStore(testClock), Random: rand.New(rand.NewPCG(1, 2))})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, l.Close()) })
	return l, testClock
}

func fingerprint(name string) [32]byte {
	return sha256.Sum256([]byte(name))
}

func TestLimiterAllowsTheLimitThenDeniesUntilUnitsAreFree(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	start := testClock.Now()
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 1}

	for i := range 10 {
		result, err := l.Allow(ctx, request)
		require.NoError(t, err)
		require.Equal(t, Result{Allowed: true, Remaining: int64(9 - i), ResetAt: start.Add(time.Duration(i+1) * time.Second)}, result)
	}
	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: false, Remaining: 0, ResetAt: start.Add(10 * time.Second)}, result)

	check := request
	check.Units = 0
	result, err = l.Allow(ctx, check)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 0, ResetAt: start.Add(10 * time.Second)}, result)

	testClock.Tick(time.Second)
	result, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 0, ResetAt: start.Add(11 * time.Second)}, result)
}

// TestLimiterAdmitsTheLimitWhenDurationIsNotAMultipleOfLimit guarantees that
// rounding each unit up to whole nanoseconds costs no unit: `Limit` single
// units fit, the next one does not, and `Remaining` counts down by one.
func TestLimiterAdmitsTheLimitWhenDurationIsNotAMultipleOfLimit(t *testing.T) {
	cases := []struct {
		name string
		rate Rate
		gap  time.Duration
	}{
		{"3 per second at one instant", Rate{Limit: 3, Duration: time.Second}, 0},
		{"3 per second 1µs apart", Rate{Limit: 3, Duration: time.Second}, time.Microsecond},
		{"7 per minute 1µs apart", Rate{Limit: 7, Duration: time.Minute}, time.Microsecond},
		{"4 per minute 1µs apart", Rate{Limit: 4, Duration: time.Minute}, time.Microsecond},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			l, testClock := newTestLimiter(t)
			request := Request{Fingerprint: fingerprint("a"), Rate: tc.rate, Units: 1}
			for i := range tc.rate.Limit {
				result, err := l.Allow(ctx, request)
				require.NoError(t, err)
				require.True(t, result.Allowed, "unit %d", i+1)
				require.Equal(t, tc.rate.Limit-i-1, result.Remaining, "unit %d", i+1)
				testClock.Tick(tc.gap)
			}
			result, err := l.Allow(ctx, request)
			require.NoError(t, err)
			require.False(t, result.Allowed)
			require.Zero(t, result.Remaining)
		})
	}
}

func TestLimiterAllowManySpendsNothingIfOneRequestIsDenied(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	start := testClock.Now()
	open := Request{Fingerprint: fingerprint("open"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 5}
	full := Request{Fingerprint: fingerprint("full"), Rate: Rate{Limit: 1, Duration: 10 * time.Second}, Units: 1}
	_, err := l.Allow(ctx, full)
	require.NoError(t, err)

	results, err := l.AllowMany(ctx, []Request{open, full})
	require.NoError(t, err)
	require.Equal(t, []Result{
		{Allowed: true, Remaining: 10, ResetAt: start},
		{Allowed: false, Remaining: 0, ResetAt: start.Add(10 * time.Second)},
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
		{Allowed: true, Remaining: 8, ResetAt: testClock.Now().Add(2 * time.Second)},
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
	otherDuration := valid
	otherDuration.Rate.Duration = time.Minute
	for name, requests := range map[string][]Request{
		"duplicate fingerprints":             {valid, valid},
		"negative units":                     {negativeUnits},
		"invalid rate":                       {invalidRate},
		"a fingerprint without the duration": {otherDuration},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := l.AllowMany(ctx, requests)
			require.Error(t, err)
		})
	}

	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("b"), Rate: Rate{Limit: 1, Duration: math.MaxInt64}, Units: 1})
	require.Error(t, err, "reset_at overflows int64")

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
