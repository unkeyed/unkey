package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_remove_roles"
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

	t.Run("remove single role by ID", func(t *testing.T) {
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
		role1ID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      role1ID,
			WorkspaceID: workspace.ID,
			Name:        "admin",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		role2ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      role2ID,
			WorkspaceID: workspace.ID,
			Name:        "developer",
			Description: sql.NullString{Valid: true, String: "Developer role"},
		})
		require.NoError(t, err)

		// Assign both roles to the key
		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      role1ID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      role2ID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has both roles initially
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentRoles, 2)

		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &role1ID},
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
		require.Equal(t, role2ID, res.Body.Data[0].Id)
		require.Equal(t, "developer", res.Body.Data[0].Name)

		// Verify role was removed from key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, role2ID, finalRoles[0].ID)

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		foundDisconnectEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.disconnect_role_and_key" {
				foundDisconnectEvent = true
				require.Contains(t, log.AuditLog.Display, "Removed role admin from key")
				break
			}
		}
		require.True(t, foundDisconnectEvent, "Should find a role disconnect audit log event")
	})

	t.Run("remove single role by name", func(t *testing.T) {
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
		roleName := "editor"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Editor role"},
		})
		require.NoError(t, err)

		// Assign role to the key
		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has role initially
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentRoles, 1)

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
		require.Len(t, res.Body.Data, 0) // No roles remaining

		// Verify role was removed from key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 0)
	})

	t.Run("idempotent operation - removing non-assigned role", func(t *testing.T) {
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

		// Create a role but don't assign it to the key
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "unassigned_role",
			Description: sql.NullString{Valid: true, String: "Unassigned role"},
		})
		require.NoError(t, err)

		// Verify key has no roles initially
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentRoles, 0)

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
		require.Len(t, res.Body.Data, 0) // Still no roles

		// Verify key still has no roles
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 0)
	})
}
