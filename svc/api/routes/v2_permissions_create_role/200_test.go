package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for creating a role without permissions
	t.Run("create role without permissions", func(t *testing.T) {
		roleName := "test.role.no.permissions"
		description := "Test role without permissions"
		req := handler.Request{
			Name:        roleName,
			Description: &description,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.NotEmpty(t, res.Body.Data.RoleId)
		require.True(t, len(res.Body.Data.RoleId) > 0, "RoleId should not be empty")

		// Verify role was created in database
		role, err := db.Query.FindRoleByID(ctx, h.DB.RO(), res.Body.Data.RoleId)
		require.NoError(t, err)
		projectID, err := db.Query.FindDefaultProjectByWorkspaceID(ctx, h.DB.RO(), workspace.ID)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, role.ID)
		require.Equal(t, req.Name, role.Name)
		require.Equal(t, description, role.Description.String)
		require.Equal(t, workspace.ID, role.WorkspaceID)
		require.Equal(t, projectID, role.ProjectID)

		// Verify the audit log was queued in clickhouse_outbox.
		auditLogs := h.FindAuditLogsByTargetID(ctx, t, res.Body.Data.RoleId)
		require.NotEmpty(t, auditLogs, "Audit log for role creation should exist")
		foundCreateEvent := false
		for _, ev := range auditLogs {
			if ev.Event == "role.create" {
				foundCreateEvent = true
				break
			}
		}
		require.True(t, foundCreateEvent, "Should find a role.create audit log event")
	})

	// Test case for creating a role with description
	t.Run("create role with description", func(t *testing.T) {
		roleName := "test.role.with.description"
		description := "Test role with a description field"

		req := handler.Request{
			Name:        roleName,
			Description: &description,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.NotEmpty(t, res.Body.Data.RoleId)

		// Verify role was created in database
		role, err := db.Query.FindRoleByID(ctx, h.DB.RO(), res.Body.Data.RoleId)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, role.ID)
		require.Equal(t, req.Name, role.Name)
		require.Equal(t, description, role.Description.String)
		require.Equal(t, workspace.ID, role.WorkspaceID)

		// Verify the audit log was queued in clickhouse_outbox.
		auditLogs := h.FindAuditLogsByTargetID(ctx, t, res.Body.Data.RoleId)
		require.NotEmpty(t, auditLogs, "Audit log for role creation should exist")
		foundCreateEvent := false
		for _, ev := range auditLogs {
			if ev.Event == "role.create" {
				foundCreateEvent = true
				break
			}
		}
		require.True(t, foundCreateEvent, "Should find a role.create audit log event")
	})

	// Test case for creating a role without description
	t.Run("create role without description", func(t *testing.T) {
		emptyPermissions := []string{}
		req := handler.Request{
			Name:        "test.role.no.desc",
			Permissions: &emptyPermissions,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.NotEmpty(t, res.Body.Data.RoleId)

		// Verify role was created in database
		role, err := db.Query.FindRoleByID(ctx, h.DB.RO(), res.Body.Data.RoleId)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, role.ID)
		require.Equal(t, req.Name, role.Name)
		require.False(t, role.Description.Valid, "Description should be null")
		require.Equal(t, workspace.ID, role.WorkspaceID)
	})

	t.Run("create role with existing and missing permissions", func(t *testing.T) {
		existingPermission := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        "documents.read.create.role",
			Slug:        "documents.read.create.role",
		})
		permissionSlugs := []string{
			existingPermission.Slug,
			"documents.write.create.role",
			"Documents.Write.Create.Role",
		}
		rootKeyWithPermissions := h.CreateRootKey(
			workspace.ID,
			"rbac.*.create_role",
			"rbac.*.add_permission_to_role",
			"rbac.*.create_permission",
		)
		permissionHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyWithPermissions)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, permissionHeaders, handler.Request{
			Name:        "test.role.with.permissions",
			Permissions: &permissionSlugs,
		})

		require.Equal(t, http.StatusOK, res.Status)

		createdPermissions, err := db.Query.FindPermissionsBySlugs(ctx, h.DB.RO(), db.FindPermissionsBySlugsParams{
			WorkspaceID: workspace.ID,
			Slugs:       []string{"documents.write.create.role"},
		})
		require.NoError(t, err)
		require.Len(t, createdPermissions, 1)

		for _, permissionID := range []string{existingPermission.ID, createdPermissions[0].ID} {
			assignments, findErr := db.Query.FindRolePermissionByRoleAndPermissionID(ctx, h.DB.RO(), db.FindRolePermissionByRoleAndPermissionIDParams{
				RoleID:       res.Body.Data.RoleId,
				PermissionID: permissionID,
			})
			require.NoError(t, findErr)
			require.Len(t, assignments, 1)
		}

		roleAuditLogs := h.FindAuditLogsByTargetID(ctx, t, res.Body.Data.RoleId)
		require.Len(t, roleAuditLogs, 3)
		require.Equal(t, "role.create", roleAuditLogs[0].Event)
		require.Equal(t, "authorization.connect_role_and_permission", roleAuditLogs[1].Event)
		require.Equal(t, roleAuditLogs[1].CorrelationID, roleAuditLogs[2].CorrelationID)

		permissionAuditLogs := h.FindAuditLogsByTargetID(ctx, t, createdPermissions[0].ID)
		require.Len(t, permissionAuditLogs, 2)
		require.Equal(t, "permission.create", permissionAuditLogs[0].Event)
		require.Equal(t, permissionAuditLogs[0].CorrelationID, permissionAuditLogs[1].CorrelationID)
	})

	t.Run("existing permission does not require create_permission", func(t *testing.T) {
		existingPermission := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        "Documents.Read.Existing.Only",
			Slug:        "Documents.Read.Existing.Only",
		})
		permissionSlugs := []string{"documents.read.existing.only"}
		rootKeyWithoutCreatePermission := h.CreateRootKey(
			workspace.ID,
			"rbac.*.create_role",
			"rbac.*.add_permission_to_role",
		)
		permissionHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyWithoutCreatePermission)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, permissionHeaders, handler.Request{
			Name:        "test.role.with.existing.permission",
			Permissions: &permissionSlugs,
		})

		require.Equal(t, http.StatusOK, res.Status)
		assignments, err := db.Query.FindRolePermissionByRoleAndPermissionID(ctx, h.DB.RO(), db.FindRolePermissionByRoleAndPermissionIDParams{
			RoleID:       res.Body.Data.RoleId,
			PermissionID: existingPermission.ID,
		})
		require.NoError(t, err)
		require.Len(t, assignments, 1)
	})

	t.Run("concurrent roles reuse a missing permission", func(t *testing.T) {
		rootKeyWithPermissions := h.CreateRootKey(
			workspace.ID,
			"rbac.*.create_role",
			"rbac.*.add_permission_to_role",
			"rbac.*.create_permission",
		)
		permissionHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyWithPermissions)},
		}

		type result struct {
			status int
			roleID string
		}
		results := make(chan result, 2)
		start := make(chan struct{})
		var requests sync.WaitGroup
		for i := range 2 {
			requests.Add(1)
			go func() {
				defer requests.Done()
				<-start
				permissionSlugs := []string{"documents.concurrent.create.role"}
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, permissionHeaders, handler.Request{
					Name:        fmt.Sprintf("test.role.concurrent.%d", i),
					Permissions: &permissionSlugs,
				})
				results <- result{status: res.Status, roleID: res.Body.Data.RoleId}
			}()
		}
		close(start)
		requests.Wait()
		close(results)

		permissions, err := db.Query.FindPermissionsBySlugs(ctx, h.DB.RO(), db.FindPermissionsBySlugsParams{
			WorkspaceID: workspace.ID,
			Slugs:       []string{"documents.concurrent.create.role"},
		})
		require.NoError(t, err)
		require.Len(t, permissions, 1)
		for res := range results {
			require.Equal(t, http.StatusOK, res.status)
			assignments, findErr := db.Query.FindRolePermissionByRoleAndPermissionID(ctx, h.DB.RO(), db.FindRolePermissionByRoleAndPermissionIDParams{
				RoleID:       res.roleID,
				PermissionID: permissions[0].ID,
			})
			require.NoError(t, findErr)
			require.Len(t, assignments, 1)
		}
	})
}
