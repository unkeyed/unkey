package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_set_role_permissions"
	"golang.org/x/sync/errgroup"
)

func TestSetRolePermissions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	authorized := h.CreateRootKey(workspace.ID, "rbac.*.add_permission_to_role", "rbac.*.remove_permission_from_role", "rbac.*.create_permission")
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {fmt.Sprintf("Bearer %s", authorized)}}

	t.Run("replace, deduplicate, create, repeat, and clear", func(t *testing.T) {
		role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspace.ID, Name: "set-role-permissions"})
		old := h.CreatePermission(seed.CreatePermissionRequest{WorkspaceID: workspace.ID, Name: "old.permission", Slug: "old.permission"})
		keep := h.CreatePermission(seed.CreatePermissionRequest{WorkspaceID: workspace.ID, Name: "keep.permission", Slug: "keep.permission"})
		for _, permissionID := range []string{old.ID, keep.ID} {
			require.NoError(t, db.Query.InsertRolePermission(ctx, h.DB.RW(), db.InsertRolePermissionParams{RoleID: role.ID, PermissionID: permissionID, WorkspaceID: workspace.ID, CreatedAtM: time.Now().UnixMilli()}))
		}

		req := handler.Request{RoleId: role.ID, Permissions: []string{"KEEP.Permission", "new.permission", "NEW.Permission"}}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, res.RawBody)
		require.Len(t, res.Body.Data, 2)
		assigned, err := db.Query.ListDirectPermissionsByRoleID(ctx, h.DB.RO(), role.ID)
		require.NoError(t, err)
		require.Len(t, assigned, 2)
		require.Equal(t, []string{"keep.permission", "new.permission"}, []string{assigned[0].Slug, assigned[1].Slug})
		logs := h.FindAuditLogsByTargetID(ctx, t, role.ID)
		require.Len(t, logs, 2)
		require.ElementsMatch(t, []string{"authorization.disconnect_role_and_permissions", "authorization.connect_role_and_permission"}, []string{logs[0].Event, logs[1].Event})
		require.NotEmpty(t, logs[0].CorrelationID)
		require.Equal(t, logs[0].CorrelationID, logs[1].CorrelationID)

		repeat := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, repeat.Status, repeat.RawBody)
		require.Len(t, repeat.Body.Data, 2)
		require.Len(t, h.FindAuditLogsByTargetID(ctx, t, role.ID), 2, "an idempotent repeat must not add audit records")

		clear := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{RoleId: role.ID, Permissions: []string{}})
		require.Equal(t, http.StatusOK, clear.Status, clear.RawBody)
		require.Empty(t, clear.Body.Data)
		assigned, err = db.Query.ListDirectPermissionsByRoleID(ctx, h.DB.RO(), role.ID)
		require.NoError(t, err)
		require.Empty(t, assigned)
	})

	t.Run("unknown and foreign roles are not found", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		for _, roleID := range []string{"role_missing", h.CreateRole(seed.CreateRoleRequest{WorkspaceID: otherWorkspace.ID, Name: "foreign-role"}).ID} {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{RoleId: roleID, Permissions: []string{}})
			require.Equal(t, http.StatusNotFound, res.Status, res.RawBody)
		}
	})

	t.Run("requires add, remove, and create authorization", func(t *testing.T) {
		role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspace.ID, Name: "auth-role"})
		cases := [][]string{{"rbac.*.remove_permission_from_role"}, {"rbac.*.add_permission_to_role"}, {"rbac.*.add_permission_to_role", "rbac.*.remove_permission_from_role"}}
		for i, grants := range cases {
			rootKey := h.CreateRootKey(workspace.ID, grants...)
			authHeaders := http.Header{"Content-Type": {"application/json"}, "Authorization": {fmt.Sprintf("Bearer %s", rootKey)}}
			permissions := []string{}
			if i == 2 {
				permissions = []string{"missing.permission"}
			}
			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, authHeaders, handler.Request{RoleId: role.ID, Permissions: permissions})
			require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
		}
	})

	t.Run("malformed requests", func(t *testing.T) {
		for name, req := range map[string]map[string]any{
			"missing roleId":      {"permissions": []string{}},
			"missing permissions": {"roleId": "role_missing"},
			"invalid slug":        {"roleId": "role_missing", "permissions": []string{""}},
		} {
			t.Run(name, func(t *testing.T) {
				res := testutil.CallRoute[map[string]any, openapi.BadRequestErrorResponse](h, route, headers, req)
				require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
			})
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer invalid"}}, handler.Request{RoleId: "role_missing", Permissions: []string{}})
		require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
	})
}

func TestConcurrentMissingPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.add_permission_to_role", "rbac.*.remove_permission_from_role", "rbac.*.create_permission")
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {fmt.Sprintf("Bearer %s", rootKey)}}

	roles := []db.Role{
		h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspace.ID, Name: "concurrent-role-a"}),
		h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspace.ID, Name: "concurrent-role-b"}),
	}

	// Warm the request validator before concurrent access to its schema cache.
	warmup := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{RoleId: roles[0].ID, Permissions: []string{}})
	require.Equal(t, http.StatusOK, warmup.Status, warmup.RawBody)

	g := errgroup.Group{}
	for _, role := range roles {
		g.Go(func() error {
			body, err := json.Marshal(handler.Request{RoleId: role.ID, Permissions: []string{"concurrent.permission"}})
			if err != nil {
				return err
			}
			req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewReader(body))
			req.Header = headers.Clone()
			recorder := httptest.NewRecorder()
			h.Mux().ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				return fmt.Errorf("role %s: status %d: %s", role.ID, recorder.Code, recorder.Body.String())
			}
			return nil
		})
	}
	require.NoError(t, g.Wait())

	permissions, err := db.Query.FindPermissionsBySlugs(t.Context(), h.DB.RO(), db.FindPermissionsBySlugsParams{
		WorkspaceID: workspace.ID,
		ProjectID:   roles[0].ProjectID,
		Slugs:       []string{"concurrent.permission"},
	})
	require.NoError(t, err)
	require.Len(t, permissions, 1)
	for _, role := range roles {
		assigned, listErr := db.Query.ListDirectPermissionsByRoleID(t.Context(), h.DB.RO(), role.ID)
		require.NoError(t, listErr)
		require.Len(t, assigned, 1)
		require.Equal(t, permissions[0].ID, assigned[0].ID)
	}

	logs := h.FindAuditLogsByTargetID(t.Context(), t, permissions[0].ID)
	createEvents := 0
	for _, log := range logs {
		if log.Event == "permission.create" {
			createEvents++
		}
	}
	require.Equal(t, 1, createEvents)
}
