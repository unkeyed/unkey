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
	l, err := newLimiter(Config{Clock: testClock, Store: newMemoryStore(), Random: rand.New(rand.NewPCG(1, 2))})
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

	negativeUnits := valid
	negativeUnits.Units = -1
	invalidRate := valid
	invalidRate.Rate.Limit = 0
	for name, requests := range map[string][]Request{
		"duplicate fingerprints": {valid, valid},
		"negative units":         {negativeUnits},
		"invalid rate":           {invalidRate},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := l.AllowMany(ctx, requests)
			require.Error(t, err)
		})
	}

	_, err := l.Allow(ctx, Request{Fingerprint: fingerprint("b"), Rate: Rate{Limit: 1, Duration: math.MaxInt64}, Units: 1})
	require.Error(t, err, "reset_at overflows int64")

	require.NoError(t, l.Close())
	_, err = l.Allow(ctx, valid)
	require.Error(t, err)
}

// TestLimiterStartsANewSessionWhenTheLifetimeEnds uses 1 unit per second and
// keeps `reset_at` ahead of now, so only the lifetime can end the session.
func TestLimiterStartsANewSessionWhenTheLifetimeEnds(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	start := testClock.Now()
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 3600, Duration: time.Hour}, Units: 1800}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	e := l.shards[shardIndex(request.Fingerprint)].entries[request.Fingerprint]
	first := e.sessions[e.liveID]
	lifetime := time.Duration(e.liveSessionEndsAtNs - start.UnixNano())
	require.GreaterOrEqual(t, lifetime, 12*time.Minute)
	require.Less(t, lifetime, 15*time.Minute)

	request.Units = 1
	testClock.Set(time.Unix(0, e.liveSessionEndsAtNs-1))
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, first.id, e.liveID)

	testClock.Tick(time.Nanosecond)
	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.NotEqual(t, first.id, e.liveID)
	require.Len(t, e.sessions, 2)
	require.Equal(t, session{id: first.id, anchorAtNs: first.anchorAtNs, consumedNs: 1801 * time.Second.Nanoseconds(), own: true, committedNs: 0}, e.sessions[first.id])
	require.Equal(t, start.Add(1802*time.Second), result.ResetAt)
}

func TestLimiterNewSessionGivesUniqueIDsAndLifetimesOf12To15Minutes(t *testing.T) {
	l, _ := newTestLimiter(t)
	ids := make(map[string]struct{})
	shortest, longest := time.Hour, time.Duration(0)
	for range 10_000 {
		id, lifetime := l.newSession()
		require.Regexp(t, "^[a-zA-Z0-9]{16}$", id)
		ids[id] = struct{}{}
		shortest, longest = min(shortest, lifetime), max(longest, lifetime)
	}
	require.Len(t, ids, 10_000)
	require.GreaterOrEqual(t, shortest, 12*time.Minute)
	require.Less(t, shortest, 12*time.Minute+10*time.Second)
	require.Less(t, longest, 15*time.Minute)
	require.Greater(t, longest, 15*time.Minute-10*time.Second)
}

func TestLimiterSessionIDsFollowTheSeed(t *testing.T) {
	sessionID := func(random *rand.Rand) string {
		l, err := newLimiter(Config{Clock: clock.NewTestClock(), Store: newMemoryStore(), Random: random})
		require.NoError(t, err)
		id, _ := l.newSession()
		return id
	}
	require.Equal(t, sessionID(rand.New(rand.NewPCG(1, 2))), sessionID(rand.New(rand.NewPCG(1, 2))))
	require.NotEqual(t, sessionID(nil), sessionID(nil))

	_, err := New(Config{Clock: nil, Store: newMemoryStore(), Random: nil})
	require.Error(t, err)
	_, err = New(Config{Clock: clock.NewTestClock(), Store: nil, Random: nil})
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
