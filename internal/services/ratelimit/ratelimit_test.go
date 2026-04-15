package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
)

// TestRatelimit_SlidingWindowDecision locks in the math of the sliding window
// algorithm. Each case pre-seeds the current and previous window counters and
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
			svc, err := New(Config{Clock: clk, Counter: counter.NewMemory()})
			require.NoError(t, err)
			t.Cleanup(func() { _ = svc.Close() })

			duration := time.Minute
			durationMs := duration.Milliseconds()

			// Derive a request time that produces the desired elapsed fraction.
			curSeq := calculateSequence(clk.Now(), duration)
			windowStartMs := curSeq * durationMs
			elapsedMs := int64(tt.elapsed * float64(durationMs))
			reqTime := time.UnixMilli(windowStartMs + elapsedMs)

			// Seed both windows so prepareCheck takes the local fast path.
			curKey := counterKey{name: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq}
			prevKey := counterKey{name: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq - 1}
			cur := svc.loadCounter(curKey)
			cur.val.Store(tt.curCount)
			prev := svc.loadCounter(prevKey)
			prev.val.Store(tt.prevCount)

			resp, err := svc.Ratelimit(context.Background(), RatelimitRequest{
				Name:       "ns",
				Identifier: "id",
				Limit:      tt.limit,
				Duration:   duration,
				Cost:       tt.cost,
				Time:       reqTime,
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantPass, resp.Success, "Success")
			require.Equal(t, tt.wantRem, resp.Remaining, "Remaining")
		})
	}
}

// TestRatelimit_DenialSetsStrictUntil asserts a denied request sets the
// strict-mode deadline to req.Time + req.Duration for its (name, identifier,
// duration) tuple.
func TestRatelimit_DenialSetsStrictUntil(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	svc, err := New(Config{Clock: clk, Counter: counter.NewMemory()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	duration := time.Minute
	req := RatelimitRequest{
		Name:       "ns",
		Identifier: "id",
		Limit:      1,
		Duration:   duration,
		Cost:       1,
		Time:       clk.Now(),
	}

	resp, err := svc.Ratelimit(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Success, "first request at limit=1 should pass")

	resp, err = svc.Ratelimit(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.Success, "second request should be denied")

	sk := strictKey{name: req.Name, identifier: req.Identifier, durationMs: duration.Milliseconds()}
	got := svc.loadStrictUntil(sk)
	want := req.Time.Add(duration).UnixMilli()
	require.Equal(t, want, got, "strictUntil should equal req.Time + req.Duration")
}

// TestRatelimit_StrictModeForcesOriginFetch asserts that once strict mode is
// active, prepareCheck fetches from origin even when the local windows are
// warm. We verify the behavior by observing local state converge to a seeded
// origin value — the only way that can happen is if origin was consulted.
func TestRatelimit_StrictModeForcesOriginFetch(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	origin := counter.NewMemory()
	svc, err := New(Config{Clock: clk, Counter: origin})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	duration := time.Minute
	durationMs := duration.Milliseconds()
	reqTime := clk.Now()
	curSeq := calculateSequence(reqTime, duration)

	curKey := counterKey{name: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq}
	prevKey := counterKey{name: "ns", identifier: "id", durationMs: durationMs, sequence: curSeq - 1}

	// Warm local windows so the first denial does not trigger a cold-window fetch.
	cur := svc.loadCounter(curKey)
	prev := svc.loadCounter(prevKey)

	req := RatelimitRequest{
		Name:       "ns",
		Identifier: "id",
		Limit:      1,
		Duration:   duration,
		Cost:       1,
		Time:       reqTime,
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
	require.GreaterOrEqual(t, prev.val.Load(), originValue, "previous window should have picked up origin value")
}

// TestRatelimitMany_RollsBackOnPartialFailure asserts that when any entry in
// a batch exceeds its limit, none of the batch's counters are incremented —
// matching the all-or-nothing contract expected by callers.
func TestRatelimitMany_RollsBackOnPartialFailure(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	svc, err := New(Config{Clock: clk, Counter: counter.NewMemory()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	duration := time.Minute
	durationMs := duration.Milliseconds()
	reqTime := clk.Now()
	curSeq := calculateSequence(reqTime, duration)

	// Seed warm windows so prepareCheck is local-only.
	aCur := counterKey{name: "A", identifier: "user", durationMs: durationMs, sequence: curSeq}
	aPrev := counterKey{name: "A", identifier: "user", durationMs: durationMs, sequence: curSeq - 1}
	bCur := counterKey{name: "B", identifier: "user", durationMs: durationMs, sequence: curSeq}
	bPrev := counterKey{name: "B", identifier: "user", durationMs: durationMs, sequence: curSeq - 1}
	a := svc.loadCounter(aCur)
	_ = svc.loadCounter(aPrev)
	b := svc.loadCounter(bCur)
	_ = svc.loadCounter(bPrev)

	// A is far below its limit. B is at 9 of 10 — a cost of 5 will exceed it.
	b.val.Store(9)

	reqs := []RatelimitRequest{
		{Name: "A", Identifier: "user", Limit: 10, Duration: duration, Cost: 1, Time: reqTime},
		{Name: "B", Identifier: "user", Limit: 10, Duration: duration, Cost: 5, Time: reqTime},
	}

	resp, err := svc.RatelimitMany(context.Background(), reqs)
	require.NoError(t, err)
	require.Len(t, resp, 2)

	require.True(t, resp[0].Success, "A individually passed")
	require.False(t, resp[1].Success, "B individually exceeded")

	require.Equal(t, int64(0), a.val.Load(), "A counter must be unchanged after batch failure")
	require.Equal(t, int64(9), b.val.Load(), "B counter must be rolled back after batch failure")
}
