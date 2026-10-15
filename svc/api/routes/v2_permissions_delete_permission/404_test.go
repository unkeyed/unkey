package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_permission"
)

func TestDeletePermissionNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspace.ID,
		Name:        "Delete me",
		Slug:        "delete-me",
	})
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_permission")
	deleted := testutil.CallRoute[handler.Request, handler.Response](
		h, route, testutil.Bearer(rootKey), handler.Request{Permission: permission.ID},
	)
	require.Equal(t, http.StatusOK, deleted.Status, "%s", deleted.RawBody)
	before := h.FindAuditLogs(t.Context(), t)
	require.Len(t, before, 1)

	for _, tc := range []struct {
		name       string
		permission string
	}{
		{name: "missing ID", permission: uid.New(uid.PermissionPrefix)},
		{name: "missing slug", permission: uid.New(uid.TestPrefix)},
		{name: "already deleted", permission: permission.ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
				h, route, testutil.Bearer(rootKey), handler.Request{Permission: tc.permission},
			)
			require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
			require.Equal(t, openapi.BaseError{
				Status: http.StatusNotFound,
				Title:  "Not Found",
				Type:   "https://unkey.com/docs/errors/unkey/data/permission_not_found",
				Detail: "The requested permission does not exist.",
			}, res.Body.Error)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Equal(t, before, h.FindAuditLogs(t.Context(), t))
		})
	}
}
