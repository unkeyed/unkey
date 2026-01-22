package clickhouse_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestKeyVerifications(t *testing.T) {
	t.Parallel()
	dsn := containers.ClickHouse(t)

	opts, err := ch.ParseDSN(dsn)
	require.NoError(t, err)

	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

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
	keySpaces := array.Fill(numKeySpaces, func() string { return uid.New(uid.KeySpacePrefix) })
	identities := array.Fill(numIdentities, func() string { return uid.New(uid.IdentityPrefix) })

	// Map each identity to an external_id (1:1 mapping to ensure external_id uniqueness per identity)
	identityToExternalID := make(map[string]string)
	for _, identityID := range identities {
		identityToExternalID[identityID] = "ext_" + uid.New("")
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

	t0 := time.Now()
	// Source of truth: track all inserted rows
	verifications := array.Fill(numRecords, func() schema.KeyVerification {
		timeRange := endTime.Sub(startTime)
		randomOffset := time.Duration(rand.Int63n(int64(timeRange)))
		timestamp := startTime.Add(randomOffset)

		latency := rand.ExpFloat64()*50 + 1 // 1-100ms base range
		if rand.Float64() < 0.1 {           // 10% chance of high latency
			latency += rand.Float64() * 400 // Up to 500ms
		}
		identityID := array.Random(identities)
		return schema.KeyVerification{
			RequestID:    uid.New(uid.RequestPrefix),
			Time:         timestamp.UnixMilli(),
			WorkspaceID:  workspaceID,
			IdentityID:   identityID,
			ExternalID:   identityToExternalID[identityID],
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

		batch, err := conn.PrepareBatch(ctx, "INSERT INTO default.key_verifications_raw_v2")
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
		err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE workspace_id = ?", workspaceID).Scan(&rawCount)
		require.NoError(c, err)
		require.Equal(c, len(verifications), int(rawCount))
	}, time.Minute, time.Second)

	t.Run("totals are correct", func(t *testing.T) {
		for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
			t.Run(table, func(t *testing.T) {
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					queried := int64(0)
					err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(count) FROM %s WHERE workspace_id = ?;", table), workspaceID).Scan(&queried)
					require.NoError(c, err)
					t.Logf("expected %d, got %d", len(verifications), queried)
					require.Equal(c, len(verifications), int(queried))
				}, time.Minute, time.Second)
			})
		}
	})

	t.Run("all outcomes are correct", func(t *testing.T) {
		countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerification) map[string]int {
			acc[v.Outcome] = acc[v.Outcome] + 1
			return acc
		}, map[string]int{})

		for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
			t.Run(table, func(t *testing.T) {
				for outcome, count := range countByOutcome {
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						queried := int64(0)
						err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(count) FROM %s WHERE workspace_id = ? AND outcome = ?;", table), workspaceID, outcome).Scan(&queried)
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

			countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerification) map[string]int {
				if v.KeyID == keyID {
					acc[v.Outcome] = acc[v.Outcome] + 1
				}
				return acc
			}, map[string]int{})

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				t.Run(table, func(t *testing.T) {
					t.Parallel()

					for outcome, count := range countByOutcome {
						require.EventuallyWithT(t, func(c *assert.CollectT) {
							queried := int64(0)
							err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(count) FROM %s WHERE workspace_id = ? AND key_id = ? AND outcome = ?;", table), workspaceID, keyID, outcome).Scan(&queried)
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

			countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerification) map[string]int {
				if v.IdentityID == identityID {
					acc[v.Outcome] = acc[v.Outcome] + 1
				}
				return acc
			}, map[string]int{})

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				t.Run(table, func(t *testing.T) {
					for outcome, count := range countByOutcome {
						require.EventuallyWithT(t, func(c *assert.CollectT) {
							queried := int64(0)
							err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(count) FROM %s WHERE workspace_id = ? AND identity_id = ? AND outcome = ?;", table), workspaceID, identityID, outcome).Scan(&queried)
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
			countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerification) map[string]int {
				if slices.Contains(v.Tags, tag) {
					acc[v.Outcome] = acc[v.Outcome] + 1
				}
				return acc
			}, map[string]int{})

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				t.Run(table, func(t *testing.T) {
					for outcome, count := range countByOutcome {
						require.EventuallyWithT(t, func(c *assert.CollectT) {
							queried := int64(0)
							err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(count) FROM %s WHERE workspace_id = ? AND indexOf(tags, ?) > 0 AND outcome = ?;", table), workspaceID, tag, outcome).Scan(&queried)
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
		latencies := array.Map(verifications, func(v schema.KeyVerification) float64 {
			return v.Latency
		})
		avg := calculateAverage(latencies)
		p75 := percentile(latencies, 0.75)
		p99 := percentile(latencies, 0.99)

		for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
			t.Run(table, func(t *testing.T) {
				t.Parallel()
				var (
					queriedAvg float64
					queriedP75 float32
					queriedP99 float32
				)
				err := conn.QueryRow(ctx, fmt.Sprintf("SELECT avgMerge(latency_avg), quantilesTDigestMerge(0.75)(latency_p75)[1], quantilesTDigestMerge(0.99)(latency_p99)[1] FROM %s WHERE workspace_id = ?;", table), workspaceID).Scan(&queriedAvg, &queriedP75, &queriedP99)
				require.NoError(t, err)

				require.InDelta(t, avg, queriedAvg, 0.01)
				require.InDelta(t, p75, float64(queriedP75), 1.0)
				require.InDelta(t, p99, float64(queriedP99), 1.0)
			})
		}
	})

	t.Run("credits spent globally are correct", func(t *testing.T) {
		t.Parallel()
		credits := array.Reduce(verifications, func(acc int64, v schema.KeyVerification) int64 {
			return acc + v.SpentCredits
		}, int64(0))

		for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
			t.Run(table, func(t *testing.T) {
				t.Parallel()
				var queried int64
				err := conn.QueryRow(ctx, fmt.Sprintf("SELECT sum(spent_credits) FROM %s WHERE workspace_id = ?;", table), workspaceID).Scan(&queried)
				require.NoError(t, err)

				require.Equal(t, credits, queried)
			})
		}
	})
	t.Run("credits spent per identity are correct", func(t *testing.T) {
		t.Parallel()
		for _, identityID := range identities[:10] {
			credits := array.Reduce(verifications, func(acc int64, v schema.KeyVerification) int64 {
				if v.IdentityID == identityID {
					acc += v.SpentCredits
				}
				return acc
			}, int64(0))

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				t.Run(table, func(t *testing.T) {
					t.Parallel()
					var queried int64
					err := conn.QueryRow(ctx, fmt.Sprintf("SELECT sum(spent_credits) FROM %s WHERE workspace_id = ? AND identity_id = ?;", table), workspaceID, identityID).Scan(&queried)
					require.NoError(t, err)

					require.Equal(t, credits, queried)
				})
			}

		}
	})

	t.Run("credits spent per key are correct", func(t *testing.T) {
		t.Parallel()
		for _, keyID := range keys[:10] {
			credits := array.Reduce(verifications, func(acc int64, v schema.KeyVerification) int64 {
				if v.KeyID == keyID {
					acc += v.SpentCredits
				}
				return acc
			}, int64(0))

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				t.Run(table, func(t *testing.T) {
					t.Parallel()
					var queried int64
					err := conn.QueryRow(ctx, fmt.Sprintf("SELECT sum(spent_credits) FROM %s WHERE workspace_id = ? AND key_id = ?;", table), workspaceID, keyID).Scan(&queried)
					require.NoError(t, err)

					require.Equal(t, credits, queried)
				})
			}

		}
	})

	t.Run("external_id is stored correctly per identity", func(t *testing.T) {
		t.Parallel()
		// Test that external_id is correctly stored for each identity across all aggregation tables
		for _, identityID := range identities[:10] {
			id := identityID
			expectedExternalID := identityToExternalID[id]

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				tbl := table
				t.Run(tbl, func(t *testing.T) {
					t.Parallel()
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						var queriedExternalID string
						err := conn.QueryRow(ctx, fmt.Sprintf("SELECT external_id FROM %s WHERE workspace_id = ? AND identity_id = ? LIMIT 1;", tbl), workspaceID, id).Scan(&queriedExternalID)
						require.NoError(c, err)
						require.Equal(c, expectedExternalID, queriedExternalID, "external_id should match for identity %s in table %s", id, tbl)
					}, time.Minute, time.Second)
				})
			}
		}
	})

	t.Run("external_id filtering works in all tables", func(t *testing.T) {
		t.Parallel()
		// Pick a random identity and verify filtering by external_id returns correct count
		for _, identityID := range identities[:10] {
			id := identityID
			extID := identityToExternalID[id]
			expectedCount := array.Reduce(verifications, func(acc int, v schema.KeyVerification) int {
				if v.ExternalID == extID {
					acc++
				}
				return acc
			}, 0)

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				tbl := table
				t.Run(tbl, func(t *testing.T) {
					t.Parallel()
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						var queriedCount int64
						err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(count) FROM %s WHERE workspace_id = ? AND external_id = ?;", tbl), workspaceID, extID).Scan(&queriedCount)
						require.NoError(c, err)
						require.Equal(c, expectedCount, int(queriedCount), "count should match for external_id %s in table %s", extID, tbl)
					}, time.Minute, time.Second)
				})
			}
		}
	})

	t.Run("external_id + outcome combinations are correct", func(t *testing.T) {
		t.Parallel()
		// Test that we can group by external_id and outcome correctly
		for _, identityID := range identities[:10] {
			id := identityID
			extID := identityToExternalID[id]

			countByOutcome := array.Reduce(verifications, func(acc map[string]int, v schema.KeyVerification) map[string]int {
				if v.ExternalID == extID {
					acc[v.Outcome]++
				}
				return acc
			}, map[string]int{})

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				tbl := table
				t.Run(tbl, func(t *testing.T) {
					t.Parallel()
					for outcome, expectedCount := range countByOutcome {
						out := outcome
						expCount := expectedCount
						require.EventuallyWithT(t, func(c *assert.CollectT) {
							var queriedCount int64
							err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(count) FROM %s WHERE workspace_id = ? AND external_id = ? AND outcome = ?;", tbl), workspaceID, extID, out).Scan(&queriedCount)
							require.NoError(c, err)
							require.Equal(c, expCount, int(queriedCount), "count for external_id %s and outcome %s should match in table %s", extID, out, tbl)
						}, time.Minute, time.Second)
					}
				})
			}
		}
	})

	t.Run("external_id and identity_id are consistently mapped", func(t *testing.T) {
		t.Parallel()
		// Verify that each external_id is always paired with the same identity_id
		for _, identityID := range identities[:10] {
			id := identityID
			extID := identityToExternalID[id]

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				tbl := table
				t.Run(tbl, func(t *testing.T) {
					t.Parallel()
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						// Count should be zero when querying for this external_id with a different identity_id
						var countWithWrongIdentity int64
						wrongIdentityID := identities[(slices.Index(identities, id)+1)%len(identities)]
						if wrongIdentityID == id {
							wrongIdentityID = identities[(slices.Index(identities, id)+2)%len(identities)]
						}

						err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(count) FROM %s WHERE workspace_id = ? AND external_id = ? AND identity_id = ?;", tbl), workspaceID, extID, wrongIdentityID).Scan(&countWithWrongIdentity)
						if err != nil {
							// It's OK if there are no rows, that means the mapping is correct
							if errors.Is(err, sql.ErrNoRows) {
								countWithWrongIdentity = 0
							} else {
								require.NoError(c, err, "unexpected error querying for wrong identity mapping in table %s", tbl)
							}
						}
						require.Equal(c, int64(0), countWithWrongIdentity, "external_id %s should never be paired with identity_id %s in table %s", extID, wrongIdentityID, tbl)
					}, time.Minute, time.Second)
				})
			}
		}
	})

	t.Run("credits spent per external_id are correct", func(t *testing.T) {
		t.Parallel()
		for _, identityID := range identities[:10] {
			id := identityID
			extID := identityToExternalID[id]
			expectedCredits := array.Reduce(verifications, func(acc int64, v schema.KeyVerification) int64 {
				if v.ExternalID == extID {
					acc += v.SpentCredits
				}
				return acc
			}, int64(0))

			for _, table := range []string{"default.key_verifications_per_minute_v3", "default.key_verifications_per_hour_v3", "default.key_verifications_per_day_v3", "default.key_verifications_per_month_v3"} {
				tbl := table
				t.Run(tbl, func(t *testing.T) {
					t.Parallel()
					require.EventuallyWithT(t, func(c *assert.CollectT) {
						var queriedCredits int64
						err := conn.QueryRow(ctx, fmt.Sprintf("SELECT SUM(spent_credits) FROM %s WHERE workspace_id = ? AND external_id = ?;", tbl), workspaceID, extID).Scan(&queriedCredits)
						require.NoError(c, err)
						require.Equal(c, expectedCredits, queriedCredits, "spent_credits for external_id %s should match in table %s", extID, tbl)
					}, time.Minute, time.Second)
				})
			}
		}
	})

	t.Run("billing per workspace is correct", func(t *testing.T) {
		t.Parallel()
		billableVerifications := array.Reduce(verifications, func(acc int64, v schema.KeyVerification) int64 {
			if v.Outcome == "VALID" {
				acc += 1
			}
			return acc
		}, int64(0))

		var queried int64
		err := conn.QueryRow(ctx, "SELECT sum(count) FROM default.billable_verifications_per_month_v2 WHERE workspace_id = ?;", workspaceID).Scan(&queried)

		require.NoError(t, err)

		require.Equal(t, billableVerifications, queried)
	})
}
