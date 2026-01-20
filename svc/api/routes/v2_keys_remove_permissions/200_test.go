package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_remove_permissions"
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
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.remove_permission_from_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("remove single permission by name", func(t *testing.T) {
		// Create API with keyring using testutil helper
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		// Create a test key with permission using testutil helper
		keyName := "Test Key"
		permissionName := "documents.write.remove.single.name"
		permissionDescription := "Write documents permission"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
			Permissions: []seed.CreatePermissionRequest{
				{
					WorkspaceID: workspace.ID,
					Name:        permissionName,
					Slug:        permissionName,
					Description: &permissionDescription,
				},
			},
		})
		keyID := keyResponse.KeyID

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permissionName},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotNil(t, res.Body.Data)

		// Verify permission was removed from key
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 0)
	})

	t.Run("remove multiple permissions", func(t *testing.T) {
		// Create API with keyring using testutil helper
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		// Create a test key using testutil helper
		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create permissions using testutil helpers
		permission1Description := "Read documents permission"
		permission1Name := "documents.read.remove.multiple"
		permission1 := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permission1Name,
			Slug:        permission1Name,
			Description: &permission1Description,
		})

		permission2Description := "Write documents permission"
		permission2Name := "documents.write.remove.multiple"
		permission2 := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permission2Name,
			Slug:        permission2Name,
			Description: &permission2Description,
		})

		// Add both permissions to key first
		err := db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission1.ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission2.ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permission1Name, permission2Name},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotNil(t, res.Body.Data)

		// Verify both permissions were removed from key
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 0)
	})

	t.Run("idempotent operation - removing permission that isn't assigned", func(t *testing.T) {
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

		// Create a permission but don't assign it to the key
		permissionDescription := "Read documents permission"
		permission1Name := "documents.read.remove.idempotent"
		h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permission1Name,
			Slug:        permission1Name,
			Description: &permissionDescription,
		})

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permission1Name},
		}

		// Remove permission (which isn't assigned)
		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res1.Status)
		require.NotNil(t, res1.Body)

		// Remove same permission again
		res2 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res2.Status)
		require.NotNil(t, res2.Body)

		// Verify key still has no permissions
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 0)
	})

	t.Run("remove some permissions from key with multiple permissions", func(t *testing.T) {
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

		// Create permissions
		keepPermissionID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: keepPermissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.remove.partial.keep",
			Slug:         "documents.read.remove.partial.keep",
			Description:  dbtype.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		removePermissionID := uid.New(uid.TestPrefix)
		removePermissionName := "documents.write.remove.partial.remove"
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: removePermissionID,
			WorkspaceID:  workspace.ID,
			Name:         removePermissionName,
			Slug:         removePermissionName,
			Description:  dbtype.NullString{Valid: true, String: "Write documents permission"},
		})
		require.NoError(t, err)

		// Add both permissions to key first
		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: keepPermissionID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: removePermissionID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{removePermissionName},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotNil(t, res.Body.Data)

		// Verify only one permission remains
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, keepPermissionID, finalPermissions[0].ID)
	})

	t.Run("remove all permissions from key", func(t *testing.T) {
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

		// Create multiple permissions using testutil helpers
		permission1Description := "Read documents permission"
		permission1Name := "documents.read.remove.all.1"
		permission1 := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permission1Name,
			Slug:        permission1Name,
			Description: &permission1Description,
		})

		permission2Description := "Write documents permission"
		permission2Name := "documents.write.remove.all.2"
		permission2 := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permission2Name,
			Slug:        permission2Name,
			Description: &permission2Description,
		})

		permission3Description := "Delete documents permission"
		permission3Name := "documents.delete.remove.all.3"
		permission3 := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permission3Name,
			Slug:        permission3Name,
			Description: &permission3Description,
		})

		// Add all permissions to key
		err := db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission1.ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission2.ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission3.ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has all permissions initially
		initialPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, initialPermissions, 3)

		req := handler.Request{
			KeyId: keyID,
			Permissions: []string{
				permission1Name,
				permission2Name,
				permission3Name,
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
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotNil(t, res.Body.Data)

		// Verify all permissions were removed
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 0)
	})
}
