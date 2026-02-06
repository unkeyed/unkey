package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_add_permissions"
	"golang.org/x/sync/errgroup"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a permission using testutil helper
		permission := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        "documents.write.single.name",
			Slug:        "documents.write.single.name",
			Description: ptr.P("Write documents permission"),
		})

		// Verify key has no permissions initially
		currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Empty(t, currentPermissions)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permission.Name},
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
		require.Equal(t, permission.ID, res.Body.Data[0].Id)
		require.Equal(t, "documents.write.single.name", res.Body.Data[0].Name)

		// Verify permission was added to key
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, permission.ID, finalPermissions[0].ID)
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
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create permissions using testutil helper
		permission1 := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        "documents.read.multiple",
			Slug:        "documents.read.multiple",
			Description: ptr.P("Read documents permission"),
		})

		permission2 := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        "documents.write.multiple",
			Slug:        "documents.write.multiple",
			Description: ptr.P("Write documents permission"),
		})

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permission1.Name, permission2.Name},
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
		require.True(t, contains(permission1.ID))
		require.True(t, contains(permission2.ID))

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
			KeySpaceID:  api.KeyAuthID.String,
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
		newPermission := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        "documents.write.existing",
			Slug:        "documents.write.existing",
			Description: &newPermissionDescription,
		})

		// Create a test key with existing permission using testutil helper
		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
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
			Permissions: []string{newPermission.Name},
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
		require.True(t, permissionIDs[newPermission.ID])

		// Verify permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 2)
	})
}

// TestAddPermissionsConcurrent tests that concurrent requests to add permissions
// to the same key don't deadlock. The handler uses SELECT ... FOR UPDATE
// on the key row to serialize concurrent modifications.
func TestAddPermissionsConcurrent(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.add_permission_to_key", "rbac.*.create_permission")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Create a single key that all concurrent requests will update
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        ptr.P("concurrent-permissions-test-key"),
	})

	// Create permissions that will be added concurrently
	numConcurrent := 10
	permissions := make([]string, numConcurrent)
	for i := range numConcurrent {
		perm := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("concurrent.permission.%d", i),
			Slug:        fmt.Sprintf("concurrent.permission.%d", i),
		})
		permissions[i] = perm.Name
	}

	g := errgroup.Group{}
	for i := range numConcurrent {
		g.Go(func() error {
			// Each request adds multiple permissions to the same key
			req := handler.Request{
				KeyId:       keyResponse.KeyID,
				Permissions: []string{permissions[i], permissions[(i+1)%numConcurrent], permissions[(i+2)%numConcurrent]},
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if res.Status != 200 {
				return fmt.Errorf("request %d: unexpected status %d", i, res.Status)
			}
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err, "All concurrent updates should succeed without deadlock")

	// Verify all permissions were added
	finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(t.Context(), h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.Len(t, finalPermissions, numConcurrent)
}
