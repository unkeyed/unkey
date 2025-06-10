package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

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
		// Create a test keyring
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        workspace.ID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test key
		keyID := uid.New(uid.KeyPrefix)
		keyString := "test_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              hash.Sha256(keyString),
			Start:             keyString[:4],
			WorkspaceID:       workspace.ID,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Name:              sql.NullString{Valid: true, String: "Test Key"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			IdentityID:        sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Create a role
		roleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_single_id",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		// Verify key has no roles initially
		currentRoles, err := db.Query.FindRolesForKey(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Empty(t, currentRoles)

		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &roleID},
			},
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
		require.Equal(t, "admin_single_id", res.Body.Data[0].Name)

		// Verify role was added to key
		finalRoles, err := db.Query.FindRolesForKey(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, roleID, finalRoles[0].ID)

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetById(ctx, h.DB.RO(), keyID)
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
		// Create a test keyring
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        workspace.ID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test key
		keyID := uid.New(uid.KeyPrefix)
		keyString := "test_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              hash.Sha256(keyString),
			Start:             keyString[:4],
			WorkspaceID:       workspace.ID,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Name:              sql.NullString{Valid: true, String: "Test Key"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			IdentityID:        sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Create a role
		roleID := uid.New(uid.TestPrefix)
		roleName := "editor_single_name"
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Editor role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Name: &roleName},
			},
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
		finalRoles, err := db.Query.FindRolesForKey(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, roleID, finalRoles[0].ID)
	})

	t.Run("add multiple roles mixed references", func(t *testing.T) {
		// Create a test keyring
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        workspace.ID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test key
		keyID := uid.New(uid.KeyPrefix)
		keyString := "test_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              hash.Sha256(keyString),
			Start:             keyString[:4],
			WorkspaceID:       workspace.ID,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Name:              sql.NullString{Valid: true, String: "Test Key"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			IdentityID:        sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Create multiple roles
		adminRoleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      adminRoleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_multi",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		editorRoleID := uid.New(uid.TestPrefix)
		editorRoleName := "editor_multi"
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
			Name:        "viewer_multi",
			Description: sql.NullString{Valid: true, String: "Viewer role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &adminRoleID},      // By ID
				{Name: &editorRoleName}, // By name
				{Id: &viewerRoleID},     // By ID
			},
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
		finalRoles, err := db.Query.FindRolesForKey(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 3)

		// Verify audit logs were created (one for each role)
		auditLogs, err := db.Query.FindAuditLogTargetById(ctx, h.DB.RO(), keyID)
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
		// Create a test keyring
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        workspace.ID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test key
		keyID := uid.New(uid.KeyPrefix)
		keyString := "test_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              hash.Sha256(keyString),
			Start:             keyString[:4],
			WorkspaceID:       workspace.ID,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Name:              sql.NullString{Valid: true, String: "Test Key"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			IdentityID:        sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Create roles
		adminRoleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      adminRoleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_idempotent",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		editorRoleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      editorRoleID,
			WorkspaceID: workspace.ID,
			Name:        "editor_idempotent",
			Description: sql.NullString{Valid: true, String: "Editor role"},
		})
		require.NoError(t, err)

		// First, add admin role to the key
		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      adminRoleID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Now try to add both admin (existing) and editor (new) roles
		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &adminRoleID},  // Already exists
				{Id: &editorRoleID}, // New role
			},
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
		finalRoles, err := db.Query.FindRolesForKey(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 2)

		// Verify audit logs - should only have one new log for the editor role
		auditLogs, err := db.Query.FindAuditLogTargetById(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		// Count connect events that mention editor (the new role)
		editorConnectEvents := 0
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.connect_role_and_key" &&
				log.AuditLog.Display == fmt.Sprintf("Added role editor_idempotent to key %s", keyID) {
				editorConnectEvents++
			}
		}
		require.Equal(t, 1, editorConnectEvents, "Should find only 1 new role connect event for editor")
	})

	t.Run("role reference with both ID and name", func(t *testing.T) {
		// Create a test keyring
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        workspace.ID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test key
		keyID := uid.New(uid.KeyPrefix)
		keyString := "test_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              hash.Sha256(keyString),
			Start:             keyString[:4],
			WorkspaceID:       workspace.ID,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Name:              sql.NullString{Valid: true, String: "Test Key"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			IdentityID:        sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Create roles
		role1ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      role1ID,
			WorkspaceID: workspace.ID,
			Name:        "admin_both_ref",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		role2ID := uid.New(uid.TestPrefix)
		role2Name := "editor_both_ref"
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      role2ID,
			WorkspaceID: workspace.ID,
			Name:        role2Name,
			Description: sql.NullString{Valid: true, String: "Editor role"},
		})
		require.NoError(t, err)

		// Request with role reference having both ID and name
		// ID should take precedence
		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{
					Id:   &role1ID,
					Name: &role2Name, // This should be ignored, ID takes precedence
				},
			},
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
		require.Equal(t, role1ID, res.Body.Data[0].Id)
		require.Equal(t, "admin_both_ref", res.Body.Data[0].Name) // Should be role1, not role2

		// Verify correct role was added
		finalRoles, err := db.Query.FindRolesForKey(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, role1ID, finalRoles[0].ID)
	})
}
