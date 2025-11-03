package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFoundErrors(t *testing.T) {
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
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.add_role_to_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for key not found
	t.Run("key not found", func(t *testing.T) {
		req := handler.Request{
			KeyId: "key_nonexistent123456789", // Non-existent key ID
			Roles: []string{"role_123"},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "The specified key was not found")
	})

	// Test case for key in different workspace
	t.Run("key in different workspace", func(t *testing.T) {
		// Create a second workspace with a key
		workspace2 := h.CreateWorkspace()

		// Create a test API and key in workspace2 using testutil helper
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		req := handler.Request{
			KeyId: keyID, // Key from workspace2, but accessed with workspace1 root key
			Roles: []string{"role_123"},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "The specified key was not found")
	})

	// Test case for role not found by ID
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		nonExistentRoleId := "role_nonexistent123456789"
		req := handler.Request{
			KeyId: keyID,
			Roles: []string{nonExistentRoleId},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for role not found by name
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		nonExistentRoleName := "nonexistent_role"
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
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for role in different workspace
	t.Run("role in different workspace", func(t *testing.T) {
		// Create a second workspace with a role
		workspace2 := h.CreateWorkspace()
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace2.ID,
			Name:        "admin_diff_workspace",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		// Create API and key in workspace1 using testutil helpers
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

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

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for role by name in different workspace
	t.Run("role by name in different workspace", func(t *testing.T) {
		// Create a second workspace with a role
		workspace2 := h.CreateWorkspace()
		roleID := uid.New(uid.TestPrefix)
		roleName := "admin_by_name_diff_workspace"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace2.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		// Create API and key in workspace1 using testutil helpers
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

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

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for multiple roles with one not found
	t.Run("multiple roles one not found", func(t *testing.T) {
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create one valid role
		validRoleID := uid.New(uid.TestPrefix)
		validName := "admin_multiple_roles"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      validRoleID,
			WorkspaceID: workspace.ID,
			Name:        validName,
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		invalidName := "role_nonexistent123456789"

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{validName, invalidName},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, invalidName)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for deleted key
	t.Run("deleted key", func(t *testing.T) {
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
			Deleted:     true,
		})

		err := db.Query.SoftDeleteKeyByID(ctx, h.DB.RW(), db.SoftDeleteKeyByIDParams{
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ID:  keyResponse.KeyID,
		})
		require.NoError(t, err)

		// Create a role
		roleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_deleted_key",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyResponse.KeyID,
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
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "The specified key was not found")
	})

	// Test case for deleted role
	t.Run("deleted role", func(t *testing.T) {
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a role
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_deleted_role",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		// Use non-existent role ID to simulate deleted role
		deletedRoleID := "role_deleted123456789"

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{deletedRoleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})
}
