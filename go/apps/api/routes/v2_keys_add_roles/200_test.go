package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace and root key
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("add single role by ID", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			EncryptedKeys: false,
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			KeyAuthID:   api.KeyAuthID.String,
			WorkspaceID: workspace.ID,
		})

		roleId := h.CreateRole(seed.CreateRoleRequest{

			WorkspaceID: workspace.ID,
			Name:        "admin_single_id",
			Description: ptr.P("Admin Role"),
		})

		// Verify key has no roles initially
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Empty(t, currentRoles)

		req := handler.Request{
			KeyId: key.KeyID,
			Roles: []string{roleId},
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
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, roleId, res.Body.Data[0].Id)
		require.Equal(t, "admin_single_id", res.Body.Data[0].Name)

		// Verify role was added to key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, roleId, finalRoles[0].ID)

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		foundConnectEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.connect_role_and_key" {
				foundConnectEvent = true
				require.Contains(t, log.AuditLog.Display, "Added role admin_single_id to key")
				break
			}
		}
		require.True(t, foundConnectEvent, "Should find a role connect audit log event")
	})

	t.Run("add single role by name", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			EncryptedKeys: false,
		})

		roleName := "editor_single_name"
		key := h.CreateKey(seed.CreateKeyRequest{
			KeyAuthID:   api.KeyAuthID.String,
			WorkspaceID: workspace.ID,
			Roles: []seed.CreateRoleRequest{
				{
					WorkspaceID: workspace.ID,
					Name:        "editor_single_name",
					Description: ptr.P(roleName),
				},
			},
		})

		req := handler.Request{
			KeyId: key.KeyID,
			Roles: []string{roleName},
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
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, key.RolesIds[0], res.Body.Data[0].Id)
		require.Equal(t, roleName, res.Body.Data[0].Name)

		// Verify role was added to key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, key.RolesIds[0], finalRoles[0].ID)
	})

	t.Run("add multiple roles mixed references", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			EncryptedKeys: false,
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			KeyAuthID:   api.KeyAuthID.String,
			WorkspaceID: workspace.ID,
		})

		adminRole := h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        "admin_multi",
		})

		viewerMultiRole := h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        "viewer_multi",
		})

		editorRoleName := "editor_multi"
		h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        editorRoleName,
		})

		req := handler.Request{
			KeyId: key.KeyID,
			Roles: []string{adminRole, editorRoleName, viewerMultiRole},
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
		require.Len(t, res.Body.Data, 3)

		// Verify all roles are present and sorted alphabetically
		roleNames := []string{res.Body.Data[0].Name, res.Body.Data[1].Name, res.Body.Data[2].Name}
		require.Equal(t, []string{"admin_multi", "editor_multi", "viewer_multi"}, roleNames)

		// Verify roles were added to key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 3)

		// Verify audit logs were created (one for each role)
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		connectEvents := 0
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.connect_role_and_key" {
				connectEvents++
				require.Contains(t, log.AuditLog.Display, "Added role")
			}
		}
		require.Equal(t, 3, connectEvents, "Should find 3 role connect audit log events")
	})

	t.Run("idempotent behavior - add existing roles", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			EncryptedKeys: false,
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			KeyAuthID:   api.KeyAuthID.String,
			WorkspaceID: workspace.ID,
		})
		adminId := h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        "admin_idempotent",
			Description: ptr.P("admin_idempotent"),
		})

		editorId := h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        "editor_idempotent",
			Description: ptr.P("editor_idempotent"),
		})

		// First, add admin role to the key
		err := db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       key.KeyID,
			RoleID:      adminId,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Now try to add both admin (existing) and editor (new) roles
		req := handler.Request{
			KeyId: key.KeyID,
			Roles: []string{adminId, editorId},
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
		require.Len(t, res.Body.Data, 2)

		// Verify both roles are present
		roleNames := []string{res.Body.Data[0].Name, res.Body.Data[1].Name}
		require.Equal(t, []string{"admin_idempotent", "editor_idempotent"}, roleNames)

		// Verify roles in database
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 2)

		// Verify audit logs - should only have one new log for the editor role
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		// Count connect events that mention editor (the new role)
		editorConnectEvents := 0
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.connect_role_and_key" &&
				log.AuditLog.Display == fmt.Sprintf("Added role editor_idempotent to key %s", key.KeyID) {
				editorConnectEvents++
			}
		}
		require.Equal(t, 1, editorConnectEvents, "Should find only 1 new role connect event for editor")
	})

	t.Run("role reference with both ID and name", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			EncryptedKeys: false,
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			KeyAuthID:   api.KeyAuthID.String,
			WorkspaceID: workspace.ID,
		})

		adminID := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspace.ID, Name: "admin_both_ref"})
		editorRoleName := "editor_both_ref"
		h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspace.ID, Name: editorRoleName})

		// Request with role reference having both ID and name
		// ID should take precedence
		req := handler.Request{
			KeyId: key.KeyID,
			Roles: []string{adminID, editorRoleName},
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
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, adminID, res.Body.Data[0].Id)
		require.Equal(t, "admin_both_ref", res.Body.Data[0].Name) // Should be role1, not role2

		// Verify correct role was added
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, adminID, finalRoles[0].ID)
	})
}
