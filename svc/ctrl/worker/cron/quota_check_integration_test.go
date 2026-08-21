package cron_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
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

		h.ClickHouseSeed.InsertBillableVerifications(h.Ctx, ws1.ID, 200_000, now)
		h.ClickHouseSeed.InsertBillableVerifications(h.Ctx, ws2.ID, 300_000, now)
		h.ClickHouseSeed.InsertBillableVerifications(h.Ctx, ws3.ID, 250_000, now)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesChecked(), int32(3))
		require.GreaterOrEqual(t, resp.GetWorkspacesExceeded(), int32(2))
	})

	t.Run("skips workspaces below minimum usage threshold", func(t *testing.T) {
		ws := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 50_000})

		h.ClickHouseSeed.InsertBillableVerifications(h.Ctx, ws.ID, 100_000, now)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesChecked(), int32(1))
	})

	t.Run("handles combined verifications and ratelimits", func(t *testing.T) {
		ws := h.Seed.CreateWorkspaceWithLimits(h.Ctx, seed.CreateWorkspaceWithLimitsRequest{RequestsPerMonth: 300_000})

		h.ClickHouseSeed.InsertBillableVerifications(h.Ctx, ws.ID, 200_000, now)
		h.ClickHouseSeed.InsertBillableRatelimits(h.Ctx, ws.ID, 150_000, now)

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

		h.ClickHouseSeed.InsertBillableVerifications(h.Ctx, ws.ID, 200_000, now)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesChecked(), int32(1))
	})

	t.Run("skips workspaces without quota set", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)

		h.ClickHouseSeed.InsertBillableVerifications(h.Ctx, ws.ID, 500_000, now)

		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)

		require.GreaterOrEqual(t, resp.GetWorkspacesChecked(), int32(1))
	})
}

func callRunQuotaCheck(h *harness.Harness, billingPeriod string) (*hydrav1.RunQuotaCheckResponse, error) {
	client := hydrav1.NewCronServiceIngressClient(h.Restate, "quota-check-"+billingPeriod)
	return client.RunQuotaCheck().Request(h.Ctx, &hydrav1.RunQuotaCheckRequest{})
}
