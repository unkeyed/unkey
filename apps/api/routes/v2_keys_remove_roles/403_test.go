package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_keys_remove_roles"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestAuthorizationErrors(t *testing.T) {
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

	// Create workspace for test setup
	workspace := h.Resources().UserWorkspace

	t.Run("root key without required permissions", func(t *testing.T) {
		// Create root key WITHOUT update permissions
		rootKey := h.CreateRootKey(workspace.ID) // No permissions

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
			Name:        "test_role_no_perms_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	t.Run("root key with partial permissions", func(t *testing.T) {
		// Create root key with insufficient permissions
		rootKey := h.CreateRootKey(workspace.ID, "api.read.update_key") // Read instead of wildcard

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
			Name:        "test_role_partial_perms_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	t.Run("root key with unrelated permissions", func(t *testing.T) {
		// Create root key with unrelated permissions
		rootKey := h.CreateRootKey(workspace.ID, "different.permission.scope")

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
			Name:        "test_role_unrelated_perms_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	t.Run("root key with create permissions but not update", func(t *testing.T) {
		// Create root key with create permission instead of update
		rootKey := h.CreateRootKey(workspace.ID, "api.*.create_key")

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
			Name:        "test_role_create_perms_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	t.Run("root key with delete permissions but not update", func(t *testing.T) {
		// Create root key with delete permission instead of update
		rootKey := h.CreateRootKey(workspace.ID, "api.*.delete_key")

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
			Name:        "test_role_delete_perms_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	t.Run("root key with specific API permissions but not wildcard", func(t *testing.T) {
		// Create root key with specific API permission (not wildcard)
		rootKey := h.CreateRootKey(workspace.ID, "api.specific.update_key")

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
			Name:        "test_role_specific_perms_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	t.Run("root key with mixed permissions but missing required", func(t *testing.T) {
		// Create root key with multiple permissions but missing the required one
		rootKey := h.CreateRootKey(workspace.ID, "api.*.create_key", "api.*.delete_key", "api.*.read_key")
		// Missing "api.*.update_key"

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
			Name:        "test_role_mixed_perms_" + uid.New(""),
			Description: sql.NullString{Valid: true, String: "Test role"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Roles: []string{roleID},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})
}
