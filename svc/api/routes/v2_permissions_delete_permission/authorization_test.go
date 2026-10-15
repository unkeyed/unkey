package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_permission"
)

func TestDeletePermissionAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspace.ID,
		Name:        "Legacy",
		Slug:        "legacy",
	})
	canonicalPermission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspace.ID,
		Name:        "Canonical",
		Slug:        "canonical",
	})
	otherWorkspace := h.CreateWorkspace()

	canonicalGrant := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/rbac/permissions/%s#delete",
		workspace.ID, canonicalPermission.ProjectID, canonicalPermission.ID,
	)
	request := handler.Request{Permission: permission.ID}
	notFound := openapi.BaseError{
		Status: http.StatusNotFound,
		Title:  "Not Found",
		Type:   "https://unkey.com/docs/errors/unkey/data/permission_not_found",
		Detail: "The requested permission does not exist.",
	}

	testutil.CheckAuthorization(t, h, route, testutil.Authorization[handler.Request, handler.Response]{
		Denied: []testutil.DeniedRequest[handler.Request]{
			{
				Name: "wrong action",
				Grant: testutil.Grant{
					WorkspaceID: workspace.ID,
					Permissions: []string{"rbac.*.read_permission"},
				},
				Request: request,
				Error:   notFound,
			},
			{
				Name: "wrong resource",
				Grant: testutil.Grant{
					WorkspaceID: workspace.ID,
					Permissions: []string{canonicalGrant},
				},
				Request: request,
				Error:   notFound,
			},
			{
				Name: "other workspace",
				Grant: testutil.Grant{
					WorkspaceID: otherWorkspace.ID,
					Permissions: []string{"rbac.*.delete_permission"},
				},
				Request: request,
				Error:   notFound,
			},
		},
		Allowed: []testutil.AllowedRequest[handler.Request]{
			{
				Name: "legacy wildcard",
				Grant: testutil.Grant{
					WorkspaceID: workspace.ID,
					Permissions: []string{"rbac.*.delete_permission"},
				},
				Request: request,
				Status:  http.StatusOK,
			},
			{
				Name: "canonical exact permission",
				Grant: testutil.Grant{
					WorkspaceID: workspace.ID,
					Permissions: []string{canonicalGrant},
				},
				Request: handler.Request{Permission: canonicalPermission.ID},
				Status:  http.StatusOK,
			},
		},
		VerifyDenied: func(t *testing.T, req handler.Request) {
			t.Helper()

			stored, err := db.Query.FindPermissionByID(t.Context(), h.DB.RO(), req.Permission)
			require.NoError(t, err)
			require.Equal(t, permission.ID, stored.ID)
			require.Empty(t, h.FindAuditLogsByTargetID(t.Context(), t, req.Permission))
		},
		VerifyAllowed: func(t *testing.T, req handler.Request, res testutil.TestResponse[handler.Response]) {
			t.Helper()

			require.NotEmpty(t, res.Body.Meta.RequestId)

			_, err := db.Query.FindPermissionByID(t.Context(), h.DB.RO(), req.Permission)
			require.True(t, db.IsNotFound(err), "deleted permission must be absent, got %v", err)
			require.Len(t, h.FindAuditLogsByTargetID(t.Context(), t, req.Permission), 1)
		},
	})
}
