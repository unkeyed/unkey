package handler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// TestPortalSessionAnalyticsRejectsOversizedWindow verifies the query window is
// bounded by the workspace's log retention. This endpoint runs on the shared
// ClickHouse connection, so an unbounded window would let an end user force an
// arbitrarily large scan.
func TestPortalSessionAnalyticsRejectsOversizedWindow(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	// SetupAnalytics defaults LogsRetentionDays to 30.
	h.SetupAnalytics(workspace.ID)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{api.KeyAuthID.String}, []string{"analytics:read"})

	now := time.Now().UnixMilli()
	dayMs := int64(24 * time.Hour / time.Millisecond)

	// 40-day window against a 30-day retention must be rejected.
	req := Request{
		StartTime: now - 40*dayMs,
		EndTime:   now,
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "window wider than retention must be rejected")

	// A window within retention is accepted.
	ok := Request{
		StartTime: now - 10*dayMs,
		EndTime:   now,
	}
	okRes := testutil.CallRoute[Request, Response](h, route, headers, ok)
	require.Equal(t, 200, okRes.Status, "window within retention must be accepted")
}
