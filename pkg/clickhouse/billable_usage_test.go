package clickhouse_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestGetBillableUsageAboveThreshold(t *testing.T) {
	dsn := containers.ClickHouse(t)

	// Create ClickHouse client using the clickhouse package
	client, err := clickhouse.New(clickhouse.Config{
		URL:    dsn,
		Logger: logging.NewNoop(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	err = client.Ping(context.Background())
	require.NoError(t, err)

	// Also get a direct connection for inserting test data
	opts, err := ch.ParseDSN(dsn)
	require.NoError(t, err)

	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()

	// Use current year/month for testing
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	t.Run("multiple workspaces with both verifications and ratelimits", func(t *testing.T) {
		// Create test workspaces
		workspace1 := uid.New(uid.WorkspacePrefix)
		workspace2 := uid.New(uid.WorkspacePrefix)
		workspace3 := uid.New(uid.WorkspacePrefix)

		// Insert verifications
		// Workspace 1: 100 VALID verifications
		verifications := createVerifications(workspace1, 100, now, "VALID")
		verifications = append(verifications, createVerifications(workspace2, 50, now, "VALID")...)
		verifications = append(verifications, createVerifications(workspace3, 200, now, "VALID")...)

		insertVerifications(t, ctx, conn, verifications)

		// Insert ratelimits
		// Workspace 1: 30 passed ratelimits
		ratelimits := createRatelimits(workspace1, 30, now, true)
		ratelimits = append(ratelimits, createRatelimits(workspace2, 70, now, true)...)
		// No ratelimits for workspace3

		insertRatelimits(t, ctx, conn, ratelimits)

		// Wait for materialized views to process
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			usage, err := client.GetBillableUsageAboveThreshold(ctx, year, month, 0)
			require.NoError(c, err)

			// Workspace1: 100 verifications + 30 ratelimits = 130
			assert.Equal(c, int64(130), usage[workspace1], "workspace1 usage mismatch")
			// Workspace2: 50 verifications + 70 ratelimits = 120
			assert.Equal(c, int64(120), usage[workspace2], "workspace2 usage mismatch")
			// Workspace3: 200 verifications + 0 ratelimits = 200
			assert.Equal(c, int64(200), usage[workspace3], "workspace3 usage mismatch")
		}, time.Minute, time.Second)
	})

	t.Run("workspace with only verifications", func(t *testing.T) {
		workspace := uid.New(uid.WorkspacePrefix)
		verifications := createVerifications(workspace, 75, now, "VALID")
		insertVerifications(t, ctx, conn, verifications)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			usage, err := client.GetBillableUsageAboveThreshold(ctx, year, month, 0)
			require.NoError(c, err)
			assert.Equal(c, int64(75), usage[workspace], "workspace should have 75 verifications only")
		}, time.Minute, time.Second)
	})

	t.Run("workspace with only ratelimits", func(t *testing.T) {
		workspace := uid.New(uid.WorkspacePrefix)
		ratelimits := createRatelimits(workspace, 42, now, true)
		insertRatelimits(t, ctx, conn, ratelimits)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			usage, err := client.GetBillableUsageAboveThreshold(ctx, year, month, 0)
			require.NoError(c, err)
			assert.Equal(c, int64(42), usage[workspace], "workspace should have 42 ratelimits only")
		}, time.Minute, time.Second)
	})

	t.Run("empty result for nonexistent year/month", func(t *testing.T) {
		// Query for a year/month in the far past where we shouldn't have any data
		usage, err := client.GetBillableUsageAboveThreshold(ctx, 1999, 1, 0)
		require.NoError(t, err)
		require.Empty(t, usage, "should return empty map for nonexistent data")
	})

	t.Run("threshold filters out workspaces below minimum", func(t *testing.T) {
		// Create workspaces with different usage levels
		wsLow := uid.New(uid.WorkspacePrefix)
		wsMid := uid.New(uid.WorkspacePrefix)
		wsHigh := uid.New(uid.WorkspacePrefix)

		// Low: 50 verifications
		insertVerifications(t, ctx, conn, createVerifications(wsLow, 50, now, "VALID"))
		// Mid: 150 verifications
		insertVerifications(t, ctx, conn, createVerifications(wsMid, 150, now, "VALID"))
		// High: 300 verifications
		insertVerifications(t, ctx, conn, createVerifications(wsHigh, 300, now, "VALID"))

		// Wait for data to be available
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			usage, err := client.GetBillableUsageAboveThreshold(ctx, year, month, 0)
			require.NoError(c, err)
			assert.Equal(c, int64(50), usage[wsLow])
			assert.Equal(c, int64(150), usage[wsMid])
			assert.Equal(c, int64(300), usage[wsHigh])
		}, time.Minute, time.Second)

		// Now query with threshold of 100 - should exclude wsLow
		usage, err := client.GetBillableUsageAboveThreshold(ctx, year, month, 100)
		require.NoError(t, err)

		_, hasLow := usage[wsLow]
		require.False(t, hasLow, "wsLow should be filtered out by threshold")
		require.Equal(t, int64(150), usage[wsMid], "wsMid should be included")
		require.Equal(t, int64(300), usage[wsHigh], "wsHigh should be included")

		// Query with threshold of 200 - should exclude wsLow and wsMid
		usage, err = client.GetBillableUsageAboveThreshold(ctx, year, month, 200)
		require.NoError(t, err)

		_, hasLow = usage[wsLow]
		_, hasMid := usage[wsMid]
		require.False(t, hasLow, "wsLow should be filtered out")
		require.False(t, hasMid, "wsMid should be filtered out")
		require.Equal(t, int64(300), usage[wsHigh], "wsHigh should be included")
	})

	t.Run("correct summation across multiple time periods within month", func(t *testing.T) {
		workspace := uid.New(uid.WorkspacePrefix)

		// Create verifications at different times within the same month
		startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		midMonth := time.Date(year, time.Month(month), 15, 12, 0, 0, 0, time.UTC)

		insertVerifications(t, ctx, conn, createVerifications(workspace, 25, startOfMonth, "VALID"))
		insertVerifications(t, ctx, conn, createVerifications(workspace, 35, midMonth, "VALID"))

		insertRatelimits(t, ctx, conn, createRatelimits(workspace, 10, startOfMonth, true))
		insertRatelimits(t, ctx, conn, createRatelimits(workspace, 15, midMonth, true))

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			usage, err := client.GetBillableUsageAboveThreshold(ctx, year, month, 0)
			require.NoError(c, err)
			// Total: 25 + 35 + 10 + 15 = 85
			assert.Equal(c, int64(85), usage[workspace], "should sum all events within the month")
		}, time.Minute, time.Second)
	})

	t.Run("only counts VALID verifications", func(t *testing.T) {
		workspace := uid.New(uid.WorkspacePrefix)

		// Create a mix of verification outcomes
		insertVerifications(t, ctx, conn, createVerifications(workspace, 50, now, "VALID"))
		insertVerifications(t, ctx, conn, createVerifications(workspace, 30, now, "INVALID"))
		insertVerifications(t, ctx, conn, createVerifications(workspace, 20, now, "EXPIRED"))

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			usage, err := client.GetBillableUsageAboveThreshold(ctx, year, month, 0)
			require.NoError(c, err)
			// Only VALID verifications should count: 50
			assert.Equal(c, int64(50), usage[workspace], "should only count VALID verifications")
		}, time.Minute, time.Second)
	})

	t.Run("only counts passed ratelimits", func(t *testing.T) {
		workspace := uid.New(uid.WorkspacePrefix)

		// Create passed and failed ratelimits
		insertRatelimits(t, ctx, conn, createRatelimits(workspace, 40, now, true))
		insertRatelimits(t, ctx, conn, createRatelimits(workspace, 25, now, false))

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			usage, err := client.GetBillableUsageAboveThreshold(ctx, year, month, 0)
			require.NoError(c, err)
			// Only passed ratelimits should count: 40
			assert.Equal(c, int64(40), usage[workspace], "should only count passed ratelimits")
		}, time.Minute, time.Second)
	})
}

// Helper functions to create test data

func createVerifications(workspaceID string, count int, timestamp time.Time, outcome string) []schema.KeyVerification {
	verifications := make([]schema.KeyVerification, count)
	for i := range count {
		verifications[i] = schema.KeyVerification{
			RequestID:    uid.New(uid.RequestPrefix),
			Time:         timestamp.Add(time.Duration(i) * time.Second).UnixMilli(),
			WorkspaceID:  workspaceID,
			KeySpaceID:   uid.New(uid.KeySpacePrefix),
			IdentityID:   "",
			ExternalID:   "",
			KeyID:        uid.New(uid.KeyPrefix),
			Region:       "us-east-1",
			Outcome:      outcome,
			Tags:         []string{},
			SpentCredits: 0,
			Latency:      rand.Float64() * 100,
		}
	}
	return verifications
}

func createRatelimits(workspaceID string, count int, timestamp time.Time, passed bool) []schema.Ratelimit {
	ratelimits := make([]schema.Ratelimit, count)
	var remaining uint64 = 50
	if !passed {
		remaining = 0
	}
	for i := range count {
		ratelimits[i] = schema.Ratelimit{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        timestamp.Add(time.Duration(i) * time.Second).UnixMilli(),
			WorkspaceID: workspaceID,
			NamespaceID: uid.New(uid.RatelimitNamespacePrefix),
			Identifier:  uid.New(uid.IdentityPrefix),
			Passed:      passed,
			Latency:     rand.Float64() * 10,
			OverrideID:  "",
			Limit:       100,
			Remaining:   remaining,
			ResetAt:     timestamp.Add(time.Minute).UnixMilli(),
		}
	}
	return ratelimits
}

func insertVerifications(t *testing.T, ctx context.Context, conn ch.Conn, verifications []schema.KeyVerification) {
	if len(verifications) == 0 {
		return
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO default.key_verifications_raw_v2")
	require.NoError(t, err)

	for _, v := range verifications {
		err = batch.AppendStruct(&v)
		require.NoError(t, err)
	}

	err = batch.Send()
	require.NoError(t, err)
}

func insertRatelimits(t *testing.T, ctx context.Context, conn ch.Conn, ratelimits []schema.Ratelimit) {
	if len(ratelimits) == 0 {
		return
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO default.ratelimits_raw_v2")
	require.NoError(t, err)

	for _, r := range ratelimits {
		err = batch.AppendStruct(&r)
		require.NoError(t, err)
	}

	err = batch.Send()
	require.NoError(t, err)
}
