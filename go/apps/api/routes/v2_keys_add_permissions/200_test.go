package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
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
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.add_permission_to_key", "rbac.*.create_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("add single permission by name", func(t *testing.T) {
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
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a permission using testutil helper
		permissionSlug := "documents.write.single.name"
		permissionDescription := "Write documents permission"
		permissionID := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permissionSlug,
			Slug:        permissionSlug,
			Description: &permissionDescription,
		})

		// Verify key has no permissions initially
		currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Empty(t, currentPermissions)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permissionSlug},
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
		require.Equal(t, permissionID, res.Body.Data[0].Id)
		require.Equal(t, "documents.write.single.name", res.Body.Data[0].Name)

		// Verify permission was added to key
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, permissionID, finalPermissions[0].ID)
	})

	t.Run("add multiple permissions", func(t *testing.T) {
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
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create permissions using testutil helper
		permission1Name := "documents.read.multiple"
		permissionDescription1 := "Read documents permission"
		permission1ID := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permission1Name,
			Slug:        permission1Name,
			Description: &permissionDescription1,
		})

		permission2Slug := "documents.write.multiple"
		permissionDescription2 := "Write documents permission"
		permission2ID := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permission2Slug,
			Slug:        permission2Slug,
			Description: &permissionDescription2,
		})

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permission1Name, permission2Slug},
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

		contains := func(id string) bool {
			for _, p := range res.Body.Data {
				if p.Id == id {
					return true
				}
			}
			return false
		}

		// Verify both permissions are in response
		require.True(t, contains(permission1ID))
		require.True(t, contains(permission2ID))

		// Verify permissions were added to key
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 2)
	})

	t.Run("idempotent operation - adding same permission twice", func(t *testing.T) {
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
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a permission using testutil helper
		permissionDescription := "Read documents permission"
		permissionName := "documents.read.idempotent"
		h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        permissionName,
			Slug:        permissionName,
			Description: &permissionDescription,
		})

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permissionName},
		}

		// Add permission first time
		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res1.Status)
		require.Len(t, res1.Body.Data, 1)

		// Add same permission again
		res2 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res2.Status)
		require.Len(t, res2.Body.Data, 1)

		// Both responses should be identical
		require.Equal(t, res1.Body.Data[0].Id, res2.Body.Data[0].Id)
		require.Equal(t, res1.Body.Data[0].Name, res2.Body.Data[0].Name)

		// Verify only one permission in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
	})

	t.Run("add permissions to key that already has permissions", func(t *testing.T) {
		// Create API with keyring using testutil helper
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		// Create permissions using testutil helper
		existingPermissionDescription := "Read documents permission"
		newPermissionDescription := "Write documents permission"
		newPermissionSlug := "documents.write.existing"
		newPermissionID := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        newPermissionSlug,
			Slug:        newPermissionSlug,
			Description: &newPermissionDescription,
		})

		// Create a test key with existing permission using testutil helper
		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
			Permissions: []seed.CreatePermissionRequest{
				{
					WorkspaceID: workspace.ID,
					Name:        "documents.read.existing",
					Slug:        "documents.read.existing",
					Description: &existingPermissionDescription,
				},
			},
		})
		keyID := keyResponse.KeyID

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{newPermissionSlug},
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
		require.Len(t, res.Body.Data, 2) // Should have both permissions

		// Verify all permissions are in response (sorted alphabetically)
		permissionIDs := make(map[string]bool)
		for _, p := range res.Body.Data {
			permissionIDs[p.Id] = true
		}
		require.True(t, permissionIDs[keyResponse.PermissionIds[0]])
		require.True(t, permissionIDs[newPermissionID])

		// Verify permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 2)
	})
}
