package gcra_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/gcra"
	"github.com/unkeyed/unkey/pkg/gcra/db"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

var testRegistry = prometheus.NewRegistry()

func init() {
	lazy.SetRegistry(testRegistry)
}

func newDatabase(t *testing.T) mysql.MySQL {
	t.Helper()
	config := containers.MySQL(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN, ReadOnlyDSN: ""})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func newLimiter(t *testing.T, database mysql.MySQL, clk clock.Clock) *gcra.Limiter {
	t.Helper()
	limiter, err := gcra.New(gcra.Config{DB: database, Clock: clk})
	require.NoError(t, err)
	t.Cleanup(limiter.Close)
	return limiter
}

// totals reads the combined state of every row of name from the primary.
func totals(t *testing.T, database mysql.MySQL, name string) gcra.State {
	t.Helper()
	row, err := db.Query.ReadPeers(context.Background(), database.RW(), db.ReadPeersParams{Name: name, WriterID: ""})
	require.NoError(t, err)
	return gcra.State{SpentNs: row.SpentNs, FreshStartNs: row.FreshStartNs}
}

func memoryEntries(t *testing.T) float64 {
	t.Helper()
	families, err := testRegistry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == "unkey_gcra_memory_entries" {
			return family.GetMetric()[0].GetGauge().GetValue()
		}
	}
	return 0
}

func allowN(t *testing.T, limiter *gcra.Limiter, req gcra.Request, n int) int {
	t.Helper()
	passed := 0
	for range n {
		result, err := limiter.Allow(context.Background(), req)
		require.NoError(t, err)
		if result.Allowed {
			passed++
		}
	}
	return passed
}

func TestLimiter_SharesSpendingThroughMySQL(t *testing.T) {
	database := newDatabase(t)
	clk := clock.NewTestClock(time.Now().Round(0))
	a := newLimiter(t, database, clk)
	b := newLimiter(t, database, clk)
	rate := gcra.Rate{Limit: 10, Duration: 10 * time.Second}
	exhausted := gcra.Request{Name: uid.New(uid.TestPrefix), Rate: rate, Cost: 1}

	require.Equal(t, 10, allowN(t, a, exhausted, 11))
	require.Eventually(t, func() bool {
		return totals(t, database, exhausted.Name).SpentNs == 10*time.Second.Nanoseconds()
	}, 10*time.Second, 50*time.Millisecond)

	require.Equal(t, 0, allowN(t, b, exhausted, 1), "b reads a's spending on its first request")

	fresh := gcra.Request{Name: uid.New(uid.TestPrefix), Rate: rate, Cost: 3}
	results, err := b.AllowMany(context.Background(), []gcra.Request{fresh, exhausted})
	require.NoError(t, err)
	require.True(t, results[0].Allowed)
	require.Equal(t, int64(10), results[0].Remaining, "nothing was spent, so its own cost is reported back")
	require.False(t, results[1].Allowed)

	check, err := b.Allow(context.Background(), gcra.Request{Name: fresh.Name, Rate: rate, Cost: 0})
	require.NoError(t, err)
	require.Equal(t, int64(10), check.Remaining, "a call with a denied request spends nothing")

	_, err = b.AllowMany(context.Background(), []gcra.Request{fresh, fresh})
	require.Error(t, err)

	clk.Tick(time.Second)
	require.Equal(t, 1, allowN(t, b, exhausted, 2))
}

func TestLimiter_ForgottenEntryCountsAsPeer(t *testing.T) {
	database := newDatabase(t)
	clk := clock.NewTestClock(time.Now().Round(0))
	limiter := newLimiter(t, database, clk)
	req := gcra.Request{Name: uid.New(uid.TestPrefix), Rate: gcra.Rate{Limit: 1, Duration: time.Hour}, Cost: 1}

	require.Equal(t, 1, allowN(t, limiter, req, 2))
	require.Eventually(t, func() bool {
		return totals(t, database, req.Name).SpentNs == time.Hour.Nanoseconds()
	}, 10*time.Second, 50*time.Millisecond)

	before := memoryEntries(t)
	clk.Tick(31 * time.Minute)
	require.Eventually(t, func() bool {
		clk.Tick(time.Second)
		return memoryEntries(t) < before
	}, 10*time.Second, 50*time.Millisecond)

	require.Equal(t, 0, allowN(t, limiter, req, 1), "the forgotten writer's row still counts")
}

func TestLimiter_CleanupDeletesExpiredNames(t *testing.T) {
	database := newDatabase(t)
	clk := clock.NewTestClock(time.Now().Round(0))
	limiter := newLimiter(t, database, clk)
	req := gcra.Request{Name: uid.New(uid.TestPrefix), Rate: gcra.Rate{Limit: 10, Duration: 10 * time.Second}, Cost: 10}

	require.Equal(t, 1, allowN(t, limiter, req, 1))
	require.Eventually(t, func() bool {
		return totals(t, database, req.Name).SpentNs > 0
	}, 10*time.Second, 50*time.Millisecond)

	clk.Tick(11 * time.Minute)
	require.Eventually(t, func() bool {
		clk.Tick(time.Minute)
		return totals(t, database, req.Name) == gcra.State{SpentNs: 0, FreshStartNs: 0}
	}, 10*time.Second, 50*time.Millisecond)

	result, err := limiter.Allow(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.Allowed)
}

func counterValue(t *testing.T, name string) float64 {
	t.Helper()
	families, err := testRegistry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return family.GetMetric()[0].GetCounter().GetValue()
		}
	}
	return 0
}

// TestLimiter_CleanupStaysWithinItsLimits fills the table past every cleanup
// limit: one name with more rows than a transaction may lock, three names whose
// rows together exceed that limit, and more expired names than one page lists.
func TestLimiter_CleanupStaysWithinItsLimits(t *testing.T) {
	database := newDatabase(t)
	clk := clock.NewTestClock(time.Now().Round(0))
	ctx := context.Background()
	baseNs := clk.Now().Add(-time.Hour).UnixNano()

	rowsOf := func(name string, count int, expiresAtNs func(i int) int64) []db.PersistParams {
		rows := make([]db.PersistParams, count)
		for i := range rows {
			rows[i] = db.PersistParams{Name: name, WriterID: uid.New(uid.TestPrefix), SpentNs: 1, FreshStartNs: 1, ExpiresAtNs: expiresAtNs(i)}
		}
		return rows
	}
	var rows []db.PersistParams

	oversized := uid.New(uid.TestPrefix)
	rows = append(rows, rowsOf(oversized, 5_001, func(int) int64 { return baseNs - 1 })...)

	crowded := []string{uid.New(uid.TestPrefix), uid.New(uid.TestPrefix), uid.New(uid.TestPrefix)}
	for _, name := range crowded {
		rows = append(rows, rowsOf(name, 2_000, func(i int) int64 { return baseNs + int64(min(i, 1)) })...)
	}

	small := make([]string, 1_500)
	for i := range small {
		small[i] = uid.New(uid.TestPrefix)
		rows = append(rows, rowsOf(small[i], 1, func(int) int64 { return baseNs + 2 })...)
	}

	live := uid.New(uid.TestPrefix)
	rows = append(rows, rowsOf(live, 1, func(int) int64 { return clk.Now().Add(24 * time.Hour).UnixNano() })...)

	for chunk := range slices.Chunk(rows, 5_000) {
		require.NoError(t, db.BulkQuery.Persist(ctx, database.RW(), chunk))
	}

	skippedBefore := counterValue(t, "unkey_gcra_cleanup_names_skipped_total")
	newLimiter(t, database, clk)

	deleted := append(append([]string{}, crowded...), small...)
	require.Eventually(t, func() bool {
		clk.Tick(time.Minute)
		for _, name := range deleted {
			if totals(t, database, name).SpentNs != 0 {
				return false
			}
		}
		return true
	}, 30*time.Second, 100*time.Millisecond)

	require.Equal(t, int64(5_001), totals(t, database, oversized).SpentNs, "a name too large to lock is never deleted in part")
	require.Equal(t, int64(1), totals(t, database, live).SpentNs)
	require.Greater(t, counterValue(t, "unkey_gcra_cleanup_names_skipped_total"), skippedBefore)
}

func TestLimiter_DecidesAtRequestTime(t *testing.T) {
	database := newDatabase(t)
	clk := clock.NewTestClock(time.Now().Round(0))
	limiter := newLimiter(t, database, clk)
	req := gcra.Request{Name: uid.New(uid.TestPrefix), Rate: gcra.Rate{Limit: 1, Duration: 10 * time.Second}, Cost: 1, Time: clk.Now()}

	require.Equal(t, 1, allowN(t, limiter, req, 2))

	req.Time = clk.Now().Add(10 * time.Second)
	result, err := limiter.Allow(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.Allowed, "the limiter clock has not moved, but the request time has")
	require.Equal(t, req.Time.Add(10*time.Second), result.ResetAt)
}
