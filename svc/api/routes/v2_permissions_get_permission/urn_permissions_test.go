package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_permission"
)

// TestGetPermissionAuthorizesCanonicalReadPermission guarantees an exact
// project-scoped permission grant can read its permission.
func TestGetPermissionAuthorizesCanonicalReadPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspaceID,
		Name:        "canonical.permission",
		Slug:        "canonical.permission",
	})
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/permissions/%s#read", workspaceID, permission.ProjectID, permission.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Permission: permission.ID})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, permission.ID, res.Body.Data.Id)
}
