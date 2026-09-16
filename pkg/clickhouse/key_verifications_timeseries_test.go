package clickhouse_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// sumByTime collapses per-key series into the account-wide view a caller
// reconstructs, keyed by bucket start.
func sumByTime(series []clickhouse.VerificationTimeseriesPerKey) map[int64]clickhouse.VerificationTimeseriesDataPoint {
	summed := make(map[int64]clickhouse.VerificationTimeseriesDataPoint)
	for _, s := range series {
		for _, p := range s.Data {
			agg := summed[p.Time]
			agg.Time = p.Time
			agg.Total += p.Total
			agg.Valid += p.Valid
			agg.RateLimited += p.RateLimited
			agg.InsufficientPermissions += p.InsufficientPermissions
			agg.Forbidden += p.Forbidden
			agg.Disabled += p.Disabled
			agg.Expired += p.Expired
			agg.UsageExceeded += p.UsageExceeded
			summed[p.Time] = agg
		}
	}
	return summed
}

// grandTotal is the account-wide total across every key and bucket.
func grandTotal(series []clickhouse.VerificationTimeseriesPerKey) int64 {
	var total int64
	for _, p := range sumByTime(series) {
		total += p.Total
	}
	return total
}

func TestGetVerificationsByExternalIDPerKey(t *testing.T) {
	t.Parallel()

	chCfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: chCfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	require.NoError(t, client.Ping(ctx))

	workspaceID := uid.New(uid.WorkspacePrefix)
	extA := "ext_" + uid.New("")
	extB := "ext_" + uid.New("")
	keySpaceID := uid.New(uid.KeySpacePrefix)
	otherKeySpaceID := uid.New(uid.KeySpacePrefix)
	targetKey := uid.New(uid.KeyPrefix)
	otherKey := uid.New(uid.KeyPrefix)
	outOfScopeKey := uid.New(uid.KeyPrefix)

	// Anchor inside per_day retention (365d) and in the past. Day granularity is
	// selected for windows > 4 days, so use a 10-day window over two active days.
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(-10 * 24 * time.Hour)
	dayA := base.Add(12 * time.Hour)
	dayB := base.Add(2*24*time.Hour + 12*time.Hour)

	verificationIn := func(keySpace, extID, keyID, outcome string, ts time.Time) schema.KeyVerification {
		return schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        ts.UnixMilli(),
			WorkspaceID: workspaceID,
			IdentityID:  uid.New(uid.IdentityPrefix),
			ExternalID:  extID,
			KeySpaceID:  keySpace,
			Outcome:     outcome,
			Region:      "test",
			Tags:        []string{},
			KeyID:       keyID,
		}
	}

	verification := func(extID, keyID, outcome string, ts time.Time) schema.KeyVerification {
		return verificationIn(keySpaceID, extID, keyID, outcome, ts)
	}

	rows := []schema.KeyVerification{
		// extA, day A: 3 VALID (one on targetKey), 2 RATE_LIMITED
		verification(extA, targetKey, "VALID", dayA),
		verification(extA, otherKey, "VALID", dayA),
		verification(extA, otherKey, "VALID", dayA),
		verification(extA, otherKey, "RATE_LIMITED", dayA),
		verification(extA, otherKey, "RATE_LIMITED", dayA),
		// extA, day B: 1 VALID
		verification(extA, otherKey, "VALID", dayB),
		// extB, day A: 5 VALID (must NOT appear in extA results)
		verification(extB, otherKey, "VALID", dayA),
		verification(extB, otherKey, "VALID", dayA),
		verification(extB, otherKey, "VALID", dayA),
		verification(extB, otherKey, "VALID", dayA),
		verification(extB, otherKey, "VALID", dayA),
		// extA in a second keyspace: visible only to a session scoped to it.
		verificationIn(otherKeySpaceID, extA, outOfScopeKey, "VALID", dayA),
		verificationIn(otherKeySpaceID, extA, outOfScopeKey, "VALID", dayB),
	}

	batch, err := client.Conn().PrepareBatch(ctx, clickhouse.InsertQuery[schema.KeyVerification]())
	require.NoError(t, err)
	for i := range rows {
		require.NoError(t, batch.AppendStruct(&rows[i]))
	}
	require.NoError(t, batch.Send())

	// Wait for the per_day materialized view to catch up (extA has 8 events
	// across both keyspaces).
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var got int64
		err := client.Conn().QueryRow(ctx,
			"SELECT SUM(count) FROM default.key_verifications_per_day_v3 WHERE workspace_id = ? AND external_id = ?",
			workspaceID, extA,
		).Scan(&got)
		assert.NoError(c, err)
		assert.Equal(c, int64(8), got)
	}, time.Minute, time.Second)

	startMs := base.UnixMilli()
	endMs := base.Add(10 * 24 * time.Hour).UnixMilli()
	dayABucket := dayA.Truncate(24 * time.Hour).UnixMilli()
	dayBBucket := dayB.Truncate(24 * time.Hour).UnixMilli()

	read := func(t *testing.T, keySpaceIDs []string, keyID string, maxKeys int) []clickhouse.VerificationTimeseriesPerKey {
		t.Helper()
		series, err := client.GetVerificationsByExternalIDPerKey(ctx, clickhouse.VerificationTimeseriesPerKeyRequest{
			VerificationTimeseriesRequest: clickhouse.VerificationTimeseriesRequest{
				WorkspaceID: workspaceID,
				ExternalID:  extA,
				KeySpaceIDs: keySpaceIDs,
				KeyID:       keyID,
				StartTime:   startMs,
				EndTime:     endMs,
			},
			MaxKeys: maxKeys,
		})
		require.NoError(t, err)
		return series
	}

	t.Run("scoped to external id with correct per-outcome counts", func(t *testing.T) {
		series := read(t, []string{keySpaceID}, "", 10)

		m := sumByTime(series)
		require.Equal(t, int64(5), m[dayABucket].Total)
		require.Equal(t, int64(3), m[dayABucket].Valid)
		require.Equal(t, int64(2), m[dayABucket].RateLimited)
		require.Equal(t, int64(1), m[dayBBucket].Total)
		require.Equal(t, int64(1), m[dayBBucket].Valid)

		// extB's 5 events on day A and extA's 2 events in the other keyspace must
		// not leak into the scoped totals.
		require.Equal(t, int64(6), grandTotal(series))
	})

	t.Run("zero-filled contiguous daily buckets", func(t *testing.T) {
		series := read(t, []string{keySpaceID}, "", 10)

		stepMs := clickhouse.VerificationBucketMillis(startMs, endMs)
		require.Equal(t, int64(24*60*60*1000), stepMs, "a 10-day window selects day buckets")

		// 10-day window -> at least 10 contiguous day buckets, evenly spaced.
		for _, s := range series {
			require.GreaterOrEqual(t, len(s.Data), 10)
			for i := 1; i < len(s.Data); i++ {
				require.Equal(t, stepMs, s.Data[i].Time-s.Data[i-1].Time)
			}
		}
	})

	t.Run("key id filter narrows to one key", func(t *testing.T) {
		series := read(t, []string{keySpaceID}, targetKey, 10)

		m := sumByTime(series)
		require.Equal(t, int64(1), m[dayABucket].Total)
		require.Equal(t, int64(1), m[dayABucket].Valid)
		require.Equal(t, int64(0), m[dayBBucket].Total)
	})

	t.Run("scoped to the session's keyspaces", func(t *testing.T) {
		series := read(t, []string{otherKeySpaceID}, "", 10)

		m := sumByTime(series)
		require.Equal(t, int64(1), m[dayABucket].Total)
		require.Equal(t, int64(1), m[dayBBucket].Total)
		require.Equal(t, int64(2), grandTotal(series))
	})

	t.Run("multiple keyspaces sum", func(t *testing.T) {
		series := read(t, []string{keySpaceID, otherKeySpaceID}, "", 10)
		require.Equal(t, int64(8), grandTotal(series))
	})

	t.Run("per key breakout totals", func(t *testing.T) {
		series := read(t, []string{keySpaceID}, "", 10)

		totals := make(map[string]int64, len(series))
		for _, s := range series {
			for _, p := range s.Data {
				totals[s.KeyID] += p.Total
			}
		}

		require.Len(t, series, 2)
		require.Equal(t, int64(1), totals[targetKey])
		require.Equal(t, int64(5), totals[otherKey])
		require.Equal(t, int64(6), grandTotal(series))
		require.NotContains(t, totals, outOfScopeKey, "a key outside the session keyspaces must not appear")
	})

	t.Run("per key series are zero-filled", func(t *testing.T) {
		series := read(t, []string{keySpaceID}, targetKey, 10)

		require.Len(t, series, 1)
		require.Equal(t, targetKey, series[0].KeyID)

		// Contiguous across the whole window, so the page can chart a narrowed
		// selection without filling the gaps itself.
		require.GreaterOrEqual(t, len(series[0].Data), 10, "the series covers the requested window")
		const dayMs = int64(24 * 60 * 60 * 1000)
		for i := 1; i < len(series[0].Data); i++ {
			require.Equal(t, dayMs, series[0].Data[i].Time-series[0].Data[i-1].Time,
				"buckets are evenly spaced")
		}

		byBucket := make(map[int64]int64, len(series[0].Data))
		for _, p := range series[0].Data {
			byBucket[p.Time] = p.Total
		}
		require.Equal(t, int64(1), byBucket[dayABucket], "the bucket with traffic keeps its count")
		require.Equal(t, int64(0), byBucket[dayBBucket], "a quiet bucket is present with zero")
	})

	t.Run("per key breakout is capped rather than truncated", func(t *testing.T) {
		req := clickhouse.VerificationTimeseriesPerKeyRequest{
			VerificationTimeseriesRequest: clickhouse.VerificationTimeseriesRequest{
				WorkspaceID: workspaceID,
				ExternalID:  extA,
				KeySpaceIDs: []string{keySpaceID},
				KeyID:       "",
				StartTime:   startMs,
				EndTime:     endMs,
			},
			MaxKeys: 1,
		}

		_, err := client.GetVerificationsByExternalIDPerKey(ctx, req)
		require.ErrorIs(t, err, clickhouse.ErrTooManyVerificationKeys)

		req.MaxKeys = 2
		series, err := client.GetVerificationsByExternalIDPerKey(ctx, req)
		require.NoError(t, err)
		require.Len(t, series, 2, "a request exactly at the cap is served")
	})

	t.Run("empty keyspace list is an error", func(t *testing.T) {
		_, err := client.GetVerificationsByExternalIDPerKey(ctx, clickhouse.VerificationTimeseriesPerKeyRequest{
			VerificationTimeseriesRequest: clickhouse.VerificationTimeseriesRequest{
				WorkspaceID: workspaceID,
				ExternalID:  extA,
				KeySpaceIDs: nil,
				KeyID:       "",
				StartTime:   startMs,
				EndTime:     endMs,
			},
			MaxKeys: 10,
		})
		require.Error(t, err)
	})

	// An unset cap means a caller wired the handler without one; serving it
	// would put an unbounded per-key read on the shared connection.
	t.Run("missing key cap is an error", func(t *testing.T) {
		for _, maxKeys := range []int{0, -1} {
			_, err := client.GetVerificationsByExternalIDPerKey(ctx, clickhouse.VerificationTimeseriesPerKeyRequest{
				VerificationTimeseriesRequest: clickhouse.VerificationTimeseriesRequest{
					WorkspaceID: workspaceID,
					ExternalID:  extA,
					KeySpaceIDs: []string{keySpaceID},
					KeyID:       "",
					StartTime:   startMs,
					EndTime:     endMs,
				},
				MaxKeys: maxKeys,
			})
			require.Error(t, err, "MaxKeys %d must be refused", maxKeys)
		}
	})
}
