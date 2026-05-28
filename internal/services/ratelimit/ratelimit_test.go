package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestRatelimit_SlidingWindowDecision locks in the math of the sliding window
// algorithm. Each case pre-seeds the current and previous counters and
// picks a req.Time that produces the desired elapsed fraction, so the decision
// path runs against exactly the inputs under test.
func TestRatelimit_SlidingWindowDecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		curCount  int64
		prevCount int64
		elapsed   float64 // fraction of current window already elapsed (0.0–1.0)
		cost      int64
		limit     int64
		wantPass  bool
		wantRem   int64
	}{
		{name: "empty bucket allows", curCount: 0, prevCount: 0, elapsed: 0.0, cost: 1, limit: 10, wantPass: true, wantRem: 9},
		{name: "just below limit allows", curCount: 9, prevCount: 0, elapsed: 0.0, cost: 1, limit: 10, wantPass: true, wantRem: 0},
		{name: "exactly at limit allows", curCount: 10, prevCount: 0, elapsed: 0.0, cost: 0, limit: 10, wantPass: true, wantRem: 0},
		{name: "one over limit denies", curCount: 10, prevCount: 0, elapsed: 0.0, cost: 1, limit: 10, wantPass: false, wantRem: 0},
		// Sliding window: prev=10, elapsed=0.5 → prev contributes 5. 5+5+1=11 > 10.
		{name: "prev window contributes early in new window", curCount: 5, prevCount: 10, elapsed: 0.5, cost: 1, limit: 10, wantPass: false, wantRem: 0},
		// prev=10, elapsed=0.99 → prev contributes ~0. 5+0+1=6 <= 10.
		{name: "prev window decays as window advances", curCount: 5, prevCount: 10, elapsed: 0.99, cost: 1, limit: 10, wantPass: true, wantRem: 4},
		{name: "zero cost allows even at limit", curCount: 10, prevCount: 0, elapsed: 0.0, cost: 0, limit: 10, wantPass: true, wantRem: 0},
		{name: "large cost denies", curCount: 0, prevCount: 0, elapsed: 0.0, cost: 100, limit: 10, wantPass: false, wantRem: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clk := clock.NewTestClock()
			svc, err := New(Config{
				Clock: clk, Counter: counter.NewMemory(), DB: newTestDB(t), Region: "test-region",
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = svc.Close() })

			// Per-test workspace ID so background global sync can't pollute
			// these counters from rows another test wrote to the shared MySQL.
			ws := uid.New(uid.WorkspacePrefix)
			duration := time.Minute
			durationMs := duration.Milliseconds()

			// Derive a request time that produces the desired elapsed fraction.
			curSeq := calculateSequence(clk.Now(), duration)
			windowStartMs := curSeq * durationMs
			elapsedMs := int64(tt.elapsed * float64(durationMs))
			reqTime := time.UnixMilli(windowStartMs + elapsedMs)

			// Seed both windows so prepareCheck takes the local fast path.
			curKey := counterKey{workspaceID: ws, namespace: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq}
			prevKey := counterKey{workspaceID: ws, namespace: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq - 1}
			cur := svc.loadCounter(curKey)
			cur.val.Store(tt.curCount)
			prev := svc.loadCounter(prevKey)
			prev.val.Store(tt.prevCount)

			resp, err := svc.Ratelimit(context.Background(), RatelimitRequest{
				WorkspaceID: ws,
				Namespace:   "ns",
				Identifier:  "id",
				Limit:       tt.limit,
				Duration:    duration,
				Cost:        tt.cost,
				Time:        reqTime,
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantPass, resp.Success, "Success")
			require.Equal(t, tt.wantRem, resp.Remaining, "Remaining")
		})
	}
}

// TestRatelimit_DenialSetsStrictUntil asserts a denied request sets the
// strict-mode deadline to req.Time + req.Duration for its (workspace,
// namespace, identifier, duration) tuple.
func TestRatelimit_DenialSetsStrictUntil(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	svc, err := New(Config{
		Clock: clk, Counter: counter.NewMemory(), DB: newTestDB(t), Region: "test-region",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	duration := time.Minute
	req := RatelimitRequest{
		WorkspaceID: uid.New(uid.WorkspacePrefix),
		Namespace:   "ns",
		Identifier:  "id",
		Limit:       1,
		Duration:    duration,
		Cost:        1,
		Time:        clk.Now(),
	}

	resp, err := svc.Ratelimit(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Success, "first request at limit=1 should pass")

	resp, err = svc.Ratelimit(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.Success, "second request should be denied")

	sk := strictKey{workspaceID: req.WorkspaceID, namespace: req.Namespace, identifier: req.Identifier, durationMs: duration.Milliseconds()}
	got := svc.loadStrictUntil(sk)
	want := req.Time.Add(duration).UnixMilli()
	require.Equal(t, want, got, "strictUntil should equal req.Time + req.Duration")
}

// TestRatelimit_StrictModeForcesOriginFetch asserts that once strict mode is
// active, every request fetches the current window from origin before deciding.
// We verify by observing local state converge to a seeded origin value — the
// only way that can happen is if origin was consulted.
func TestRatelimit_StrictModeForcesOriginFetch(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	origin := counter.NewMemory()
	svc, err := New(Config{
		Clock: clk, Counter: origin, DB: newTestDB(t), Region: "test-region",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	ws := uid.New(uid.WorkspacePrefix)
	duration := time.Minute
	durationMs := duration.Milliseconds()
	reqTime := clk.Now()
	curSeq := calculateSequence(reqTime, duration)

	curKey := counterKey{workspaceID: ws, namespace: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq}
	prevKey := counterKey{workspaceID: ws, namespace: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq - 1}

	// Warm local windows so the first denial does not trigger a cold-window fetch.
	cur := svc.loadCounter(curKey)
	prev := svc.loadCounter(prevKey)

	req := RatelimitRequest{
		WorkspaceID: ws,
		Namespace:   "ns",
		Identifier:  "id",
		Limit:       1,
		Duration:    duration,
		Cost:        1,
		Time:        reqTime,
	}

	_, err = svc.Ratelimit(context.Background(), req)
	require.NoError(t, err)
	_, err = svc.Ratelimit(context.Background(), req)
	require.NoError(t, err, "second call denies and sets strictUntil")

	// Poison origin with a value higher than local. If strict mode forces a
	// fetch, the next prepareCheck will CAS-merge this into local.
	const originValue int64 = 999
	_, err = origin.Increment(context.Background(), curKey.redisKey(), originValue)
	require.NoError(t, err)
	_, err = origin.Increment(context.Background(), prevKey.redisKey(), originValue)
	require.NoError(t, err)

	_, err = svc.Ratelimit(context.Background(), req)
	require.NoError(t, err)

	require.GreaterOrEqual(t, cur.val.Load(), originValue, "current window should have picked up origin value")
	require.Equal(t, int64(0), prev.val.Load(), "strict mode should not refetch the previous window")
}

func TestRatelimit_ReplayMarksEntryFresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clk := clock.NewTestClock()
	origin := counter.NewMemory()
	svc, err := New(Config{
		Clock: clk, Counter: origin, DB: newTestDB(t), Region: "test-region",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	ws := uid.New(uid.WorkspacePrefix)
	duration := time.Minute
	durationMs := duration.Milliseconds()
	reqTime := clk.Now()
	curSeq := calculateSequence(reqTime, duration)
	curKey := counterKey{workspaceID: ws, namespace: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq}

	req := RatelimitRequest{
		WorkspaceID: ws,
		Namespace:   "ns",
		Identifier:  "id",
		Limit:       10,
		Duration:    duration,
		Cost:        1,
		Time:        reqTime,
	}

	resp, err := svc.Ratelimit(ctx, req)
	require.NoError(t, err)
	require.True(t, resp.Success)

	clk.Tick(time.Second)
	req.Time = clk.Now()
	resp, err = svc.Ratelimit(ctx, req)
	require.NoError(t, err)
	require.True(t, resp.Success)

	entryValue, ok := svc.counters.Load(curKey)
	require.True(t, ok)
	entry := entryValue.(*counterEntry)
	require.Eventually(t, func() bool {
		return entry.originFreshUntilMs.Load() >= req.Time.Add(originFreshDuration).UnixMilli()
	}, 3*time.Second, 50*time.Millisecond, "successful replay should keep origin freshness current")
}

// TestRatelimitMany_RollsBackOnPartialFailure asserts that when any entry in
// a batch exceeds its limit, none of the batch's counters are incremented —
// matching the all-or-nothing contract expected by callers.
func TestRatelimitMany_RollsBackOnPartialFailure(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	svc, err := New(Config{
		Clock: clk, Counter: counter.NewMemory(), DB: newTestDB(t), Region: "test-region",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	ws := uid.New(uid.WorkspacePrefix)
	duration := time.Minute
	durationMs := duration.Milliseconds()
	reqTime := clk.Now()
	curSeq := calculateSequence(reqTime, duration)

	// Seed warm windows so prepareCheck is local-only.
	aCur := counterKey{workspaceID: ws, namespace: "A", identifier: "user", durationMs: durationMs, sequence: curSeq}
	aPrev := counterKey{workspaceID: ws, namespace: "A", identifier: "user", durationMs: durationMs, sequence: curSeq - 1}
	bCur := counterKey{workspaceID: ws, namespace: "B", identifier: "user", durationMs: durationMs, sequence: curSeq}
	bPrev := counterKey{workspaceID: ws, namespace: "B", identifier: "user", durationMs: durationMs, sequence: curSeq - 1}
	a := svc.loadCounter(aCur)
	_ = svc.loadCounter(aPrev)
	b := svc.loadCounter(bCur)
	_ = svc.loadCounter(bPrev)

	// A is far below its limit. B is at 9 of 10 — a cost of 5 will exceed it.
	b.val.Store(9)

	reqs := []RatelimitRequest{
		{WorkspaceID: ws, Namespace: "A", Identifier: "user", Limit: 10, Duration: duration, Cost: 1, Time: reqTime},
		{WorkspaceID: ws, Namespace: "B", Identifier: "user", Limit: 10, Duration: duration, Cost: 5, Time: reqTime},
	}

	resp, err := svc.RatelimitMany(context.Background(), reqs)
	require.NoError(t, err)
	require.Len(t, resp, 2)

	require.True(t, resp[0].Success, "A individually passed")
	require.False(t, resp[1].Success, "B individually exceeded")

	require.Equal(t, int64(0), a.val.Load(), "A counter must be unchanged after batch failure")
	require.Equal(t, int64(9), b.val.Load(), "B counter must be rolled back after batch failure")
}

func TestRatelimitMany_DoesNotMutateRequests(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	svc, err := New(Config{
		Clock: clk, Counter: counter.NewMemory(), DB: newTestDB(t), Region: "test-region",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	reqs := []RatelimitRequest{
		{
			WorkspaceID: uid.New(uid.WorkspacePrefix),
			Namespace:   "ns",
			Identifier:  "user",
			Limit:       10,
			Duration:    time.Minute,
			Cost:        1,
		},
	}

	_, err = svc.RatelimitMany(context.Background(), reqs)
	require.NoError(t, err)
	require.True(t, reqs[0].Time.IsZero(), "RatelimitMany must not write normalized timestamps into caller input")
}

// TestRatelimit_MidWindowBurstWhenLocalCurIsStale guards the fix for the
// production pattern where a 60s sliding window blocked almost every request
// at the boundary but let bursts through near the middle of the window.
//
// Numbers are taken straight from the offending production tuple:
//
//	namespace=tpm_ratelimit_prod, limit=162000, duration=60s, cost=1500
//
// One replica receives a token of traffic at window start so its local cur
// becomes warm at val=cost. Other replicas accept the bulk of the traffic and
// replay it to the shared origin (modeled here by writing the origin directly).
// 30s into the window, the previously-warm replica receives a burst.
//
// Pre-fix, origin freshness lasted as long as the window itself, so
// EnsureFreshFromOrigin would see the last origin fetch as still fresh and
// skip the refresh; the replica read val=cost and freely allowed the burst.
// Post-fix, originFreshDuration is shorter than the window, so the 30s tick
// crosses the refresh gate, the replica pulls the real origin, and the burst
// is capped at the actual remaining budget.
func TestRatelimit_MidWindowBurstWhenLocalCurIsStale(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// Anchor request time at the start of a window so the sliding window math
	// places us exactly at elapsed=0.5 after a 30s tick.
	windowStart := time.Now().UTC().Truncate(time.Minute)
	clk := clock.NewTestClock(windowStart)
	origin := counter.NewMemory()

	svc, err := New(Config{
		Clock: clk, Counter: origin, DB: newTestDB(t), Region: "us-east-1",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	workspaceID := uid.New(uid.WorkspacePrefix)
	namespace := uid.New(uid.TestPrefix)
	identifier := uid.New(uid.TestPrefix)
	duration := time.Minute
	durationMs := duration.Milliseconds()

	const (
		limit int64 = 162000
		cost  int64 = 1500
	)

	curSeq := calculateSequence(clk.Now(), duration)
	curKey := counterKey{
		workspaceID: workspaceID, namespace: namespace, identifier: identifier,
		durationMs: durationMs, sequence: curSeq,
	}

	req := RatelimitRequest{
		WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
		Limit: limit, Duration: duration, Cost: cost, Time: clk.Now(),
	}

	// First request at t=0 hydrates this replica's cur from origin (=0) and
	// commits a local increment to val=cost. originFreshUntilMs is now t=0 plus originFreshDuration.
	resp, err := svc.Ratelimit(ctx, req)
	require.NoError(t, err)
	require.True(t, resp.Success, "first request at window start passes")

	// Other replicas process the bulk of traffic for this window and replay
	// it to the shared origin. Directly seed the full origin value to keep the
	// test deterministic and decoupled from this replica's async replay worker.
	// After this, shared origin records 160000 out of the 162000 limit, leaving
	// room for exactly one more cost=1500 request.
	const sharedOriginCount int64 = 160000
	_, err = origin.Increment(ctx, curKey.redisKey(), sharedOriginCount, duration*3)
	require.NoError(t, err)

	// Jump to the middle of the window. Pre-fix, 30s was before freshUntil (==
	// window duration) meant EnsureFreshFromOrigin would skip the refresh and
	// reuse the stale local val=cost; post-fix, 30s exceeds originFreshUntilMs
	// so the next request triggers a fetch_stale that pulls the true origin.
	clk.Tick(30 * time.Second)
	req.Time = clk.Now()

	// True remaining budget given the shared origin is (limit - origin) / cost =
	// (162000 - 160000) / 1500 = 1 request. Any more than 1 pass proves the
	// stale local cur let the replica burst through the shared limit.
	const wantMaxPasses = 1
	var passed int
	for range 20 {
		resp, err = svc.Ratelimit(ctx, req)
		require.NoError(t, err)
		if resp.Success {
			passed++
		}
	}

	require.LessOrEqual(t, passed, wantMaxPasses,
		"mid-window burst-through: replica saw stale local cur during the window and allowed %d cost=%d requests; shared origin only had room for %d",
		passed, cost, wantMaxPasses)
}

// TestRatelimit_MultiReplicaMidWindowBurstExceedsLimit keeps the production
// burst shape that originally exposed stale local replicas. The limiter is
// still best-effort rather than a Redis reservation on every request, but replay
// freshness plus aggressive stale reads should bound the overshoot well below
// the unsynchronized case where every replica independently spends a full local
// window.
func TestRatelimit_MultiReplicaMidWindowBurstExceedsLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	windowStart := time.Now().UTC().Truncate(time.Minute)
	clk := clock.NewTestClock(windowStart)
	origin := counter.NewMemory()
	db := newTestDB(t)

	const (
		replicas                   = 5
		burstRequestsPerNode       = 200
		limit                int64 = 162000
		cost                 int64 = 1500
	)
	duration := time.Minute

	services := make([]*service, replicas)
	for i := range services {
		svc, err := New(Config{
			Clock: clk, Counter: origin, DB: db, Region: fmt.Sprintf("region-%d", i),
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = svc.Close() })
		services[i] = svc
	}

	workspaceID := uid.New(uid.WorkspacePrefix)
	namespace := uid.New(uid.TestPrefix)
	identifier := uid.New(uid.TestPrefix)
	// Advance request time through the burst so the test exercises replay-updated
	// freshness and repeated stale refreshes rather than a Redis reservation model.
	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: cost, Time: clk.Tick(20 * time.Millisecond),
		}
	}

	for _, svc := range services {
		resp, err := svc.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success, "warmup request at window start must pass")
	}

	clk.Tick(30 * time.Second)

	var (
		passedRequests atomic.Int64
		wg             sync.WaitGroup
		start          = make(chan struct{})
	)
	for _, svc := range services {
		wg.Add(1)
		go func(svc *service) {
			defer wg.Done()
			<-start
			for range burstRequestsPerNode {
				resp, err := svc.Ratelimit(ctx, makeReq())
				if err == nil && resp.Success {
					passedRequests.Add(1)
				}
			}
		}(svc)
	}
	close(start)
	wg.Wait()

	passedTokens := passedRequests.Load() * cost
	require.LessOrEqual(t, passedTokens, 3*limit,
		"multi-replica mid-window burst: %d replicas admitted %d cost=%d requests = %d tokens against a %d-token limit",
		replicas, passedRequests.Load(), cost, passedTokens, limit)
}

// TestRatelimitMany_LongWindowLowTrafficRefreshesStaleReplica models the
// production failure mode for daily hard caps with sparse traffic: one replica
// hydrates a long-lived window while the count is low, another replica advances
// the shared origin, then the first replica receives another request much later.
//
// For long windows, a stale local view can persist for hours because the window
// does not naturally rotate. The service must refresh old local state from the
// shared origin before answering; otherwise a daily cap spread across replicas
// can exceed the limit while every replica still returns passed=true locally.
func TestRatelimitMany_LongWindowLowTrafficRefreshesStaleReplica(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Hour).Add(5 * time.Minute)
	clk := clock.NewTestClock(start)
	origin := counter.NewMemory()

	staleReplica, err := New(Config{
		Clock: clk, Counter: origin, DB: newTestDB(t), Region: "us-east-1",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = staleReplica.Close() })

	workspaceID := uid.New(uid.WorkspacePrefix)
	identifier := "low-rps-user"
	day := 24 * time.Hour
	hour := time.Hour
	makeReq := func(namespace string, limit int64, duration time.Duration, cost int64) RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			Cost:        cost,
			Time:        clk.Now(),
		}
	}
	makeKey := func(namespace string, duration time.Duration) counterKey {
		return counterKey{
			workspaceID: workspaceID,
			namespace:   namespace,
			identifier:  identifier,
			durationMs:  duration.Milliseconds(),
			sequence:    calculateSequence(start, duration),
		}
	}

	readReqs := []RatelimitRequest{
		makeReq("tokens.day", 10, day, 0),
		makeReq("tokens.hour", 20, hour, 0),
	}

	// The stale replica sees the key first and hydrates both windows at zero.
	resp, err := staleReplica.RatelimitMany(ctx, readReqs)
	require.NoError(t, err)
	require.Len(t, resp, 2)
	require.True(t, resp[0].Success)
	require.Equal(t, int64(10), resp[0].Remaining)
	require.True(t, resp[1].Success)
	require.Equal(t, int64(20), resp[1].Remaining)

	// Other replicas process traffic and successfully replay it to the shared
	// origin. We write the origin directly so the test is deterministic and not
	// coupled to replay-worker scheduling.
	dayKey := makeKey("tokens.day", day)
	hourKey := makeKey("tokens.hour", hour)
	_, err = origin.Increment(ctx, dayKey.redisKey(), 8, day*3)
	require.NoError(t, err)
	_, err = origin.Increment(ctx, hourKey.redisKey(), 8, hour*3)
	require.NoError(t, err)

	// Sparse traffic means this replica may not see the identifier again for
	// minutes. It is still in the same hour and same daily window.
	clk.Tick(10 * time.Minute)
	for i := range readReqs {
		readReqs[i].Time = clk.Now()
	}

	resp, err = staleReplica.RatelimitMany(ctx, readReqs)
	require.NoError(t, err)
	require.Len(t, resp, 2)
	require.True(t, resp[0].Success)
	require.Equal(t, int64(2), resp[0].Remaining, "daily cost=0 read must reflect the shared origin after the stale replica has been idle")
	require.True(t, resp[1].Success)
	require.Equal(t, int64(12), resp[1].Remaining, "hourly cost=0 read must reflect the shared origin after the stale replica has been idle")

	chargeReqs := []RatelimitRequest{
		makeReq("tokens.day", 10, day, 3),
		makeReq("tokens.hour", 20, hour, 3),
	}

	resp, err = staleReplica.RatelimitMany(ctx, chargeReqs)
	require.NoError(t, err)
	require.Len(t, resp, 2)
	require.False(t, resp[0].Success, "daily hard cap must deny based on the shared origin")
	require.Equal(t, int64(0), resp[0].Remaining)
	require.True(t, resp[1].Success, "hourly limit passes individually, but the batch must not commit because daily failed")
	require.Equal(t, int64(12), resp[1].Remaining, "hourly remaining is unchanged because RatelimitMany rolls back the whole batch")
}
