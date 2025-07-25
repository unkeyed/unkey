package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
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
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.remove_permission_from_key", "rbac.*.add_permission_to_key", "rbac.*.create_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("set permissions using permission IDs", func(t *testing.T) {
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

		// Create permissions
		permission1ID := uid.New(uid.TestPrefix)
		permission1Slug := "documents.read.initial"
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			Name:         permission1Slug,
			Slug:         permission1Slug,
			Description:  dbtype.NullString{Valid: true, String: "Initial permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		permission2Slug := "documents.write.new"
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			Name:         permission2Slug,
			Slug:         permission2Slug,
			Description:  dbtype.NullString{Valid: true, String: "Write permission"},
		})
		require.NoError(t, err)

		permission3ID := uid.New(uid.TestPrefix)
		permission3Slug := "documents.delete.new"
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission3ID,
			WorkspaceID:  workspace.ID,
			Name:         permission3Slug,
			Slug:         permission3Slug,
			Description:  dbtype.NullString{Valid: true, String: "Delete permission"},
		})
		require.NoError(t, err)

		// Add initial permission to key
		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has initial permission
		currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentPermissions, 1)
		require.Equal(t, permission1ID, currentPermissions[0].ID)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permission2Slug, permission3Slug},
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

		// Verify response contains new permissions
		require.True(t, contains(permission3ID))
		require.True(t, contains(permission2ID))

		// Verify permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 2)

		permissionIDs := make([]string, len(finalPermissions))
		for i, perm := range finalPermissions {
			permissionIDs[i] = perm.ID
		}
		require.Contains(t, permissionIDs, permission2ID)
		require.Contains(t, permissionIDs, permission3ID)
		require.NotContains(t, permissionIDs, permission1ID) // Should be removed
	})

	t.Run("set permissions using permission names", func(t *testing.T) {
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

		// Create permissions
		permission1ID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.byname",
			Slug:         "documents.read.byname",
			Description:  dbtype.NullString{Valid: true, String: "Read permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.write.byname",
			Slug:         "documents.write.byname",
			Description:  dbtype.NullString{Valid: true, String: "Write permission"},
		})
		require.NoError(t, err)

		// Add initial permission to key
		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{"documents.write.byname"},
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
		require.Equal(t, permission2ID, res.Body.Data[0].Id)
		require.Equal(t, "documents.write.byname", res.Body.Data[0].Name)

		// Verify permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, permission2ID, finalPermissions[0].ID)
	})

	t.Run("set empty permissions (remove all)", func(t *testing.T) {
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

		// Create and assign permissions
		permission1ID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.empty",
			Slug:         "documents.read.empty",
			Description:  dbtype.NullString{Valid: true, String: "Read permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.write.empty",
			Slug:         "documents.write.empty",
			Description:  dbtype.NullString{Valid: true, String: "Write permission"},
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has permissions initially
		currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentPermissions, 2)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{},
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
		require.Len(t, res.Body.Data, 0)

		// Verify no permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 0)
	})

	t.Run("set permissions with no changes (idempotent)", func(t *testing.T) {
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

		// Create permission
		permissionID := uid.New(uid.PermissionPrefix)
		permissionSlugAndName := "documents.read.idempotent"
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         permissionSlugAndName,
			Slug:         permissionSlugAndName,
			Description:  dbtype.NullString{Valid: true, String: "Read permission"},
		})
		require.NoError(t, err)

		// Add permission to key
		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permissionSlugAndName},
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

		// Verify permission still exists in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, permissionID, finalPermissions[0].ID)
	})

	t.Run("create permission on-the-fly using slug", func(t *testing.T) {
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

		// Use a slug that doesn't exist yet
		newPermissionSlug := "documents.create.onthefly"
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
		require.Len(t, res.Body.Data, 1)

		// Verify the permission was created with the slug as both name and slug
		require.Equal(t, newPermissionSlug, res.Body.Data[0].Name)

		// Verify permission exists in database and was assigned to key
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, newPermissionSlug, finalPermissions[0].Slug)
		require.Equal(t, newPermissionSlug, finalPermissions[0].Name) // Slug used as name too

		// Verify the permission was created in the workspace
		createdPermission, err := db.Query.FindPermissionBySlugAndWorkspaceID(ctx, h.DB.RO(), db.FindPermissionBySlugAndWorkspaceIDParams{
			Slug:        newPermissionSlug,
			WorkspaceID: workspace.ID,
		})
		require.NoError(t, err)
		require.Equal(t, newPermissionSlug, createdPermission.Slug)
		require.Equal(t, newPermissionSlug, createdPermission.Name)
		require.Equal(t, workspace.ID, createdPermission.WorkspaceID)
	})
}
