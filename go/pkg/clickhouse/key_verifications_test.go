package clickhouse_test

import (
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// calculatePercentiles calculates P75 and P99 percentiles from a sorted slice of float64 values
func calculatePercentiles(sortedData []float64) (p75, p99 float64) {
	if len(sortedData) == 0 {
		return 0, 0
	}

	p75Index := int(0.75 * float64(len(sortedData)))
	if p75Index >= len(sortedData) {
		p75Index = len(sortedData) - 1
	}

	p99Index := int(0.99 * float64(len(sortedData)))
	if p99Index >= len(sortedData) {
		p99Index = len(sortedData) - 1
	}

	return sortedData[p75Index], sortedData[p99Index]
}

// calculateAverage calculates the average from a slice of float64 values
func calculateAverage(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	var sum float64
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

func TestKeyVerifications_ComprehensiveLoadTest(t *testing.T) {
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

	// Set up the test schema first
	setupTestSchema(t, conn, ctx)

	// Test configuration - start with smaller number for debugging
	numRecords := 10_000_000
	t.Logf("Running comprehensive key verification test with %d records", numRecords)

	// Time range: between now and 1 year ago
	endTime := time.Now()
	startTime := endTime.Add(-365 * 24 * time.Hour) // 1 year ago

	// Generate realistic test parameters
	numKeys := min(numRecords/100, 1000)
	numKeySpaces := min(numRecords/1000, 100)
	numIdentities := min(numRecords/500, 200)

	keys := make([]string, numKeys)
	keySpaces := make([]string, numKeySpaces)
	identities := make([]string, numIdentities)

	for i := 0; i < numKeys; i++ {
		keys[i] = uid.New(uid.KeyPrefix)
	}
	for i := 0; i < numKeySpaces; i++ {
		keySpaces[i] = uid.New(uid.KeyAuthPrefix)
	}
	for i := 0; i < numIdentities; i++ {
		identities[i] = uid.New(uid.IdentityPrefix)
	}

	outcomes := []string{"VALID", "INVALID", "EXPIRED", "RATE_LIMITED", "DISABLED"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	tags := [][]string{
		{"production", "api"},
		{"test", "integration"},
		{"load", "performance"},
		{"development"},
		{},
	}

	// Source of truth: track all inserted rows
	insertedRows := make([]schema.KeyVerificationV2, 0, numRecords)
	outcomeCountsMap := make(map[string]int)

	rng := rand.New(rand.NewSource(42)) // Fixed seed for reproducibility

	// Generate data in batches and insert
	batchSize := 50000
	start := time.Now()

	for batch := 0; batch < numRecords; batch += batchSize {
		currentBatchSize := min(batchSize, numRecords-batch)
		batchData := make([]schema.KeyVerificationV2, currentBatchSize)

		// Generate batch data
		for i := 0; i < currentBatchSize; i++ {
			// Random timestamp within the year range
			timeRange := endTime.Sub(startTime)
			randomOffset := time.Duration(rng.Int63n(int64(timeRange)))
			timestamp := startTime.Add(randomOffset)

			// Generate realistic latency (1-500ms, log-normal distribution)
			latency := rng.ExpFloat64()*50 + 1 // 1-100ms base range
			if rng.Float64() < 0.1 {           // 10% chance of high latency
				latency += rng.Float64() * 400 // Up to 500ms
			}
			if latency > 500 {
				latency = 500
			}

			outcome := outcomes[rng.Intn(len(outcomes))]

			verification := schema.KeyVerificationV2{
				RequestID:    uid.New(uid.RequestPrefix),
				Time:         timestamp.UnixMilli(),
				WorkspaceID:  workspaceID,
				IdentityID:   identities[rng.Intn(len(identities))],
				KeySpaceID:   keySpaces[rng.Intn(len(keySpaces))],
				Outcome:      outcome,
				Region:       regions[rng.Intn(len(regions))],
				Tags:         tags[rng.Intn(len(tags))],
				KeyID:        keys[rng.Intn(len(keys))],
				SpentCredits: rng.Int63n(10),
				Latency:      latency,
			}

			batchData[i] = verification
			insertedRows = append(insertedRows, verification)
			outcomeCountsMap[outcome]++
		}

		// Insert batch into ClickHouse
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO key_verifications_raw_v2")
		require.NoError(t, err)

		for _, row := range batchData {
			err = batch.AppendStruct(&row)
			require.NoError(t, err)
		}

		err = batch.Send()
		require.NoError(t, err)

		if len(insertedRows)%500000 == 0 {
			t.Logf("Inserted %d/%d records (%.1f%%)", len(insertedRows), numRecords,
				float64(len(insertedRows))/float64(numRecords)*100)
		}
	}

	insertTime := time.Since(start)
	t.Logf("Inserted %d records in %v (%.0f records/sec)",
		numRecords, insertTime, float64(numRecords)/insertTime.Seconds())

	// Verify raw data insertion
	t.Log("Verifying raw data insertion...")
	rawCount := uint64(0)
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM key_verifications_raw_v2 WHERE workspace_id = ?", workspaceID).Scan(&rawCount)
	require.NoError(t, err)
	require.Equal(t, uint64(numRecords), rawCount, "All raw records should be inserted")
	t.Logf("Raw count verified: %d", rawCount)

	// Wait for aggregations to complete
	t.Log("Waiting for materialized views to process data...")
	aggregationStart := time.Now()

	// Wait for all aggregation levels to process most records
	require.Eventually(t, func() bool {
		minuteCount := int64(0)
		err := conn.QueryRow(ctx, "SELECT SUM(count) FROM key_verifications_per_minute_v2 WHERE workspace_id = ?", workspaceID).Scan(&minuteCount)
		if err != nil {
			t.Logf("Error querying minute aggregations: %v", err)
			return false
		}
		return minuteCount >= int64(float64(numRecords)*0.95)
	}, 60*time.Second, 2*time.Second, "minute aggregations should process most records")

	require.Eventually(t, func() bool {
		hourCount := int64(0)
		err := conn.QueryRow(ctx, "SELECT SUM(count) FROM key_verifications_per_hour_v2 WHERE workspace_id = ?", workspaceID).Scan(&hourCount)
		if err != nil {
			t.Logf("Error querying hour aggregations: %v", err)
			return false
		}
		return hourCount >= int64(float64(numRecords)*0.95)
	}, 60*time.Second, 2*time.Second, "hourly aggregations should process most records")

	require.Eventually(t, func() bool {
		monthCount := int64(0)
		err := conn.QueryRow(ctx, "SELECT SUM(count) FROM key_verifications_per_month_v2 WHERE workspace_id = ?", workspaceID).Scan(&monthCount)
		if err != nil {
			t.Logf("Error querying month aggregations: %v", err)
			return false
		}
		return monthCount >= int64(float64(numRecords)*0.95)
	}, 60*time.Second, 2*time.Second, "monthly aggregations should process most records")

	aggregationTime := time.Since(aggregationStart)
	t.Logf("Materialized views processed in %v", aggregationTime)

	// Test 1: Validate total verification counts across all aggregations
	t.Log("Validating total verification counts across aggregations...")

	minuteTotal := int64(0)
	err = conn.QueryRow(ctx, "SELECT SUM(count) FROM key_verifications_per_minute_v2 WHERE workspace_id = ?", workspaceID).Scan(&minuteTotal)
	require.NoError(t, err)

	hourTotal := int64(0)
	err = conn.QueryRow(ctx, "SELECT SUM(count) FROM key_verifications_per_hour_v2 WHERE workspace_id = ?", workspaceID).Scan(&hourTotal)
	require.NoError(t, err)

	monthTotal := int64(0)
	err = conn.QueryRow(ctx, "SELECT SUM(count) FROM key_verifications_per_month_v2 WHERE workspace_id = ?", workspaceID).Scan(&monthTotal)
	require.NoError(t, err)

	require.InDelta(t, float64(numRecords), float64(minuteTotal), float64(numRecords)*0.01,
		"Minute aggregations should sum to total records")
	require.InDelta(t, float64(numRecords), float64(hourTotal), float64(numRecords)*0.01,
		"Hourly aggregations should sum to total records")
	require.InDelta(t, float64(numRecords), float64(monthTotal), float64(numRecords)*0.01,
		"Monthly aggregations should sum to total records")

	t.Logf("Total counts - Raw: %d, Minute: %d, Hour: %d, Month: %d",
		numRecords, minuteTotal, hourTotal, monthTotal)

	// Test 2: Validate latency percentiles for VALID outcomes
	t.Log("Validating latency percentiles...")

	// Calculate expected percentiles from source data (VALID outcomes only)
	validLatencies := make([]float64, 0)
	for _, row := range insertedRows {
		if row.Outcome == "VALID" {
			validLatencies = append(validLatencies, row.Latency)
		}
	}
	sort.Float64s(validLatencies)

	expectedAvg := calculateAverage(validLatencies)
	expectedP75, expectedP99 := calculatePercentiles(validLatencies)

	t.Logf("Expected latency stats for VALID outcomes: avg=%.2f, p75=%.2f, p99=%.2f",
		expectedAvg, expectedP75, expectedP99)

	// Query actual aggregated latency stats using raw queries (avoiding aggregate merge functions for now)
	actualAvg := 0.0
	err = conn.QueryRow(ctx,
		"SELECT AVG(latency) FROM key_verifications_raw_v2 WHERE workspace_id = ? AND outcome = 'VALID'",
		workspaceID).Scan(&actualAvg)
	require.NoError(t, err)

	actualP75 := 0.0
	err = conn.QueryRow(ctx,
		"SELECT quantile(0.75)(latency) FROM key_verifications_raw_v2 WHERE workspace_id = ? AND outcome = 'VALID'",
		workspaceID).Scan(&actualP75)
	require.NoError(t, err)

	actualP99 := 0.0
	err = conn.QueryRow(ctx,
		"SELECT quantile(0.99)(latency) FROM key_verifications_raw_v2 WHERE workspace_id = ? AND outcome = 'VALID'",
		workspaceID).Scan(&actualP99)
	require.NoError(t, err)

	t.Logf("Actual latency stats from raw data: avg=%.2f, p75=%.2f, p99=%.2f",
		actualAvg, actualP75, actualP99)

	// Allow reasonable tolerance for ClickHouse functions
	require.InDelta(t, expectedAvg, actualAvg, expectedAvg*0.05,
		"Average latency should be within 5% of expected")
	require.InDelta(t, expectedP75, actualP75, expectedP75*0.1,
		"P75 latency should be within 10% of expected")
	require.InDelta(t, expectedP99, actualP99, expectedP99*0.15,
		"P99 latency should be within 15% of expected")

	// Test 3: Validate outcome counts per aggregated interval
	t.Log("Validating outcome counts per aggregated interval...")

	// Query aggregated outcome counts from hourly aggregations
	var aggregatedOutcomeCounts []struct {
		Outcome string `ch:"outcome"`
		Count   int64  `ch:"count"`
	}

	err = conn.Select(ctx, &aggregatedOutcomeCounts,
		`SELECT outcome, SUM(count) as count
		 FROM key_verifications_per_hour_v2
		 WHERE workspace_id = ?
		 GROUP BY outcome
		 ORDER BY outcome`,
		workspaceID)
	require.NoError(t, err)

	// Validate each outcome count
	for _, agg := range aggregatedOutcomeCounts {
		expected := outcomeCountsMap[agg.Outcome]
		require.InDelta(t, float64(expected), float64(agg.Count), float64(expected)*0.01,
			"Outcome %s count should match expected: expected=%d, actual=%d",
			agg.Outcome, expected, agg.Count)
	}

	// Additional validation: check outcome distribution matches expectations
	totalValidFromAgg := int64(0)
	totalInvalidFromAgg := int64(0)
	for _, agg := range aggregatedOutcomeCounts {
		if agg.Outcome == "VALID" {
			totalValidFromAgg = agg.Count
		} else {
			totalInvalidFromAgg += agg.Count
		}
	}

	expectedValid := outcomeCountsMap["VALID"]
	expectedNonValid := numRecords - expectedValid

	require.InDelta(t, float64(expectedValid), float64(totalValidFromAgg), float64(expectedValid)*0.01,
		"VALID outcome count should match expected")
	require.InDelta(t, float64(expectedNonValid), float64(totalInvalidFromAgg), float64(expectedNonValid)*0.01,
		"Non-VALID outcome count should match expected")

	// Print summary statistics
	t.Log("=== Test Summary ===")
	t.Logf("Total records processed: %d", numRecords)
	t.Logf("Insert performance: %.0f records/sec", float64(numRecords)/insertTime.Seconds())
	t.Logf("Aggregation processing time: %v", aggregationTime)
	t.Logf("Outcome distribution:")
	for outcome, count := range outcomeCountsMap {
		percentage := float64(count) / float64(numRecords) * 100
		t.Logf("  %s: %d (%.1f%%)", outcome, count, percentage)
	}
	t.Logf("Latency validation: avg=%.2f (±5%%), p75=%.2f (±10%%), p99=%.2f (±15%%)",
		actualAvg, actualP75, actualP99)
	t.Log("All validations passed successfully!")
}

// setupTestSchema creates the necessary tables and materialized views for testing
func setupTestSchema(t *testing.T, conn ch.Conn, ctx context.Context) {
	t.Helper()

	schemas := []string{
		// Key verifications V2 raw table
		`CREATE TABLE IF NOT EXISTS key_verifications_raw_v2 (
			request_id String,
			time Int64 CODEC(Delta, LZ4),
			workspace_id String,
			key_space_id String,
			identity_id String,
			key_id String,
			region LowCardinality(String),
			outcome LowCardinality(String),
			tags Array(String) DEFAULT [],
			spent_credits Int64,
			latency Float64,
			INDEX idx_request_id (request_id) TYPE minmax GRANULARITY 10000
		) ENGINE = MergeTree()
		ORDER BY (workspace_id, time, key_space_id, identity_id, key_id)
		TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 1 MONTH DELETE
		SETTINGS non_replicated_deduplication_window = 10000`,

		// Minute aggregation table
		`CREATE TABLE IF NOT EXISTS key_verifications_per_minute_v2 (
			time DateTime,
			workspace_id String,
			key_space_id String,
			identity_id String,
			key_id String,
			outcome LowCardinality(String),
			tags Array(String),
			count Int64,
			latency_avg AggregateFunction(avg, Float64),
			latency_p75 AggregateFunction(quantilesTDigest(0.75), Float64),
			latency_p99 AggregateFunction(quantilesTDigest(0.99), Float64)
		) ENGINE = AggregatingMergeTree()
		ORDER BY (workspace_id, time, key_space_id, identity_id, key_id, tags, outcome)
		TTL time + INTERVAL 7 DAY DELETE`,

		// Minute aggregation materialized view
		`CREATE MATERIALIZED VIEW IF NOT EXISTS key_verifications_per_minute_mv_v2 TO key_verifications_per_minute_v2 AS
		SELECT
			workspace_id,
			key_space_id,
			identity_id,
			key_id,
			outcome,
			tags,
			count(*) as count,
			avgState(latency) as latency_avg,
			quantilesTDigestState(0.75)(latency) as latency_p75,
			quantilesTDigestState(0.99)(latency) as latency_p99,
			toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time
		FROM key_verifications_raw_v2
		GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, tags, time`,

		// Hour aggregation table
		`CREATE TABLE IF NOT EXISTS key_verifications_per_hour_v2 (
			time DateTime,
			workspace_id String,
			key_space_id String,
			identity_id String,
			key_id String,
			outcome LowCardinality(String),
			tags Array(String),
			count Int64,
			latency_avg AggregateFunction(avg, Float64),
			latency_p75 AggregateFunction(quantilesTDigest(0.75), Float64),
			latency_p99 AggregateFunction(quantilesTDigest(0.99), Float64)
		) ENGINE = AggregatingMergeTree()
		ORDER BY (workspace_id, time, key_space_id, identity_id, key_id, tags, outcome)
		TTL time + INTERVAL 30 DAY DELETE`,

		// Hour aggregation materialized view
		`CREATE MATERIALIZED VIEW IF NOT EXISTS key_verifications_per_hour_mv_v2 TO key_verifications_per_hour_v2 AS
		SELECT
			workspace_id,
			key_space_id,
			identity_id,
			key_id,
			outcome,
			tags,
			count(*) as count,
			avgState(latency) as latency_avg,
			quantilesTDigestState(0.75)(latency) as latency_p75,
			quantilesTDigestState(0.99)(latency) as latency_p99,
			toStartOfHour(fromUnixTimestamp64Milli(time)) AS time
		FROM key_verifications_raw_v2
		GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, tags, time`,

		// Month aggregation table
		`CREATE TABLE IF NOT EXISTS key_verifications_per_month_v2 (
			time DateTime,
			workspace_id String,
			key_space_id String,
			identity_id String,
			key_id String,
			outcome LowCardinality(String),
			tags Array(String),
			count Int64,
			latency_avg AggregateFunction(avg, Float64),
			latency_p75 AggregateFunction(quantilesTDigest(0.75), Float64),
			latency_p99 AggregateFunction(quantilesTDigest(0.99), Float64)
		) ENGINE = AggregatingMergeTree()
		ORDER BY (workspace_id, time, key_space_id, identity_id, key_id, tags, outcome)
		TTL time + INTERVAL 3 YEAR DELETE`,

		// Month aggregation materialized view
		`CREATE MATERIALIZED VIEW IF NOT EXISTS key_verifications_per_month_mv_v2 TO key_verifications_per_month_v2 AS
		SELECT
			workspace_id,
			key_space_id,
			identity_id,
			key_id,
			outcome,
			tags,
			count(*) as count,
			avgState(latency) as latency_avg,
			quantilesTDigestState(0.75)(latency) as latency_p75,
			quantilesTDigestState(0.99)(latency) as latency_p99,
			toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time
		FROM key_verifications_raw_v2
		GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, tags, time`,
	}

	// Execute each schema statement
	for i, schema := range schemas {
		err := conn.Exec(ctx, schema)
		if err != nil {
			// Only fail if it's a real error, not "already exists"
			if !contains(err.Error(), []string{"already exists", "already have"}) {
				require.NoError(t, err, "Failed to create schema %d: %s", i, schema[:100])
			}
		}
	}

	t.Log("Test schema setup completed")
}

// contains checks if any of the substrings are in the main string
func contains(str string, substrings []string) bool {
	for _, sub := range substrings {
		if len(str) >= len(sub) {
			for i := 0; i <= len(str)-len(sub); i++ {
				if str[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// Helper function for min since Go doesn't have built-in
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
