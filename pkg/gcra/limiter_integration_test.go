package gcra_test

import (
	"context"
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
