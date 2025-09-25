package clickhouse_test

import (
	"context"
	"math/rand"
	"slices"
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

func TestKeyVerifications(t *testing.T) {
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

	batchSize := 100_000
	for i := 0; i < len(verifications); i += batchSize {
		t0 = time.Now()

		batch, err := conn.PrepareBatch(ctx, "INSERT INTO key_verifications_raw_v2")
		require.NoError(t, err)

		for _, row := range verifications[i:min(i+batchSize, len(verifications))] {
			err = batch.AppendStruct(&row)
			require.NoError(t, err)
		}
		err = batch.Send()
		require.NoError(t, err)
		t.Logf("Inserted %d verifications in %s", batch.Rows(), time.Since(t0))
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		rawCount := uint64(0)
		err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM key_verifications_raw_v2 WHERE workspace_id = ?", workspaceID).Scan(&rawCount)
		require.NoError(c, err)
		require.Equal(c, len(verifications), int(rawCount))
	}, time.Minute, time.Second)

	t.Run("totals are correct", func(t *testing.T) {

		for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
			t.Run(table, func(t *testing.T) {

				require.EventuallyWithT(t, func(c *assert.CollectT) {

					queried := int64(0)
					err := conn.QueryRow(ctx, "SELECT SUM(count) FROM ? WHERE workspace_id = ?;", table, workspaceID).Scan(&queried)
					require.NoError(c, err)
					t.Logf("expected %d, got %d", len(verifications), queried)
					require.Equal(c, len(verifications), int(queried))
				}, time.Minute, time.Second)

			})
		}
	})

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
						err := conn.QueryRow(ctx, "SELECT SUM(count) FROM ? WHERE workspace_id = ? AND outcome = ?;", table, workspaceID, outcome).Scan(&queried)
						require.NoError(c, err)
						t.Logf("%s expected %d, got %d", outcome, count, queried)
						require.Equal(c, count, int(queried))
					}, time.Minute, time.Second)
				}

			})
		}
	})

	t.Run("outcomes per key are correct", func(t *testing.T) {
		t.Parallel()
		for _, keyID := range keys[:10] {

			countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerificationV2) map[string]int {
				if v.KeyID == keyID {
					acc[v.Outcome] = acc[v.Outcome] + 1
				}
				return acc

			}, map[string]int{})

			for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
				t.Run(table, func(t *testing.T) {
					t.Parallel()

					for outcome, count := range countByOutcome {

						require.EventuallyWithT(t, func(c *assert.CollectT) {

							queried := int64(0)
							err := conn.QueryRow(ctx, "SELECT SUM(count) FROM ? WHERE workspace_id = ? AND key_id = ? AND outcome = ?;", table, workspaceID, keyID, outcome).Scan(&queried)
							require.NoError(c, err)
							require.Equal(c, count, int(queried))
						}, time.Minute, time.Second)
					}

				})
			}
		}
	})
	t.Run("outcomes per identity are correct", func(t *testing.T) {
		t.Parallel()
		for _, identityID := range identities[:10] {

			countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerificationV2) map[string]int {
				if v.IdentityID == identityID {
					acc[v.Outcome] = acc[v.Outcome] + 1
				}
				return acc

			}, map[string]int{})

			for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
				t.Run(table, func(t *testing.T) {

					for outcome, count := range countByOutcome {

						require.EventuallyWithT(t, func(c *assert.CollectT) {

							queried := int64(0)
							err := conn.QueryRow(ctx, "SELECT SUM(count) FROM ? WHERE workspace_id = ? AND identity_id = ? AND outcome = ?;", table, workspaceID, identityID, outcome).Scan(&queried)
							require.NoError(c, err)
							require.Equal(c, count, int(queried))
						}, time.Minute, time.Second)
					}

				})
			}
		}
	})
	t.Run("outcomes per tag are correct", func(t *testing.T) {
		t.Parallel()
		for _, usedTags := range tags {
			if len(usedTags) == 0 {
				continue
			}
			tag := usedTags[0]
			countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerificationV2) map[string]int {
				if slices.Contains(v.Tags, tag) {
					acc[v.Outcome] = acc[v.Outcome] + 1
				}
				return acc

			}, map[string]int{})

			for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
				t.Run(table, func(t *testing.T) {
					for outcome, count := range countByOutcome {
						require.EventuallyWithT(t, func(c *assert.CollectT) {

							queried := int64(0)
							err := conn.QueryRow(ctx, "SELECT SUM(count) FROM ? WHERE workspace_id = ? AND indexOf(tags, ?) > 0 AND outcome = ?;", table, workspaceID, tag, outcome).Scan(&queried)
							require.NoError(c, err)
							require.Equal(c, count, int(queried))
						}, time.Minute, time.Second)
					}
				})
			}
		}
	})
	t.Run("latency aggregates are correct", func(t *testing.T) {
		t.Parallel()
		latencies := array.Map(verifications, func(v schema.KeyVerificationV2) float64 {
			return v.Latency
		})
		avg := calculateAverage(latencies)
		p75 := percentile(latencies, 0.75)
		p99 := percentile(latencies, 0.99)

		for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
			t.Run(table, func(t *testing.T) {
				t.Parallel()
				var (
					queriedAvg float64
					queriedP75 float32
					queriedP99 float32
				)
				err := conn.QueryRow(ctx, "SELECT avgMerge(latency_avg), quantilesTDigestMerge(0.75)(latency_p75)[1], quantilesTDigestMerge(0.99)(latency_p99)[1] FROM ? WHERE workspace_id = ?;", table, workspaceID).Scan(&queriedAvg, &queriedP75, &queriedP99)
				require.NoError(t, err)

				require.InDelta(t, avg, queriedAvg, 0.01)
				require.InDelta(t, p75, float64(queriedP75), 1.0)
				require.InDelta(t, p99, float64(queriedP99), 1.0)

			})
		}
	})

	t.Run("credits spent globally are correct", func(t *testing.T) {
		t.Parallel()
		credits := array.Reduce(verifications, func(acc int64, v schema.KeyVerificationV2) int64 {
			return acc + v.SpentCredits
		}, int64(0))

		for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
			t.Run(table, func(t *testing.T) {
				t.Parallel()
				var queried int64
				err := conn.QueryRow(ctx, "SELECT sum(spent_credits) FROM ? WHERE workspace_id = ?;", table, workspaceID).Scan(&queried)
				require.NoError(t, err)

				require.Equal(t, credits, queried)

			})
		}
	})
	t.Run("credits spent per identity are correct", func(t *testing.T) {
		t.Parallel()
		for _, identityID := range identities[:10] {
			credits := array.Reduce(verifications, func(acc int64, v schema.KeyVerificationV2) int64 {
				if v.IdentityID == identityID {
					acc += v.SpentCredits
				}
				return acc
			}, int64(0))

			for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
				t.Run(table, func(t *testing.T) {
					t.Parallel()
					var queried int64
					err := conn.QueryRow(ctx, "SELECT sum(spent_credits) FROM ? WHERE workspace_id = ? AND identity_id = ?;", table, workspaceID, identityID).Scan(&queried)
					require.NoError(t, err)

					require.Equal(t, credits, queried)

				})
			}

		}
	})

	t.Run("credits spent per key are correct", func(t *testing.T) {
		t.Parallel()
		for _, keyID := range keys[:10] {
			credits := array.Reduce(verifications, func(acc int64, v schema.KeyVerificationV2) int64 {
				if v.KeyID == keyID {
					acc += v.SpentCredits
				}
				return acc
			}, int64(0))

			for _, table := range []string{"key_verifications_per_minute_v2", "key_verifications_per_hour_v2", "key_verifications_per_day_v2", "key_verifications_per_month_v2"} {
				t.Run(table, func(t *testing.T) {
					t.Parallel()
					var queried int64
					err := conn.QueryRow(ctx, "SELECT sum(spent_credits) FROM ? WHERE workspace_id = ? AND key_id = ?;", table, workspaceID, keyID).Scan(&queried)
					require.NoError(t, err)

					require.Equal(t, credits, queried)

				})
			}

		}
	})

	t.Run("billing per workspace is correct", func(t *testing.T) {
		t.Parallel()
		billableVerifications := array.Reduce(verifications, func(acc int64, v schema.KeyVerificationV2) int64 {
			if v.Outcome == "VALID" {
				acc += 1
			}
			return acc
		}, int64(0))

		var queried int64
		err := conn.QueryRow(ctx, "SELECT sum(count) FROM billable_verifications_per_month_v2 WHERE workspace_id = ?;", workspaceID).Scan(&queried)

		require.NoError(t, err)

		require.Equal(t, billableVerifications, queried)

	})
}
