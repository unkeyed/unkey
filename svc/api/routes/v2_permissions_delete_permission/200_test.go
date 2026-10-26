package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_permission"
)

func TestDeletePermissionRemovesBindingsAndPreservesOtherPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspace.ID,
		Name:        "Permission to remove",
		Slug:        "permission-to-remove",
	})
	unrelated := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspace.ID,
		Name:        "Unrelated permission",
		Slug:        "unrelated-permission",
	})
	role := h.CreateRole(seed.CreateRoleRequest{
		WorkspaceID: workspace.ID,
		Name:        "Member",
	})
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})
	for _, permissionID := range []string{permission.ID, unrelated.ID} {
		err := db.Query.InsertRolePermission(t.Context(), h.DB.RW(), db.InsertRolePermissionParams{
			RoleID:       role.ID,
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			CreatedAtM:   h.Clock.Now().UnixMilli(),
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(t.Context(), h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        key.KeyID,
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    h.Clock.Now().UnixMilli(),
		})
		require.NoError(t, err)
	}

	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_permission")
	res := testutil.CallRoute[handler.Request, handler.Response](
		h, route, testutil.Bearer(rootKey), handler.Request{Permission: permission.ID},
	)
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	_, err := db.Query.FindPermissionByID(t.Context(), h.DB.RO(), permission.ID)
	require.True(t, db.IsNotFound(err), "deleted permission must be absent, got %v", err)

	rolePermissions, err := db.Query.FindRolePermissionByRoleAndPermissionID(t.Context(), h.DB.RO(), db.FindRolePermissionByRoleAndPermissionIDParams{
		RoleID:       role.ID,
		PermissionID: permission.ID,
	})
	require.NoError(t, err)
	require.Empty(t, rolePermissions)

	rolePermissions, err = db.Query.FindRolePermissionByRoleAndPermissionID(t.Context(), h.DB.RO(), db.FindRolePermissionByRoleAndPermissionIDParams{
		RoleID:       role.ID,
		PermissionID: unrelated.ID,
	})
	require.NoError(t, err)
	require.Len(t, rolePermissions, 1)

	keyPermissions, err := db.Query.ListDirectPermissionsByKeyID(t.Context(), h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.Len(t, keyPermissions, 1)
	require.Equal(t, unrelated.ID, keyPermissions[0].ID)

	_, err = db.Query.FindRoleByID(t.Context(), h.DB.RO(), role.ID)
	require.NoError(t, err)
	_, err = db.Query.FindKeyByID(t.Context(), h.DB.RO(), key.KeyID)
	require.NoError(t, err)

	events := h.FindAuditLogsByTargetID(t.Context(), t, permission.ID)
	require.Len(t, events, 1)
	require.Equal(t, "permission.delete", events[0].Event)
}
