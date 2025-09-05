package clickhouse_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/array"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestRatelimits_ComprehensiveLoadTest(t *testing.T) {
	dsn := containers.ClickHouse(t)

	opts, err := ch.ParseDSN(dsn)
	require.NoError(t, err)

	conn, err := ch.Open(opts)
	require.NoError(t, err)
	defer conn.Close()

	err = conn.Ping(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	workspaceID := uid.New(uid.WorkspacePrefix)
	t.Logf("workspace: %s", workspaceID)
	numRecords := 1_000_000

	// Time range: between now and 3 days ago
	endTime := time.Now()
	startTime := endTime.Add(-3 * 24 * time.Hour)

	// Generate realistic test parameters for ratelimits
	numNamespaces := max(1, min(numRecords/1000, 100))   // 100-1000 namespaces
	numIdentifiers := max(1, min(numRecords/100, 10000)) // 1000-10000 identifiers

	namespaces := array.Fill(numNamespaces, func() string { return uid.New(uid.RatelimitNamespacePrefix) })
	identifiers := array.Fill(numIdentifiers, func() string {
		// Mix different identifier types: user IDs, API keys, IP addresses
		switch rand.Intn(4) {
		case 0:
			return uid.New(uid.IdentityPrefix)
		case 1:
			return uid.New(uid.KeyPrefix)
		case 2:
			// IP address format
			return array.Random([]string{
				"192.168.1.100", "10.0.0.50", "172.16.0.200",
				"203.0.113.45", "198.51.100.25", "192.0.2.150",
			})
		default:
			// Custom identifier
			return "custom_" + uid.New("")
		}
	})

	t0 := time.Now()
	// Source of truth: track all inserted ratelimit records
	ratelimits := array.Fill(numRecords, func() schema.RatelimitV2 {
		timeRange := endTime.Sub(startTime)
		randomOffset := time.Duration(rand.Int63n(int64(timeRange)))
		timestamp := startTime.Add(randomOffset)

		// Simulate realistic ratelimit patterns:
		// - 70% pass (under limit)
		// - 30% fail (over limit)
		passed := rand.Float64() < 0.7

		// Ratelimit check latency: typically very fast
		latency := rand.ExpFloat64()*2 + 0.1 // 0.1-5ms base range
		if rand.Float64() < 0.05 {           // 5% chance of slower checks
			latency += rand.Float64() * 10 // Up to 15ms for complex checks
		}

		return schema.RatelimitV2{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        timestamp.UnixMilli(),
			WorkspaceID: workspaceID,
			NamespaceID: array.Random(namespaces),
			Identifier:  array.Random(identifiers),
			Passed:      passed,
			Latency:     latency,
		}
	})
	t.Logf("Generated %d ratelimits in %s", len(ratelimits), time.Since(t0))

	t0 = time.Now()

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO ratelimits_raw_v2")
	require.NoError(t, err)

	for _, row := range ratelimits {
		err = batch.AppendStruct(&row)
		require.NoError(t, err)
	}
	err = batch.Send()
	require.NoError(t, err)
	t.Logf("Inserted %d ratelimits in %s", batch.Rows(), time.Since(t0))

	// Wait for raw data to be available
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		rawCount := uint64(0)
		err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM ratelimits_raw_v2 WHERE workspace_id = ?", workspaceID).Scan(&rawCount)
		require.NoError(c, err)
		require.Equal(c, len(ratelimits), int(rawCount))
	}, time.Minute, time.Second)

	t.Run("pass/total counts are correct", func(t *testing.T) {
		// Calculate expected totals from source data
		totalPassed := array.Reduce(ratelimits, func(acc int, r schema.RatelimitV2) int {
			if r.Passed {
				return acc + 1
			}
			return acc
		}, 0)

		totalRequests := len(ratelimits)

		for _, table := range []string{"ratelimits_per_minute_v2", "ratelimits_per_hour_v2", "ratelimits_per_day_v2", "ratelimits_per_month_v2"} {
			t.Run(table, func(t *testing.T) {
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					var queriedPassed, queriedTotal int64
					err = conn.QueryRow(ctx, "SELECT sum(passed), sum(total) FROM ? WHERE workspace_id = ?", table, workspaceID).Scan(&queriedPassed, &queriedTotal)
					require.NoError(c, err)
					require.Equal(c, totalPassed, int(queriedPassed), "passed count should match")
					require.Equal(c, totalRequests, int(queriedTotal), "total count should match")
				}, time.Minute, time.Second)
			})
		}
	})

	t.Run("latency aggregates are correct", func(t *testing.T) {
		latencies := array.Map(ratelimits, func(r schema.RatelimitV2) float64 {
			return r.Latency
		})
		avg := calculateAverage(latencies)
		p75 := percentile(latencies, 0.75)
		p99 := percentile(latencies, 0.99)

		for _, table := range []string{"ratelimits_per_minute_v2", "ratelimits_per_hour_v2", "ratelimits_per_day_v2", "ratelimits_per_month_v2"} {
			t.Run(table, func(t *testing.T) {
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					var (
						queriedAvg float64
						queriedP75 float32
						queriedP99 float32
					)
					err = conn.QueryRow(ctx, "SELECT avgMerge(latency_avg), quantilesTDigestMerge(0.75)(latency_p75)[1], quantilesTDigestMerge(0.99)(latency_p99)[1] FROM ? WHERE workspace_id = ?", table, workspaceID).Scan(&queriedAvg, &queriedP75, &queriedP99)
					require.NoError(c, err)

					require.InDelta(c, avg, queriedAvg, 0.01, "average latency should match")
					require.InDelta(c, p75, float64(queriedP75), 0.5, "75th percentile should match")
					require.InDelta(c, p99, float64(queriedP99), 1.0, "99th percentile should match")
				}, time.Minute, time.Second)
			})
		}
	})

	t.Run("namespace-level aggregation is correct", func(t *testing.T) {
		// Group by namespace and calculate expected totals
		namespaceStats := array.Reduce(ratelimits, func(acc map[string]struct{ passed, total int }, r schema.RatelimitV2) map[string]struct{ passed, total int } {
			stats := acc[r.NamespaceID]
			stats.total++
			if r.Passed {
				stats.passed++
			}
			acc[r.NamespaceID] = stats
			return acc
		}, make(map[string]struct{ passed, total int }))

		for _, table := range []string{"ratelimits_per_minute_v2", "ratelimits_per_hour_v2", "ratelimits_per_day_v2", "ratelimits_per_month_v2"} {
			t.Run(table, func(t *testing.T) {
				for namespaceID, expectedStats := range namespaceStats {
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						var queriedPassed, queriedTotal int64
						err = conn.QueryRow(ctx, "SELECT sum(passed), sum(total) FROM ? WHERE workspace_id = ? AND namespace_id = ?", table, workspaceID, namespaceID).Scan(&queriedPassed, &queriedTotal)
						require.NoError(c, err)
						require.Equal(c, expectedStats.passed, int(queriedPassed), "passed count for namespace %s should match", namespaceID)
						require.Equal(c, expectedStats.total, int(queriedTotal), "total count for namespace %s should match", namespaceID)
					}, time.Minute, time.Second)
				}
			})
		}
	})

	t.Run("identifier-level aggregation is correct", func(t *testing.T) {
		// Group by identifier and calculate expected totals
		identifierStats := array.Reduce(ratelimits, func(acc map[string]struct{ passed, total int }, r schema.RatelimitV2) map[string]struct{ passed, total int } {
			stats := acc[r.Identifier]
			stats.total++
			if r.Passed {
				stats.passed++
			}
			acc[r.Identifier] = stats
			return acc
		}, make(map[string]struct{ passed, total int }))

		// Test a sample of identifiers to avoid overwhelming the test
		sampleIdentifiers := array.Fill(min(50, len(identifierStats)), func() string {
			return array.Random(identifiers)
		})

		for _, table := range []string{"ratelimits_per_minute_v2", "ratelimits_per_hour_v2", "ratelimits_per_day_v2", "ratelimits_per_month_v2"} {
			t.Run(table, func(t *testing.T) {
				for _, identifier := range sampleIdentifiers {
					expectedStats, exists := identifierStats[identifier]
					if !exists {
						continue
					}

					require.EventuallyWithT(t, func(c *assert.CollectT) {
						var queriedPassed, queriedTotal int64
						err = conn.QueryRow(ctx, "SELECT sum(passed), sum(total) FROM ? WHERE workspace_id = ? AND identifier = ?", table, workspaceID, identifier).Scan(&queriedPassed, &queriedTotal)
						require.NoError(c, err)
						require.Equal(c, expectedStats.passed, int(queriedPassed), "passed count for identifier %s should match", identifier)
						require.Equal(c, expectedStats.total, int(queriedTotal), "total count for identifier %s should match", identifier)
					}, time.Minute, time.Second)
				}
			})
		}
	})

	t.Run("pass rate analysis globally", func(t *testing.T) {
		// Calculate overall pass rate
		total := 0
		passed := 0
		for _, r := range ratelimits {
			total++
			if r.Passed {
				passed += 1
			}
		}
		for _, table := range []string{"ratelimits_per_minute_v2", "ratelimits_per_hour_v2", "ratelimits_per_day_v2", "ratelimits_per_month_v2"} {
			t.Run(table, func(t *testing.T) {
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					var queriedPassed, queriedTotal int64
					err = conn.QueryRow(ctx, "SELECT sum(passed), sum(total) FROM ? WHERE workspace_id = ?", table, workspaceID).Scan(&queriedPassed, &queriedTotal)
					require.NoError(c, err)

					require.Equal(c, total, int(queriedTotal), "total queries should match")
					require.Equal(c, passed, int(queriedPassed), "passed queries should match")

				}, time.Minute, time.Second)
			})
		}
	})

	t.Run("pass rate analysis per identifier", func(t *testing.T) {
		t.Parallel()
		// Calculate overall pass rate
		for _, identifier := range identifiers[:10] {
			total := 0
			passed := 0
			for _, r := range ratelimits {
				if r.Identifier == identifier {
					total++
					if r.Passed {
						passed += 1
					}
				}
			}
			for _, table := range []string{"ratelimits_per_minute_v2", "ratelimits_per_hour_v2", "ratelimits_per_day_v2", "ratelimits_per_month_v2"} {
				t.Run(table, func(t *testing.T) {
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						var queriedPassed, queriedTotal int64
						err = conn.QueryRow(ctx, "SELECT sum(passed), sum(total) FROM ? WHERE workspace_id = ? AND identifier = ?", table, workspaceID, identifier).Scan(&queriedPassed, &queriedTotal)
						require.NoError(c, err)

						require.Equal(c, total, int(queriedTotal), "total queries should match")
						require.Equal(c, passed, int(queriedPassed), "passed queries should match")
						t.Parallel()

					}, time.Minute, time.Second)
				})
			}
		}
	})

	t.Run("pass rate analysis per namespace", func(t *testing.T) {

		for _, namespace := range namespaces {
			// Calculate overall pass rate
			total := 0
			passed := 0
			for _, r := range ratelimits {
				if r.NamespaceID == namespace {
					total++
					if r.Passed {
						passed += 1
					}
				}
			}
			for _, table := range []string{"ratelimits_per_minute_v2", "ratelimits_per_hour_v2", "ratelimits_per_day_v2", "ratelimits_per_month_v2"} {
				t.Run(table, func(t *testing.T) {
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						var queriedPassed, queriedTotal int64
						err = conn.QueryRow(ctx, "SELECT sum(passed), sum(total) FROM ? WHERE workspace_id = ? AND namespace_id = ?", table, workspaceID, namespace).Scan(&queriedPassed, &queriedTotal)
						require.NoError(c, err)

						require.Equal(c, total, int(queriedTotal), "total queries should match")
						require.Equal(c, passed, int(queriedPassed), "passed queries should match")

					}, time.Minute, time.Second)
				})
			}
		}
	})
}
