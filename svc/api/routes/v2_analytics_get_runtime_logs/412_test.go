package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test412_UnconfiguredAnalytics(t *testing.T) {
	h, route, workspaceID := newRoute(t, false)
	rootKey := h.CreateRootKey(workspaceID, rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*").Deployment("*").Logs(), permissions.Read).Value)

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT count() FROM runtime_logs_v1",
	})
	require.Equal(t, 412, res.Status)
}
