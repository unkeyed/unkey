package ratelimit

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	gcdb "github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/uid"
)

var testRegistry = prometheus.NewRegistry()

func init() {
	lazy.SetRegistry(testRegistry)
}

// decisions is how many decisions of op the system has timed.
func decisions(t *testing.T, op, system string) uint64 {
	t.Helper()
	families, err := testRegistry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "unkey_ratelimit_decision_duration_seconds" {
			continue
		}
		for _, sample := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range sample.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["op"] == op && labels["system"] == system {
				return sample.GetHistogram().GetSampleCount()
			}
		}
	}
	return 0
}

// newCountedRegion builds a region that counts requests with UseGlobalCounter
// through the global counter.
func (e *integrationTestEnv) newCountedRegion(clk clock.Clock, regionTag string) *service {
	e.t.Helper()
	svc, err := New(Config{Clock: clk, Counter: counter.NewMemory(), DB: e.db, Region: regionTag, GlobalCounter: true})
	require.NoError(e.t, err)
	e.t.Cleanup(func() { _ = svc.Close() })
	return svc
}

// persistedCount is the sum of every writer's persisted contribution to a cell.
func (e *integrationTestEnv) persistedCount(req RatelimitRequest) int64 {
	e.t.Helper()
	total, err := gcdb.Query.Read(e.t.Context(), e.db.RO(), gcdb.ReadParams{
		Name:            globalCounterName(req, windowOf(req).sequence),
		CurrentWriterID: "", PreviousWriterID: "",
	})
	require.NoError(e.t, err)
	return total
}

// hourRequest returns a global-counter request in the hour window that starts at
// 00:00 on a fixed day, so ticking the clock a few seconds never crosses a window.
func hourRequest(workspaceID, identifier string, at time.Time, cost, limit int64) RatelimitRequest {
	return RatelimitRequest{
		WorkspaceID: workspaceID, Namespace: "ns", Identifier: identifier,
		Limit: limit, Duration: time.Hour, Cost: cost, Time: at, UseGlobalCounter: true,
	}
}

// regionalRequest is hourRequest for a namespace that does not use the global counter.
func regionalRequest(workspaceID, identifier string, at time.Time, cost, limit int64) RatelimitRequest {
	req := hourRequest(workspaceID, identifier, at, cost, limit)
	req.UseGlobalCounter = false
	return req
}

func TestGlobalCounter_SharesCountsAcrossRegionsAndOnlyForEnabledNamespaces(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC))
	workspaceID := uid.New(uid.WorkspacePrefix)
	regionA := env.newCountedRegion(clk, "region-a")
	regionB := env.newCountedRegion(clk, "region-b")
	ctx := t.Context()
	identifier, regional := uid.New(uid.TestPrefix), uid.New(uid.TestPrefix)

	for range 6 {
		for _, req := range []RatelimitRequest{
			hourRequest(workspaceID, identifier, clk.Now(), 1, 10),
			regionalRequest(workspaceID, regional, clk.Now(), 1, 10),
		} {
			resp, err := regionA.Ratelimit(ctx, req)
			require.NoError(t, err)
			require.True(t, resp.Success)
		}
	}

	// A cost-0 read in region B sees region A's six, once they are persisted and
	// the peer cache refreshes. A cost of 0 is how callers check a limit without
	// consuming it.
	require.Eventually(t, func() bool {
		clk.Tick(5 * time.Second)
		resp, err := regionB.Ratelimit(ctx, hourRequest(workspaceID, identifier, clk.Now(), 0, 10))
		return err == nil && resp.Success && resp.Current == 6
	}, 10*time.Second, 50*time.Millisecond)

	denied, err := regionB.Ratelimit(ctx, hourRequest(workspaceID, identifier, clk.Now(), 5, 10))
	require.NoError(t, err)
	require.False(t, denied.Success, "6 from region A plus 5 exceeds the limit of 10")
	require.Equal(t, int64(11), denied.Current)

	allowed, err := regionB.Ratelimit(ctx, hourRequest(workspaceID, identifier, clk.Now(), 4, 10))
	require.NoError(t, err)
	require.True(t, allowed.Success, "6 from region A plus 4 reaches the limit exactly")
	require.Equal(t, int64(0), allowed.Remaining)

	require.Eventually(t, func() bool {
		return env.persistedCount(hourRequest(workspaceID, identifier, clk.Now(), 0, 10)) == 10
	}, 10*time.Second, 50*time.Millisecond, "region A's 6 and region B's 4 are stored once persisted")
	require.Zero(t, env.persistedCount(hourRequest(workspaceID, regional, clk.Now(), 0, 10)), "a namespace without the global counter must not write to it")
}

func TestGlobalCounter_PreviousWindowCountsForItsRemainingShare(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 30, 0, 0, time.UTC))
	workspaceID := uid.New(uid.WorkspacePrefix)
	region := env.newCountedRegion(clk, "region-a")
	identifier := uid.New(uid.TestPrefix)
	ctx := t.Context()

	resp, err := region.Ratelimit(ctx, hourRequest(workspaceID, identifier, clk.Now(), 10, 10))
	require.NoError(t, err)
	require.True(t, resp.Success)

	// Halfway into the next hour, half of the previous window's 10 still counts.
	clk.Set(time.Date(2026, 1, 1, 1, 30, 0, 0, time.UTC))
	denied, err := region.Ratelimit(ctx, hourRequest(workspaceID, identifier, clk.Now(), 6, 10))
	require.NoError(t, err)
	require.False(t, denied.Success)
	require.Equal(t, int64(11), denied.Current, "5 from the previous window plus the cost of 6")

	allowed, err := region.Ratelimit(ctx, hourRequest(workspaceID, identifier, clk.Now(), 5, 10))
	require.NoError(t, err)
	require.True(t, allowed.Success)
	require.Equal(t, int64(10), allowed.Current)
	require.True(t, time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC).Equal(allowed.Reset), "the window ends at 02:00, got %s", allowed.Reset)
}

func TestGlobalCounter_NamesTooLongForMySQLAreHashed(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC))
	workspaceID := uid.New(uid.WorkspacePrefix)
	region := env.newCountedRegion(clk, "region-a")
	identifier := strings.Repeat("x", 600)

	for range 2 {
		resp, err := region.Ratelimit(t.Context(), hourRequest(workspaceID, identifier, clk.Now(), 1, 2))
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	resp, err := region.Ratelimit(t.Context(), hourRequest(workspaceID, identifier, clk.Now(), 1, 2))
	require.NoError(t, err)
	require.False(t, resp.Success, "the limit must hold on the global counter")
	require.Eventually(t, func() bool {
		return env.persistedCount(hourRequest(workspaceID, identifier, clk.Now(), 0, 2)) == 2
	}, 10*time.Second, 50*time.Millisecond, "the two passed requests are stored under the hashed name")
}

func TestGlobalCounter_ManyAddsEveryCostOrNone(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC))
	workspaceID := uid.New(uid.WorkspacePrefix)
	region := env.newCountedRegion(clk, "region-a")
	ctx := t.Context()
	nearlyFull, roomy := uid.New(uid.TestPrefix), uid.New(uid.TestPrefix)

	resp, err := region.Ratelimit(ctx, hourRequest(workspaceID, nearlyFull, clk.Now(), 9, 10))
	require.NoError(t, err)
	require.True(t, resp.Success)

	// The first limit would pass alone, the second cannot, so nothing may change.
	responses, err := region.RatelimitMany(ctx, []RatelimitRequest{
		hourRequest(workspaceID, roomy, clk.Now(), 1, 10),
		hourRequest(workspaceID, nearlyFull, clk.Now(), 5, 10),
	})
	require.NoError(t, err)
	require.Len(t, responses, 2)
	require.True(t, responses[0].Success, "the roomy limit passes on its own")
	require.Equal(t, int64(10), responses[0].Remaining, "its cost was not consumed, so it is given back")
	require.False(t, responses[1].Success)
	require.Equal(t, int64(14), responses[1].Current)

	full, err := region.Ratelimit(ctx, hourRequest(workspaceID, roomy, clk.Now(), 10, 10))
	require.NoError(t, err)
	require.True(t, full.Success, "the roomy limit's cost of 1 must not have been consumed by the failed call")

	responses, err = region.RatelimitMany(ctx, []RatelimitRequest{
		hourRequest(workspaceID, nearlyFull, clk.Now(), 1, 10),
		hourRequest(workspaceID, uid.New(uid.TestPrefix), clk.Now(), 3, 10),
	})
	require.NoError(t, err)
	require.True(t, responses[0].Success)
	require.True(t, responses[1].Success)
	require.Equal(t, int64(0), responses[0].Remaining)
	require.Equal(t, int64(7), responses[1].Remaining)
}

func TestGlobalCounter_ManyWithTheSameCellTwiceFails(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC))
	workspaceID := uid.New(uid.WorkspacePrefix)
	region := env.newCountedRegion(clk, "region-a")
	identifier := uid.New(uid.TestPrefix)

	_, err := region.RatelimitMany(t.Context(), []RatelimitRequest{
		hourRequest(workspaceID, identifier, clk.Now(), 3, 5),
		hourRequest(workspaceID, identifier, clk.Now(), 4, 5),
	})
	require.Error(t, err)
	require.Zero(t, env.persistedCount(hourRequest(workspaceID, identifier, clk.Now(), 0, 5)), "a failed call must add nothing")
}

func TestGlobalCounter_ManyWithAPreviousWindowThatAnotherCheckAddsToFails(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 30, 0, 0, time.UTC))
	workspaceID := uid.New(uid.WorkspacePrefix)
	region := env.newCountedRegion(clk, "region-a")
	identifier := uid.New(uid.TestPrefix)

	// The second check's previous hour is the first check's current hour.
	first := hourRequest(workspaceID, identifier, clk.Now(), 6, 10)
	second := hourRequest(workspaceID, identifier, clk.Now().Add(30*time.Minute), 6, 10)
	_, err := region.RatelimitMany(t.Context(), []RatelimitRequest{first, second})
	require.Error(t, err)
	require.Zero(t, env.persistedCount(first), "a failed call must add nothing")
}

// TestGlobalCounter_DecisionDurationNamesTheSystemThatDecided is not parallel, so
// no other test adds decisions while it counts them.
func TestGlobalCounter_DecisionDurationNamesTheSystemThatDecided(t *testing.T) {
	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC))
	workspaceID := uid.New(uid.WorkspacePrefix)
	region := env.newCountedRegion(clk, "region-a")
	ctx := t.Context()

	global := hourRequest(workspaceID, uid.New(uid.TestPrefix), clk.Now(), 1, 10)
	regional := regionalRequest(workspaceID, uid.New(uid.TestPrefix), clk.Now(), 1, 10)
	type counts struct{ limitGlobal, limitRegional, manyGlobal, manyRegional uint64 }
	read := func() counts {
		return counts{
			limitGlobal:   decisions(t, "limit", "global_counter"),
			limitRegional: decisions(t, "limit", "regional"),
			manyGlobal:    decisions(t, "multi_limit", "global_counter"),
			manyRegional:  decisions(t, "multi_limit", "regional"),
		}
	}
	before := read()

	for range 2 {
		_, err := region.Ratelimit(ctx, global)
		require.NoError(t, err)
	}
	_, err := region.Ratelimit(ctx, regional)
	require.NoError(t, err)
	_, err = region.RatelimitMany(ctx, []RatelimitRequest{global, hourRequest(workspaceID, uid.New(uid.TestPrefix), clk.Now(), 1, 10)})
	require.NoError(t, err)
	for range 3 {
		_, err = region.RatelimitMany(ctx, []RatelimitRequest{global, regional})
		require.NoError(t, err)
	}

	after := read()
	require.Equal(t, counts{
		limitGlobal:   before.limitGlobal + 2,
		limitRegional: before.limitRegional + 1,
		manyGlobal:    before.manyGlobal + 1,
		manyRegional:  before.manyRegional + 3,
	}, after, "a call that mixes systems is decided by the regional counters")
}
