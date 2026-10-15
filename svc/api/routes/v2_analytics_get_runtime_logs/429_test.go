package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test429_WorkspaceQueryQuota(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithMaxQueriesPerWindow(1))
	rootKey := h.CreateRootKey(workspace.ID, rbac.U(urn.New().Workspace(workspace.ID).Project("*").App("*").Environment("*").Deployment("*").Logs(), permissions.Read).Value)
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)

	req := Request{Query: "SELECT count() FROM runtime_logs_v1"}
	require.Equal(t, 200, testutil.CallRoute[Request, Response](h, route, auth(rootKey), req).Status)
	require.Equal(t, 429, testutil.CallRoute[Request, Response](h, route, auth(rootKey), req).Status)
}
