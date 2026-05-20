package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
)

// rawCountRow is the per-region row shape stored in MySQL. Tests use
// it for inspection because the production GlobalCountersImported query
// returns pre-aggregated sums, which is correct for the pull loop but
// hides the per-region detail needed to assert which region wrote what.
type rawCountRow struct {
	WorkspaceID string
	Namespace   string
	Identifier  string
	DurationMs  uint64
	Sequence    int64
	Region      string
	Count       uint64
	ExpiresAt   uint64
	UpdatedAt   uint64
}

// listAllRows returns every row in ratelimit_global_counters
// regardless of region or expiry. Tests bypass the production query
// (which aggregates and filters out the caller's own region) so they can
// assert which region wrote what.
func (e *integrationTestEnv) listAllRows() []rawCountRow {
	e.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rows, err := e.db.RO().QueryContext(ctx,
		"SELECT workspace_id, namespace, identifier, duration_ms, sequence, region, count, expires_at, updated_at FROM ratelimit_global_counters")
	require.NoError(e.t, err)
	defer func() { _ = rows.Close() }()
	var out []rawCountRow
	for rows.Next() {
		var r rawCountRow
		require.NoError(e.t, rows.Scan(
			&r.WorkspaceID,
			&r.Namespace,
			&r.Identifier,
			&r.DurationMs,
			&r.Sequence,
			&r.Region,
			&r.Count,
			&r.ExpiresAt,
			&r.UpdatedAt,
		))
		out = append(out, r)
	}
	require.NoError(e.t, rows.Err())
	return out
}

// findRow returns the row for (key, region) if present, ok=false
// otherwise. Used by both positive and negative assertions.
func (e *integrationTestEnv) findRow(workspaceID, namespace, identifier, region string, durationMs int64) (rawCountRow, bool) {
	e.t.Helper()
	for _, r := range e.listAllRows() {
		if r.WorkspaceID == workspaceID && r.Namespace == namespace &&
			r.Identifier == identifier && int64(r.DurationMs) == durationMs &&
			r.Region == region {
			return r, true
		}
	}
	return rawCountRow{}, false
}

// waitForRow blocks until a row from `region` for the matching
// window cell is visible in MySQL. Manual runGlobalPushOnce calls
// in tests are synchronous, so the row is normally visible immediately;
// the polling loop covers the case where the periodic push goroutine
// (running on its own jittered cadence) is the writer. Polling, never
// goroutine-ticker timing.
func (e *integrationTestEnv) waitForRow(workspaceID, namespace, identifier, region string, durationMs int64) rawCountRow {
	e.t.Helper()
	var found rawCountRow
	require.Eventually(e.t, func() bool {
		row, ok := e.findRow(workspaceID, namespace, identifier, region, durationMs)
		if !ok {
			return false
		}
		found = row
		return true
	}, 3*time.Second, 50*time.Millisecond, "expected global-counters row from region %q", region)
	return found
}

// hasRow reports whether any row exists for the (key, region)
// tuple. Used by negative tests that assert a flush did NOT happen.
func (e *integrationTestEnv) hasRow(workspaceID, namespace, identifier, region string, durationMs int64) bool {
	e.t.Helper()
	_, ok := e.findRow(workspaceID, namespace, identifier, region, durationMs)
	return ok
}

// TestGlobal_PropagatesCountAcrossRegions is the headline scenario:
// region A consumes part of the limit and flushes its count; region B
// reads the row and folds it into its sliding-window math. A request to
// B that would have passed against B's local-only count is now denied
// because the cross-region picture says the user is already over. A
// never denies in this scenario; the value of sharing is that B sees A's
// count even though no individual region has crossed the limit.
func TestGlobal_PropagatesCountAcrossRegions(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionA := env.newRegionAs(clk, "region-a")
	regionB := env.newRegionAs(clk, "region-b")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-shared"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		}
	}

	// Region A consumes 6 of 10 — past the 0.5 utilization floor so flush
	// is eligible. None of these denies; A is well within its limit on its
	// own, but its consumption is now relevant to other regions.
	for range 6 {
		resp, err := regionA.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	// Trigger flush deterministically rather than waiting for the 10s
	// jittered ticker. The bulk upsert inside runGlobalPushOnce is
	// synchronous, so by the time it returns the row is in MySQL.
	regionA.runGlobalPushOnce()
	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(6), row.Count, "region A's row should reflect its 6 consumed tokens")

	// Region B has seen no traffic. Without count sharing, B would happily
	// allow up to its own limit. Pull from MySQL: B.globalCount should now
	// reflect A's contribution.
	regionB.runGlobalPullOnce()

	// One more cost-1 request to B: effective = 0 (B's val) + 6 (B's globalCount
	// from A) + 0 (prev) + 1 (cost) = 7, still under 10, so it passes.
	resp, err := regionB.Ratelimit(ctx, makeReq())
	require.NoError(t, err)
	require.True(t, resp.Success, "B should allow first request when combined effective is 7/10")

	// After that one passing request, B's val=1 and globalCount=6. A request
	// of cost 4 would push effective to 1+4+6=11 > 10: deny. This is the
	// behavior the cross-region sharing buys us.
	denyReq := makeReq()
	denyReq.Cost = 4
	resp, err = regionB.Ratelimit(ctx, denyReq)
	require.NoError(t, err)
	require.False(t, resp.Success, "B must deny when combined effective exceeds limit; without sharing this would have passed")
}

func TestGlobal_RealWorldTwoRegionsWithTwoNodesEach(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionAOrigin := counter.NewMemory()
	regionBOrigin := counter.NewMemory()

	regionA1 := env.newRegionWithCounter(clk, "region-a", regionAOrigin)
	regionA2 := env.newRegionWithCounter(clk, "region-a", regionAOrigin)
	regionB1 := env.newRegionWithCounter(clk, "region-b", regionBOrigin)
	regionB2 := env.newRegionWithCounter(clk, "region-b", regionBOrigin)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-real-world"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			Cost:        1,
			Time:        clk.Now(),
		}
	}
	curKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  duration.Milliseconds(),
		sequence:    calculateSequence(clk.Now(), duration),
	}

	// Region A receives traffic on two nodes backed by the same regional origin
	// counter. This models two API instances sharing one regional Redis while
	// other regions have independent Redis state.
	for range 3 {
		resp, err := regionA1.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	for range 3 {
		resp, err := regionA2.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	waitForLocalCount := func(regions []*service, expected int64) *service {
		t.Helper()
		var selected *service
		require.Eventually(t, func() bool {
			for _, region := range regions {
				entry, ok := region.counters.Load(curKey)
				if ok && entry.(*counterEntry).val.Load() >= expected {
					selected = region
					return true
				}
			}
			return false
		}, 3*time.Second, 50*time.Millisecond, "expected regional origin replay to converge local count to %d", expected)
		if selected == nil {
			t.Fatalf("expected a selected region after convergence")
		}
		return selected
	}

	// Regional replay should converge at least one A node to the region's total
	// count, and the global counter row should publish that converged value.
	waitForLocalCount([]*service{regionA1, regionA2}, 6)
	regionA1.runGlobalPushOnce()
	regionA2.runGlobalPushOnce()
	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(6), row.Count)

	// Both B nodes import region A's contribution. They still use their own
	// regional origin for B-local traffic; MySQL only carries cross-region state.
	regionB1.runGlobalPullOnce()
	regionB2.runGlobalPullOnce()
	for _, region := range []*service{regionB1, regionB2} {
		entryValue, ok := region.counters.Load(curKey)
		require.True(t, ok)
		require.Equal(t, int64(6), entryValue.(*counterEntry).globalCount.Load())
	}

	for range 2 {
		resp, err := regionB1.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	for range 2 {
		resp, err := regionB2.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	// Once B's regional origin converges its two nodes' local traffic to 4, any
	// further cost-1 request in B must deny: local 4 + imported 6 + cost 1 = 11.
	denyRegion := waitForLocalCount([]*service{regionB1, regionB2}, 4)
	resp, err := denyRegion.Ratelimit(ctx, makeReq())
	require.NoError(t, err)
	require.False(t, resp.Success,
		"region B must deny after combining its own regional count with region A's imported count")
}

// TestGlobal_BelowUtilizationFloorDoesNotPush asserts the write-side
// utilization filter that gates emission. An entry whose val/limit is below
// the floor cannot meaningfully push another region over its limit, so
// flushing it would be wasted MySQL load. The cursed scenario it prevents
// is "cost > limit on a fresh window" — the user has consumed nothing in
// this window but would otherwise be propagated to all other regions.
func TestGlobal_BelowUtilizationFloorDoesNotPush(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-low-util"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Consume 4 of 10 — strictly below the default 0.5 floor.
	for range 4 {
		resp, err := region.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	region.runGlobalPushOnce()

	// The flush filter must have skipped this entry. require.Never polls
	// to guard against a periodic flush from the background goroutine
	// firing concurrently.
	require.Never(t, func() bool {
		return env.hasRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	}, 2*time.Second, 100*time.Millisecond,
		"sub-floor utilization must not write a global-counters row")
}

// TestGlobal_PushUsesConvergedLocalCount asserts that push eligibility is
// based on the live local counter, not a request-path latch. Two instances in
// the same region can each stay below the utilization floor locally while
// Redis convergence raises one instance's val above the floor; that converged
// value must still flush cross-region.
func TestGlobal_PushUsesConvergedLocalCount(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	sharedCounter := counter.NewMemory()
	regionA1, err := New(Config{
		Clock:   clk,
		Counter: sharedCounter,
		DB:      env.db,
		Region:  "region-a",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = regionA1.Close() })
	regionA2, err := New(Config{
		Clock:   clk,
		Counter: sharedCounter,
		DB:      env.db,
		Region:  "region-a",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = regionA2.Close() })

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-converged"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			Cost:        1,
			Time:        clk.Now(),
		}
	}

	// Each instance accepts 4/10 locally, below the 0.5 utilization floor.
	// Their shared Redis origin converges the region total to 8/10.
	for range 4 {
		resp, reqErr := regionA1.Ratelimit(ctx, makeReq())
		require.NoError(t, reqErr)
		require.True(t, resp.Success)
	}
	for range 4 {
		resp, reqErr := regionA2.Ratelimit(ctx, makeReq())
		require.NoError(t, reqErr)
		require.True(t, resp.Success)
	}

	curKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  duration.Milliseconds(),
		sequence:    calculateSequence(clk.Now(), duration),
	}
	require.Eventually(t, func() bool {
		for _, region := range []*service{regionA1, regionA2} {
			entry, ok := region.counters.Load(curKey)
			if ok && entry.(*counterEntry).val.Load() >= 8 {
				return true
			}
		}
		return false
	}, 3*time.Second, 50*time.Millisecond, "expected Redis replay to converge local count above the push floor")

	regionA1.runGlobalPushOnce()
	regionA2.runGlobalPushOnce()

	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(8), row.Count, "push must use the converged local count")
}

// TestGlobal_PushIgnoresSpeculativeBatchIncrements covers the small window in
// [Service.RatelimitMany] where a batch has added its optimistic increments to
// the local counters but has not yet decided whether the full batch commits.
// Cross-region push must subtract those speculative increments so a batch that
// later rolls back cannot publish temporary state to MySQL.
//
// The test drives the speculative field directly rather than racing a live
// RatelimitMany call against the background pusher. The invariant under test is
// the push-side read of val minus speculative, not scheduler timing.
func TestGlobal_PushIgnoresSpeculativeBatchIncrements(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-speculative"
		limit       = int64(10)
	)
	duration := time.Minute

	resp, err := region.Ratelimit(context.Background(), RatelimitRequest{
		WorkspaceID: workspaceID,
		Namespace:   namespace,
		Identifier:  identifier,
		Limit:       limit,
		Duration:    duration,
		Cost:        6,
		Time:        clk.Now(),
	})
	require.NoError(t, err)
	require.True(t, resp.Success)

	curKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  duration.Milliseconds(),
		sequence:    calculateSequence(clk.Now(), duration),
	}
	entryValue, ok := region.counters.Load(curKey)
	require.True(t, ok)
	entry := entryValue.(*counterEntry)

	entry.speculative.Add(6)
	region.runGlobalPushOnce()
	require.False(t, env.hasRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds()),
		"in-flight RatelimitMany increments must not be pushed")

	entry.speculative.Add(-6)
	region.runGlobalPushOnce()
	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(6), row.Count)
}

// TestGlobal_AtFloorPushes is the inverse of the previous test: at
// or above the floor, the entry must flush. Tests the boundary specifically
// to guard against an off-by-one in the comparison.
func TestGlobal_AtFloorPushes(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-at-floor"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Consume exactly 5 of 10 — at the 0.5 floor.
	for range 5 {
		resp, err := region.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	region.runGlobalPushOnce()

	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(5), row.Count, "row count must reflect the at-floor utilization")
}

// TestGlobal_SyncExcludesOwnRegion asserts that a region's sync skips
// rows it wrote itself. Without the region != self predicate, a region would
// fold its own contribution back into its globalCount field, double-counting on
// every request and over-blocking by a factor proportional to sync frequency.
func TestGlobal_SyncExcludesOwnRegion(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-self-only"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Push past floor so a row gets written.
	for range 6 {
		resp, err := region.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	region.runGlobalPushOnce()
	env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())

	// Sync from this same region — should NOT pull its own row, so globalCount
	// must stay at zero.
	region.runGlobalPullOnce()

	curKey := counterKey{
		workspaceID: workspaceID, namespace: namespace, identifier: identifier,
		durationMs: duration.Milliseconds(),
		sequence:   calculateSequence(clk.Now(), duration),
	}
	entry, ok := region.counters.Load(curKey)
	require.True(t, ok)
	require.Equal(t, int64(0), entry.(*counterEntry).globalCount.Load(),
		"sync must not import the region's own contribution")
}

// TestGlobal_SumsAcrossMultipleRegions asserts that a third region
// folding in contributions from two others sums them rather than taking
// max or replacing. This is the load-bearing G-Counter property: each
// region's row holds its own contribution; aggregation across rows is sum.
func TestGlobal_SumsAcrossMultipleRegions(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionA := env.newRegionAs(clk, "region-a")
	regionB := env.newRegionAs(clk, "region-b")
	regionC := env.newRegionAs(clk, "region-c")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-multi"
		limit       = int64(20)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		}
	}

	// A consumes 12 (above floor=10), B consumes 11 (above floor=10).
	// Total true global at this point is 23.
	for range 12 {
		_, err := regionA.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
	}
	for range 11 {
		_, err := regionB.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
	}

	regionA.runGlobalPushOnce()
	regionB.runGlobalPushOnce()
	env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	env.waitForRow(workspaceID, namespace, identifier, "region-b", duration.Milliseconds())

	// Region C has seen nothing locally. After sync, its globalCount must
	// equal the sum of A's 12 and B's 11. Max would give 12 (under-count),
	// replace would give whichever arrived last.
	regionC.runGlobalPullOnce()

	curKey := counterKey{
		workspaceID: workspaceID, namespace: namespace, identifier: identifier,
		durationMs: duration.Milliseconds(),
		sequence:   calculateSequence(clk.Now(), duration),
	}
	entry, ok := regionC.counters.Load(curKey)
	require.True(t, ok, "sync must create the entry on C even with no local traffic")
	require.Equal(t, int64(23), entry.(*counterEntry).globalCount.Load(),
		"region C must see SUM(12+11)=23, not max or last-write-wins")

	// Concretely: any cost-1 request on C must now deny because effective
	// is already 23 > 20.
	resp, err := regionC.Ratelimit(ctx, makeReq())
	require.NoError(t, err)
	require.False(t, resp.Success,
		"C must deny when globalCount sum already exceeds the limit")
}

// TestGlobal_ChangeFilterAvoidsRedundantWrites asserts that a flush
// run twice with no intervening traffic does not produce a second write.
// The change filter (skip if val <= lastPushed) is the dominant cost
// reduction in steady state where most active windows are quiet between
// flushes.
func TestGlobal_ChangeFilterAvoidsRedundantWrites(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-quiet"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Push to 6/10 so the first flush emits.
	for range 6 {
		resp, err := region.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	region.runGlobalPushOnce()
	first := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(6), first.Count)

	// Advance the test clock so a second flush, if it emits, writes a
	// distinguishably newer updated_at. Without this advance, the second
	// upsert (if it happens) would carry the same updated_at as the
	// first, and the assertion below would pass trivially.
	clk.Tick(time.Second)

	// Trigger another flush with no intervening traffic. The filter must
	// recognize val unchanged and skip the upsert. The bulk upsert is
	// synchronous, so by the time runGlobalPushOnce returns the
	// row is either updated or it isn't — no waiting needed.
	region.runGlobalPushOnce()

	second, ok := env.findRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.True(t, ok)
	require.Equal(t, first.UpdatedAt, second.UpdatedAt,
		"change filter must skip the second flush; a re-emitted upsert would advance updated_at")
}

// TestGlobal_DoesNotPropagateColdOversizedRequest asserts that a single
// oversized request denied locally does not propagate to other regions.
// A user has consumed zero tokens this window; the deny path doesn't
// bump val, so val stays 0 and the utilization filter excludes the entry
// from the flush. Other regions see no row and do not over-block this
// user on their next sync.
func TestGlobal_DoesNotPropagateColdOversizedRequest(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-oversized"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	resp, err := region.Ratelimit(ctx, RatelimitRequest{
		WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
		Limit: limit, Duration: duration, Cost: limit + 1, Time: clk.Now(),
	})
	require.NoError(t, err)
	require.False(t, resp.Success, "cost > limit must deny locally")

	region.runGlobalPushOnce()

	// The cold-window denial increments nothing (deny path doesn't bump
	// val), so val stays 0 and the utilization filter skips the flush.
	require.Never(t, func() bool {
		return env.hasRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	}, 2*time.Second, 100*time.Millisecond,
		"cold oversized denial must not write a global-counters row")
}

// TestGlobal_EntriesCreatedOnSync asserts that the pull goroutine
// materializes a local counterEntry when it sees a row for a key the
// region has not seen any traffic for. The new entry's globalCount
// carries the remote sum; subsequent local requests on this identifier
// deny correctly even though the region's val starts at zero.
func TestGlobal_EntriesCreatedOnSync(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionA := env.newRegionAs(clk, "region-a")
	regionB := env.newRegionAs(clk, "region-b")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-cold-import"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	for range 6 {
		_, err := regionA.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
	}
	regionA.runGlobalPushOnce()
	env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())

	// Region B has no local entry for this key. Sync must materialize one
	// with globalCount=6.
	regionB.runGlobalPullOnce()

	curKey := counterKey{
		workspaceID: workspaceID, namespace: namespace, identifier: identifier,
		durationMs: duration.Milliseconds(),
		sequence:   calculateSequence(clk.Now(), duration),
	}
	entry, ok := regionB.counters.Load(curKey)
	require.True(t, ok, "sync must create the entry on B even with no local traffic")
	require.Equal(t, int64(6), entry.(*counterEntry).globalCount.Load())
}
