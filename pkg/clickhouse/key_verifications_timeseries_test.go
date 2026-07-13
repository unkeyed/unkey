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
	targetKey := uid.New(uid.KeyPrefix)
	otherKey := uid.New(uid.KeyPrefix)

	// Anchor inside per_day retention (365d) and in the past. Day granularity is
	// selected for windows > 4 days, so use a 10-day window over two active days.
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(-10 * 24 * time.Hour)
	dayA := base.Add(12 * time.Hour)
	dayB := base.Add(2*24*time.Hour + 12*time.Hour)

	mk := func(extID, keyID, outcome string, ts time.Time) schema.KeyVerification {
		return schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        ts.UnixMilli(),
			WorkspaceID: workspaceID,
			IdentityID:  uid.New(uid.IdentityPrefix),
			ExternalID:  extID,
			KeySpaceID:  keySpaceID,
			Outcome:     outcome,
			Region:      "test",
			Tags:        []string{},
			KeyID:       keyID,
		}
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
	}

	batch, err := client.Conn().PrepareBatch(ctx, clickhouse.InsertQuery[schema.KeyVerification]())
	require.NoError(t, err)
	for i := range rows {
		require.NoError(t, batch.AppendStruct(&rows[i]))
	}
	require.NoError(t, batch.Send())

	// Wait for the per_day materialized view to catch up (extA has 6 events).
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var got int64
		err := client.Conn().QueryRow(ctx,
			"SELECT SUM(count) FROM default.key_verifications_per_day_v3 WHERE workspace_id = ? AND external_id = ?",
			workspaceID, extA,
		).Scan(&got)
		assert.NoError(c, err)
		assert.Equal(c, int64(6), got)
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

		// extB's 5 events on day A must not leak into extA's totals.
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
}
