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

func TestDeletePermissionRejectsUnknownCredentialsWithoutDeletion(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspace.ID,
		Name:        "Protected permission",
		Slug:        "protected-permission",
	})
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_permission")

	testutil.CheckAuthentication(t, h, route, testutil.Authentication[handler.Request, handler.Response]{
		Request:      handler.Request{Permission: permission.ID},
		ValidHeaders: testutil.Bearer(rootKey),
		Status:       http.StatusOK,
		VerifyDenied: func(t *testing.T) {
			t.Helper()
			stored, err := db.Query.FindPermissionByID(t.Context(), h.DB.RO(), permission.ID)
			require.NoError(t, err)
			require.Equal(t, permission.ID, stored.ID)
			require.Empty(t, h.FindAuditLogsByTargetID(t.Context(), t, permission.ID))
		},
		VerifyAllowed: func(t *testing.T, res testutil.TestResponse[handler.Response]) {
			t.Helper()
			require.NotEmpty(t, res.Body.Meta.RequestId)

			_, err := db.Query.FindPermissionByID(t.Context(), h.DB.RO(), permission.ID)
			require.True(t, db.IsNotFound(err), "deleted permission must be absent, got %v", err)
			require.Len(t, h.FindAuditLogsByTargetID(t.Context(), t, permission.ID), 1)
		},
	})
}
