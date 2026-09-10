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

func TestGetVerificationsByExternalID(t *testing.T) {
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

	mkIn := func(keySpace, extID, keyID, outcome string, ts time.Time) schema.KeyVerification {
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

	mk := func(extID, keyID, outcome string, ts time.Time) schema.KeyVerification {
		return mkIn(keySpaceID, extID, keyID, outcome, ts)
	}

	rows := []schema.KeyVerification{
		// extA, day A: 3 VALID (one on targetKey), 2 RATE_LIMITED
		mk(extA, targetKey, "VALID", dayA),
		mk(extA, otherKey, "VALID", dayA),
		mk(extA, otherKey, "VALID", dayA),
		mk(extA, otherKey, "RATE_LIMITED", dayA),
		mk(extA, otherKey, "RATE_LIMITED", dayA),
		// extA, day B: 1 VALID
		mk(extA, otherKey, "VALID", dayB),
		// extB, day A: 5 VALID (must NOT appear in extA results)
		mk(extB, otherKey, "VALID", dayA),
		mk(extB, otherKey, "VALID", dayA),
		mk(extB, otherKey, "VALID", dayA),
		mk(extB, otherKey, "VALID", dayA),
		mk(extB, otherKey, "VALID", dayA),
		// extA in a second keyspace: visible only to a session scoped to it.
		mkIn(otherKeySpaceID, extA, outOfScopeKey, "VALID", dayA),
		mkIn(otherKeySpaceID, extA, outOfScopeKey, "VALID", dayB),
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

	byTime := func(points []clickhouse.VerificationTimeseriesDataPoint) map[int64]clickhouse.VerificationTimeseriesDataPoint {
		m := make(map[int64]clickhouse.VerificationTimeseriesDataPoint, len(points))
		for _, p := range points {
			m[p.Time] = p
		}
		return m
	}

	t.Run("scoped to external id with correct per-outcome counts", func(t *testing.T) {
		points, err := client.GetVerificationsByExternalID(ctx, clickhouse.VerificationTimeseriesRequest{
			WorkspaceID: workspaceID,
			ExternalID:  extA,
			KeySpaceIDs: []string{keySpaceID},
			StartTime:   startMs,
			EndTime:     endMs,
		})
		require.NoError(t, err)

		m := byTime(points)
		require.Equal(t, int64(5), m[dayABucket].Total)
		require.Equal(t, int64(3), m[dayABucket].Valid)
		require.Equal(t, int64(2), m[dayABucket].RateLimited)
		require.Equal(t, int64(1), m[dayBBucket].Total)
		require.Equal(t, int64(1), m[dayBBucket].Valid)

		// extB's 5 events on day A and extA's 2 events in the other keyspace must
		// not leak into the scoped totals.
		var grand int64
		for _, p := range points {
			grand += p.Total
		}
		require.Equal(t, int64(6), grand)
	})

	t.Run("zero-filled contiguous daily buckets", func(t *testing.T) {
		points, err := client.GetVerificationsByExternalID(ctx, clickhouse.VerificationTimeseriesRequest{
			WorkspaceID: workspaceID,
			ExternalID:  extA,
			KeySpaceIDs: []string{keySpaceID},
			StartTime:   startMs,
			EndTime:     endMs,
		})
		require.NoError(t, err)

		// 10-day window -> at least 10 contiguous day buckets, evenly spaced.
		require.GreaterOrEqual(t, len(points), 10)
		const dayMs = int64(24 * 60 * 60 * 1000)
		for i := 1; i < len(points); i++ {
			require.Equal(t, dayMs, points[i].Time-points[i-1].Time)
		}
	})

	t.Run("key id filter narrows to one key", func(t *testing.T) {
		points, err := client.GetVerificationsByExternalID(ctx, clickhouse.VerificationTimeseriesRequest{
			WorkspaceID: workspaceID,
			ExternalID:  extA,
			KeySpaceIDs: []string{keySpaceID},
			KeyID:       targetKey,
			StartTime:   startMs,
			EndTime:     endMs,
		})
		require.NoError(t, err)

		m := byTime(points)
		require.Equal(t, int64(1), m[dayABucket].Total)
		require.Equal(t, int64(1), m[dayABucket].Valid)
		require.Equal(t, int64(0), m[dayBBucket].Total)
	})

	t.Run("scoped to the session's keyspaces", func(t *testing.T) {
		points, err := client.GetVerificationsByExternalID(ctx, clickhouse.VerificationTimeseriesRequest{
			WorkspaceID: workspaceID,
			ExternalID:  extA,
			KeySpaceIDs: []string{otherKeySpaceID},
			KeyID:       "",
			StartTime:   startMs,
			EndTime:     endMs,
		})
		require.NoError(t, err)

		m := byTime(points)
		require.Equal(t, int64(1), m[dayABucket].Total)
		require.Equal(t, int64(1), m[dayBBucket].Total)

		var grand int64
		for _, p := range points {
			grand += p.Total
		}
		require.Equal(t, int64(2), grand)
	})

	t.Run("multiple keyspaces sum", func(t *testing.T) {
		points, err := client.GetVerificationsByExternalID(ctx, clickhouse.VerificationTimeseriesRequest{
			WorkspaceID: workspaceID,
			ExternalID:  extA,
			KeySpaceIDs: []string{keySpaceID, otherKeySpaceID},
			KeyID:       "",
			StartTime:   startMs,
			EndTime:     endMs,
		})
		require.NoError(t, err)

		var grand int64
		for _, p := range points {
			grand += p.Total
		}
		require.Equal(t, int64(8), grand)
	})

	t.Run("per key breakout splits the account-wide series", func(t *testing.T) {
		series, err := client.GetVerificationsByExternalIDPerKey(ctx, clickhouse.VerificationTimeseriesPerKeyRequest{
			VerificationTimeseriesRequest: clickhouse.VerificationTimeseriesRequest{
				WorkspaceID: workspaceID,
				ExternalID:  extA,
				KeySpaceIDs: []string{keySpaceID},
				KeyID:       "",
				StartTime:   startMs,
				EndTime:     endMs,
			},
			MaxKeys: 10,
		})
		require.NoError(t, err)

		totals := make(map[string]int64, len(series))
		var grand int64
		for _, s := range series {
			for _, p := range s.Data {
				totals[s.KeyID] += p.Total
				grand += p.Total
			}
		}

		require.Len(t, series, 2)
		require.Equal(t, int64(1), totals[targetKey])
		require.Equal(t, int64(5), totals[otherKey])
		require.Equal(t, int64(6), grand, "per-key totals must sum to the account-wide total")
		require.NotContains(t, totals, outOfScopeKey, "a key outside the session keyspaces must not appear")
	})

	t.Run("per key series are sparse", func(t *testing.T) {
		series, err := client.GetVerificationsByExternalIDPerKey(ctx, clickhouse.VerificationTimeseriesPerKeyRequest{
			VerificationTimeseriesRequest: clickhouse.VerificationTimeseriesRequest{
				WorkspaceID: workspaceID,
				ExternalID:  extA,
				KeySpaceIDs: []string{keySpaceID},
				KeyID:       targetKey,
				StartTime:   startMs,
				EndTime:     endMs,
			},
			MaxKeys: 10,
		})
		require.NoError(t, err)

		require.Len(t, series, 1)
		require.Equal(t, targetKey, series[0].KeyID)
		require.Len(t, series[0].Data, 1, "only the bucket with traffic is returned")
		require.Equal(t, dayABucket, series[0].Data[0].Time)
		require.Equal(t, int64(1), series[0].Data[0].Total)
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

	t.Run("empty keyspace list is an error for the per key read", func(t *testing.T) {
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

	t.Run("empty keyspace list is an error", func(t *testing.T) {
		_, err := client.GetVerificationsByExternalID(ctx, clickhouse.VerificationTimeseriesRequest{
			WorkspaceID: workspaceID,
			ExternalID:  extA,
			KeySpaceIDs: nil,
			KeyID:       "",
			StartTime:   startMs,
			EndTime:     endMs,
		})
		require.Error(t, err)
	})
}
