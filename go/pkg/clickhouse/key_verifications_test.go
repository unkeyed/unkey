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
	t.Logf("workspace: %s", workspaceID)
	numRecords := 1_000_000

	// Time range: between now and 1 year ago
	endTime := time.Now()
	startTime := endTime.Add(-3 * 24 * time.Hour)

	// Generate realistic test parameters
	numKeys := max(1, min(numRecords/100, 1000))
	numKeySpaces := max(1, min(numRecords/1000, 100))
	numIdentities := max(1, min(numRecords/500, 200))

	keys := array.Fill(numKeys, func() string { return uid.New(uid.KeyPrefix) })
	keySpaces := array.Fill(numKeySpaces, func() string { return uid.New(uid.KeyAuthPrefix) })
	identities := array.Fill(numIdentities, func() string { return uid.New(uid.IdentityPrefix) })

	outcomes := []string{"VALID", "INVALID", "EXPIRED", "RATE_LIMITED", "DISABLED"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	tags := [][]string{
		{"production", "api"},
		{"test", "integration"},
		{"load", "performance"},
		{"development"},
		{},
	}

	t0 := time.Now()
	// Source of truth: track all inserted rows
	verifications := array.Fill(numRecords, func() schema.KeyVerificationV2 {
		timeRange := endTime.Sub(startTime)
		randomOffset := time.Duration(rand.Int63n(int64(timeRange)))
		timestamp := startTime.Add(randomOffset)

		latency := rand.ExpFloat64()*50 + 1 // 1-100ms base range
		if rand.Float64() < 0.1 {           // 10% chance of high latency
			latency += rand.Float64() * 400 // Up to 500ms
		}
		return schema.KeyVerificationV2{
			RequestID:    uid.New(uid.RequestPrefix),
			Time:         timestamp.UnixMilli(),
			WorkspaceID:  workspaceID,
			IdentityID:   array.Random(identities),
			KeySpaceID:   array.Random(keySpaces),
			Outcome:      array.Random(outcomes),
			Region:       array.Random(regions),
			Tags:         array.Random(tags),
			KeyID:        array.Random(keys),
			SpentCredits: rand.Int63n(10),
			Latency:      latency,
		}

	})
	t.Logf("Generated %d verifications in %s", len(verifications), time.Since(t0))

	t0 = time.Now()

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO key_verifications_raw_v2")
	require.NoError(t, err)

	for _, row := range verifications {
		err = batch.AppendStruct(&row)
		require.NoError(t, err)
	}
	err = batch.Send()
	require.NoError(t, err)
	t.Logf("Inserted %d verifications in %s", batch.Rows(), time.Since(t0))

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		rawCount := uint64(0)
		err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM key_verifications_raw_v2 WHERE workspace_id = ?", workspaceID).Scan(&rawCount)
		require.NoError(c, err)
		require.Equal(c, len(verifications), int(rawCount))
	}, time.Minute, time.Second)

	t.Run("all outcomes are correct", func(t *testing.T) {
		countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerificationV2) map[string]int {
			acc[v.Outcome] = acc[v.Outcome] + 1
			return acc

		}, map[string]int{})

		for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
			t.Run(table, func(t *testing.T) {

				for outcome, count := range countByOutcome {

					require.EventuallyWithT(t, func(c *assert.CollectT) {

						queried := int64(0)
						err = conn.QueryRow(ctx, "SELECT SUM(count) FROM ? WHERE workspace_id = ? AND outcome = ?;", table, workspaceID, outcome).Scan(&queried)
						require.NoError(c, err)
						require.Equal(c, count, int(queried))
					}, time.Minute, time.Second)
				}

			})
		}
	})

	t.Run("latency aggregates are correct", func(t *testing.T) {
		latencies := array.Map(verifications, func(v schema.KeyVerificationV2) float64 {
			return v.Latency
		})
		avg := calculateAverage(latencies)
		p75 := percentile(latencies, 0.75)
		p99 := percentile(latencies, 0.99)

		for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
			t.Run(table, func(t *testing.T) {
				var (
					queriedAvg float64
					queriedP75 float32
					queriedP99 float32
				)
				err = conn.QueryRow(ctx, "SELECT avgMerge(latency_avg), quantilesTDigestMerge(0.75)(latency_p75)[1], quantilesTDigestMerge(0.99)(latency_p99)[1] FROM ? WHERE workspace_id = ?;", table, workspaceID).Scan(&queriedAvg, &queriedP75, &queriedP99)
				require.NoError(t, err)

				require.InDelta(t, avg, queriedAvg, 0.01)
				require.InDelta(t, p75, float64(queriedP75), 1.0)
				require.InDelta(t, p99, float64(queriedP99), 1.0)

			})
		}
	})
}
