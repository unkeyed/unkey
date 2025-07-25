package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_remove_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFoundErrors(t *testing.T) {
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

	// Create workspace and root key
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.remove_role_from_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("key not found", func(t *testing.T) {
		// Create a role for the request
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "test_role_key_not_found_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		// Generate a non-existent key ID
		nonExistentKeyID := uid.New(uid.KeyPrefix)

		req := handler.Request{
			KeyId: nonExistentKeyID,
			Roles: []string{roleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "key")
		require.Contains(t, res.Body.Error.Detail, "not found")
	})

	t.Run("role not found by ID", func(t *testing.T) {
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

		// Generate a non-existent role ID
		nonExistentRoleID := uid.New(uid.TestPrefix)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{nonExistentRoleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Role")
		require.Contains(t, res.Body.Error.Detail, "not found")
		require.Contains(t, res.Body.Error.Detail, nonExistentRoleID)
	})

	t.Run("role not found by name", func(t *testing.T) {
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

		// Use a non-existent role name
		nonExistentRoleName := "non_existent_role_name"

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{nonExistentRoleName},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Role")
		require.Contains(t, res.Body.Error.Detail, "not found")
		require.Contains(t, res.Body.Error.Detail, nonExistentRoleName)
	})

	t.Run("key from different workspace", func(t *testing.T) {
		// Create two workspaces
		workspace2 := h.CreateWorkspace()

		// Create API and key in workspace2 using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace2.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace2.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create role in original workspace
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "test_role_key_diff_workspace_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should return 404 for security (workspace isolation)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "key")
		require.Contains(t, res.Body.Error.Detail, "not found")
	})

	t.Run("role from different workspace by ID", func(t *testing.T) {
		// Create two workspaces
		workspace2 := h.CreateWorkspace()

		// Create API and key in original workspace using testutil helpers
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

		// Create role in workspace2 (different workspace)
		roleInDifferentWorkspace := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleInDifferentWorkspace,
			WorkspaceID: workspace2.ID,
			Name:        "admin_diff_workspace_id_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleInDifferentWorkspace},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should return 404 for security (workspace isolation)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "not found")
	})

	t.Run("role from different workspace by name", func(t *testing.T) {
		// Create two workspaces
		workspace2 := h.CreateWorkspace()

		// Create API and key in original workspace using testutil helpers
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

		// Create role in workspace2 (different workspace)
		roleInDifferentWorkspace := uid.New(uid.TestPrefix)
		roleName := "admin_by_name_" + uid.New("")
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleInDifferentWorkspace,
			WorkspaceID: workspace2.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleName},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should return 404 for security (workspace isolation)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "not found")
	})

	t.Run("multiple roles with one not found", func(t *testing.T) {
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

		// Create one valid role
		validRoleID := uid.New(uid.TestPrefix)
		validName := "valid_role"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      validRoleID,
			WorkspaceID: workspace.ID,
			Name:        validName,
			Description: sql.NullString{Valid: true, String: "Valid role"},
		})
		require.NoError(t, err)

		// Generate a non-existent role ID
		nonExistentRoleName := "someRole"

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{validName, nonExistentRoleName},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should fail when ANY role is not found
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Role")
		require.Contains(t, res.Body.Error.Detail, "not found")
		require.Contains(t, res.Body.Error.Detail, nonExistentRoleName)
	})
}
