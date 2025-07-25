package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
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

	t.Run("set single role by ID", func(t *testing.T) {
		// Create API with keyring using testutil helper
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		// Create a test key with role using testutil helper
		roleID := h.CreateRole(seed.CreateRoleRequest{
			Name:        "admin_set_single_id",
			WorkspaceID: workspace.ID,
		})

		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
		})
		keyID := keyResponse.KeyID

		// Verify key has no roles initially
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Empty(t, currentRoles)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
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
		require.Equal(t, roleID, res.Body.Data[0].Id)
		require.Equal(t, "admin_set_single_id", res.Body.Data[0].Name)

		// Verify role was added to key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, roleID, finalRoles[0].ID)

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		foundConnectEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.connect_role_and_key" {
				foundConnectEvent = true
				require.Contains(t, log.AuditLog.Display, "Added role admin_set_single_id to key")
				break
			}
		}
		require.True(t, foundConnectEvent, "Should find a role connect audit log event")
	})

	t.Run("set single role by name", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a role
		roleID := uid.New(uid.TestPrefix)
		roleName := "editor_set_single_name"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Editor role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
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
		require.Equal(t, roleID, res.Body.Data[0].Id)
		require.Equal(t, roleName, res.Body.Data[0].Name)

		// Verify role was added to key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, roleID, finalRoles[0].ID)
	})

	t.Run("set multiple roles mixed references", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create multiple roles
		adminRoleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      adminRoleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_set_multi",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		editorRoleID := uid.New(uid.TestPrefix)
		editorRoleName := "editor_set_multi"
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      editorRoleID,
			WorkspaceID: workspace.ID,
			Name:        editorRoleName,
			Description: sql.NullString{Valid: true, String: "Editor role"},
		})
		require.NoError(t, err)

		viewerRoleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      viewerRoleID,
			WorkspaceID: workspace.ID,
			Name:        "viewer_set_multi",
			Description: sql.NullString{Valid: true, String: "Viewer role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{adminRoleID, editorRoleName, viewerRoleID},
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
		require.ElementsMatch(t, []string{"admin_set_multi", "editor_set_multi", "viewer_set_multi"}, roleNames)

		// Verify roles were added to key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 3)

		// Verify audit logs were created (one for each role)
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), keyID)
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

	t.Run("replace existing roles", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create roles
		oldRoleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      oldRoleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_replace_old",
			Description: sql.NullString{Valid: true, String: "Old admin role"},
		})
		require.NoError(t, err)

		newRoleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      newRoleID,
			WorkspaceID: workspace.ID,
			Name:        "editor_replace_new",
			Description: sql.NullString{Valid: true, String: "New editor role"},
		})
		require.NoError(t, err)

		// First, add old role to the key
		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      oldRoleID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has the old role
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentRoles, 1)
		require.Equal(t, oldRoleID, currentRoles[0].ID)

		// Now set the key to have only the new role (should remove old, add new)
		req := handler.Request{
			KeyId: keyID,
			Roles: []string{newRoleID},
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
		require.Equal(t, newRoleID, res.Body.Data[0].Id)
		require.Equal(t, "editor_replace_new", res.Body.Data[0].Name)

		// Verify only new role exists on key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, newRoleID, finalRoles[0].ID)

		// Verify audit logs show both removal and addition
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		foundDisconnectEvent := false
		foundConnectEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.disconnect_role_and_key" {
				foundDisconnectEvent = true
				require.Contains(t, log.AuditLog.Display, "Removed role admin_replace_old from key")
			}
			if log.AuditLog.Event == "authorization.connect_role_and_key" {
				foundConnectEvent = true
				require.Contains(t, log.AuditLog.Display, "Added role editor_replace_new to key")
			}
		}
		require.True(t, foundDisconnectEvent, "Should find a role disconnect audit log event")
		require.True(t, foundConnectEvent, "Should find a role connect audit log event")
	})

	t.Run("set roles to empty - remove all roles", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a role and assign it to the key
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_remove_all",
			Description: sql.NullString{Valid: true, String: "Admin role to be removed"},
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has the role
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentRoles, 1)

		// Set roles to empty array
		req := handler.Request{
			KeyId: keyID,
			Roles: []string{}, // Empty roles array - remove all
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
		require.Len(t, res.Body.Data, 0) // No roles left

		// Verify no roles exist on key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 0)

		// Verify audit log shows removal
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		foundDisconnectEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.disconnect_role_and_key" {
				foundDisconnectEvent = true
				require.Contains(t, log.AuditLog.Display, "Removed role admin_remove_all from key")
				break
			}
		}
		require.True(t, foundDisconnectEvent, "Should find a role disconnect audit log event")
	})

	t.Run("set same roles as current - no changes", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a role and assign it to the key
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_no_change",
			Description: sql.NullString{Valid: true, String: "Admin role - no change"},
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Count existing audit logs
		auditLogsBefore, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		auditLogCountBefore := len(auditLogsBefore)

		// Set roles to the same role (no change)
		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
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
		require.Equal(t, roleID, res.Body.Data[0].Id)
		require.Equal(t, "admin_no_change", res.Body.Data[0].Name)

		// Verify role still exists on key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, roleID, finalRoles[0].ID)

		// Verify no new audit logs were created (no changes)
		auditLogsAfter, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		auditLogCountAfter := len(auditLogsAfter)
		require.Equal(t, auditLogCountBefore, auditLogCountAfter, "No new audit logs should be created when no changes are made")
	})
}
