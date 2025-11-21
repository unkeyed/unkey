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

func TestValidationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		LiveKeyCache: h.Caches.LiveKeyByID,
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

	t.Run("missing keyId field", func(t *testing.T) {
		// Create a role for the request
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "test_role_missing_keyid_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		// Use map to completely omit the keyId field (not just empty string)
		reqMap := map[string]interface{}{
			"roles": []map[string]interface{}{
				{"id": roleID},
			},
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			reqMap,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/keys.removeRoles' failed to validate schema")
	})

	t.Run("empty keyId string", func(t *testing.T) {
		// Create a role for the request
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "test_role_empty_keyid_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: "", // Empty keyId string
			Roles: []string{roleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/keys.removeRoles' failed to validate schema")
	})

	t.Run("invalid keyId format", func(t *testing.T) {
		// Create a role for the request
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "test_role_invalid_keyid_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: "invalid_key_format", // Invalid format (doesn't start with key_)
			Roles: []string{roleID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// This returns 404 because the handler tries to find the key and fails
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "not found")
	})

	t.Run("missing roles field", func(t *testing.T) {
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

		// Use map to completely omit the roles field
		reqMap := map[string]interface{}{
			"keyId": keyID,
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			reqMap,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/keys.removeRoles' failed to validate schema")
	})

	t.Run("empty roles array", func(t *testing.T) {
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

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{}, // Empty roles array
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/keys.removeRoles' failed to validate schema")
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

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
		require.Contains(t, res.Body.Error.Detail, "not found")
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

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
		require.Contains(t, res.Body.Error.Detail, "not found")
	})

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
		require.Contains(t, res.Body.Error.Detail, "not found")
	})

	t.Run("role from different workspace by ID", func(t *testing.T) {
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

		// Create a second workspace and role in it
		workspace2 := h.CreateWorkspace()
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace2.ID,
			Name:        "role_different_workspace_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Role in different workspace"},
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
		require.Contains(t, res.Body.Error.Detail, "not found")
	})

	t.Run("role from different workspace by name", func(t *testing.T) {
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

		// Create a second workspace and role in it
		workspace2 := h.CreateWorkspace()
		roleID := uid.New(uid.TestPrefix)
		roleName := "role_different_workspace_by_name_" + uid.New("")
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace2.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Role in different workspace"},
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
}
