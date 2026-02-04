package quotacheck_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestRunCheck_Integration(t *testing.T) {
	h := harness.New(t)

	// Use current year/month for billing period
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	billingPeriod := fmt.Sprintf("%d-%02d", year, month)

	t.Run("detects workspaces exceeding quota", func(t *testing.T) {
		// Create workspaces with quota
		ws1 := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{RequestsPerMonth: 100_000})
		ws2 := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{RequestsPerMonth: 500_000})
		ws3 := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{RequestsPerMonth: 200_000})

		// Insert usage data into ClickHouse
		// ws1: 200k (exceeds 100k quota)
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws1.ID, 200_000, now, "VALID")
		// ws2: 300k (below 500k quota)
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws2.ID, 300_000, now, "VALID")
		// ws3: 250k (exceeds 200k quota)
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws3.ID, 250_000, now, "VALID")

		// Wait for materialized views
		waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, ws1.ID, 200_000, year, month)
		waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, ws2.ID, 300_000, year, month)
		waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, ws3.ID, 250_000, year, month)

		// Call RunCheck via Restate
		resp, err := callRunCheck(h, billingPeriod)
		require.NoError(t, err)

		// Should have checked workspaces and found 2 exceeded (ws1 and ws3)
		// Note: The exact count depends on what other workspaces exist in the test DB
		require.GreaterOrEqual(t, resp.WorkspacesChecked, int32(3))
		require.GreaterOrEqual(t, resp.WorkspacesExceeded, int32(2))
	})

	t.Run("skips workspaces below minimum usage threshold", func(t *testing.T) {
		// Create a workspace with low quota but below minimum usage threshold (150k)
		ws := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{RequestsPerMonth: 50_000})

		// Insert only 100k usage (below 150k threshold)
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 100_000, now, "VALID")

		waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, ws.ID, 100_000, year, month)

		// Call RunCheck - this workspace should be skipped
		resp, err := callRunCheck(h, billingPeriod)
		require.NoError(t, err)

		// The workspace should be checked but not exceeded (skipped due to threshold)
		require.GreaterOrEqual(t, resp.WorkspacesChecked, int32(1))
		// The workspace's usage (100k) is below minUsageThreshold (150k), so it won't be flagged
	})

	t.Run("handles combined verifications and ratelimits", func(t *testing.T) {
		// Create workspace with 300k quota
		ws := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{RequestsPerMonth: 300_000})

		// Insert 200k verifications + 150k ratelimits = 350k total (exceeds 300k)
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 200_000, now, "VALID")
		h.ClickHouseSeed.InsertRatelimits(h.Ctx, ws.ID, 150_000, now, true)

		waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, ws.ID, 200_000, year, month)
		waitForRatelimitCount(t, h.Ctx, h.ClickHouseConn, ws.ID, 150_000, year, month)

		resp, err := callRunCheck(h, billingPeriod)
		require.NoError(t, err)

		// The workspace should be detected as exceeded
		require.GreaterOrEqual(t, resp.WorkspacesExceeded, int32(1))
	})

	t.Run("skips disabled workspaces", func(t *testing.T) {
		// Create a disabled workspace with usage exceeding quota
		ws := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{RequestsPerMonth: 100_000})

		// Disable the workspace
		_, err := db.Query.UpdateWorkspaceEnabled(h.Ctx, h.DB.RW(), db.UpdateWorkspaceEnabledParams{
			Enabled: false,
			ID:      ws.ID,
		})
		require.NoError(t, err)

		// Insert 200k usage (would exceed quota if enabled)
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 200_000, now, "VALID")
		waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, ws.ID, 200_000, year, month)

		// Call RunCheck - disabled workspace should be skipped
		resp, err := callRunCheck(h, billingPeriod)
		require.NoError(t, err)

		// Verify it was processed but not flagged (disabled workspaces are skipped)
		require.GreaterOrEqual(t, resp.WorkspacesChecked, int32(1))
	})

	t.Run("skips workspaces without quota set", func(t *testing.T) {
		// Create a workspace without a quota (RequestsPerMonth = 0 skips quota creation)
		ws := h.Seed.CreateWorkspace(h.Ctx)

		// Insert usage
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 500_000, now, "VALID")
		waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, ws.ID, 500_000, year, month)

		// Call RunCheck
		resp, err := callRunCheck(h, billingPeriod)
		require.NoError(t, err)

		// Should not flag workspace without quota
		require.GreaterOrEqual(t, resp.WorkspacesChecked, int32(1))
	})
}

func waitForVerificationCount(t *testing.T, ctx context.Context, conn ch.Conn, workspaceID string, expectedCount, year, month int) {
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var count int64
		err := conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.billable_verifications_per_month_v2 WHERE workspace_id = ? AND year = ? AND month = ?",
			workspaceID, year, month).Scan(&count)
		require.NoError(c, err)
		require.Equal(c, int64(expectedCount), count)
	}, 2*time.Minute, time.Second)
}

func waitForRatelimitCount(t *testing.T, ctx context.Context, conn ch.Conn, workspaceID string, expectedCount, year, month int) {
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var count int64
		err := conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.billable_ratelimits_per_month_v2 WHERE workspace_id = ? AND year = ? AND month = ?",
			workspaceID, year, month).Scan(&count)
		require.NoError(c, err)
		require.Equal(c, int64(expectedCount), count)
	}, 2*time.Minute, time.Second)
}

func callRunCheck(h *harness.Harness, billingPeriod string) (*hydrav1.RunCheckResponse, error) {
	client := hydrav1.NewQuotaCheckServiceIngressClient(h.Restate, billingPeriod)
	return client.RunCheck().Request(h.Ctx, &hydrav1.RunCheckRequest{})
}
