package cron_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestRunQuotaCheck_Integration(t *testing.T) {
	h := harness.New(t)

	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	billingPeriod := fmt.Sprintf("%d-%02d", year, month)

	t.Run("detects workspaces exceeding quota", func(t *testing.T) {
		ws1 := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 100_000})
		ws2 := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 500_000})
		ws3 := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 200_000})

		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws1.ID, 200_000, now, "VALID")
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws2.ID, 300_000, now, "VALID")
		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws3.ID, 250_000, now, "VALID")

		waitForVerificationCount(t, h.Ctx, h.ClickHouse, ws1.ID, 200_000, year, month)
		waitForVerificationCount(t, h.Ctx, h.ClickHouse, ws2.ID, 300_000, year, month)
		waitForVerificationCount(t, h.Ctx, h.ClickHouse, ws3.ID, 250_000, year, month)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesChecked(), int32(3))
		require.GreaterOrEqual(t, resp.GetWorkspacesExceeded(), int32(2))
	})

	t.Run("skips workspaces below minimum usage threshold", func(t *testing.T) {
		ws := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 50_000})

		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 100_000, now, "VALID")
		waitForVerificationCount(t, h.Ctx, h.ClickHouse, ws.ID, 100_000, year, month)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesChecked(), int32(1))
	})

	t.Run("handles combined verifications and ratelimits", func(t *testing.T) {
		ws := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 300_000})

		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 200_000, now, "VALID")
		h.ClickHouseSeed.InsertRatelimits(h.Ctx, ws.ID, 150_000, now, true)

		waitForVerificationCount(t, h.Ctx, h.ClickHouse, ws.ID, 200_000, year, month)
		waitForRatelimitCount(t, h.Ctx, h.ClickHouse, ws.ID, 150_000, year, month)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesExceeded(), int32(1))
	})

	t.Run("skips disabled workspaces", func(t *testing.T) {
		ws := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 100_000})

		_, err := h.DB.UpdateWorkspaceEnabled(h.Ctx, db.UpdateWorkspaceEnabledParams{
			Enabled: false,
			ID:      ws.ID,
		})
		require.NoError(t, err)

		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 200_000, now, "VALID")
		waitForVerificationCount(t, h.Ctx, h.ClickHouse, ws.ID, 200_000, year, month)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesChecked(), int32(1))
	})

	t.Run("skips workspaces without quota set", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)

		h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 500_000, now, "VALID")
		waitForVerificationCount(t, h.Ctx, h.ClickHouse, ws.ID, 500_000, year, month)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesChecked(), int32(1))
	})
}

func waitForVerificationCount(t *testing.T, ctx context.Context, reader clickhouse.ClickHouse, workspaceID string, expectedCount, year, month int) {
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		count, err := reader.GetBillableVerifications(ctx, workspaceID, year, month)
		require.NoError(c, err)
		require.Equal(c, int64(expectedCount), count)
	}, 2*time.Minute, time.Second)
}

func waitForRatelimitCount(t *testing.T, ctx context.Context, reader clickhouse.ClickHouse, workspaceID string, expectedCount, year, month int) {
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		count, err := reader.GetBillableRatelimits(ctx, workspaceID, year, month)
		require.NoError(c, err)
		require.Equal(c, int64(expectedCount), count)
	}, 2*time.Minute, time.Second)
}

func callRunQuotaCheck(h *harness.Harness, billingPeriod string) (*hydrav1.RunQuotaCheckResponse, error) {
	client := hydrav1.NewCronServiceIngressClient(h.Restate, "quota-check-"+billingPeriod)
	return client.RunQuotaCheck().Request(h.Ctx, &hydrav1.RunQuotaCheckRequest{})
}
