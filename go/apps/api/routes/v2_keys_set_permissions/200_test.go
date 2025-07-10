package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
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

	t.Run("set permissions using permission IDs", func(t *testing.T) {
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
		})
		require.NoError(t, err)

		// Create permissions
		permission1ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.initial",
			Slug:         "documents.read.initial",
			Description:  sql.NullString{Valid: true, String: "Initial permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.write.new",
			Slug:         "documents.write.new",
			Description:  sql.NullString{Valid: true, String: "Write permission"},
		})
		require.NoError(t, err)

		permission3ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission3ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.delete.new",
			Slug:         "documents.delete.new",
			Description:  sql.NullString{Valid: true, String: "Delete permission"},
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
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Id: &permission2ID},
				{Id: &permission3ID},
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

		// Verify response contains new permissions (should be sorted alphabetically by name)
		require.Equal(t, permission3ID, res.Body.Data[0].Id) // documents.delete.new
		require.Equal(t, "documents.delete.new", res.Body.Data[0].Name)
		require.Equal(t, permission2ID, res.Body.Data[1].Id) // documents.write.new
		require.Equal(t, "documents.write.new", res.Body.Data[1].Name)

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
		})
		require.NoError(t, err)

		// Create permissions
		permission1ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.byname",
			Slug:         "documents.read.byname",
			Description:  sql.NullString{Valid: true, String: "Read permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.write.byname",
			Slug:         "documents.write.byname",
			Description:  sql.NullString{Valid: true, String: "Write permission"},
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
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Slug: &[]string{"documents.write.byname"}[0]},
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
		require.Equal(t, permission2ID, res.Body.Data[0].Id)
		require.Equal(t, "documents.write.byname", res.Body.Data[0].Name)

		// Verify permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, permission2ID, finalPermissions[0].ID)
	})

	t.Run("set empty permissions (remove all)", func(t *testing.T) {
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
		})
		require.NoError(t, err)

		// Create and assign permissions
		permission1ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.empty",
			Slug:         "documents.read.empty",
			Description:  sql.NullString{Valid: true, String: "Read permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.write.empty",
			Slug:         "documents.write.empty",
			Description:  sql.NullString{Valid: true, String: "Write permission"},
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
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{},
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
		})
		require.NoError(t, err)

		// Create permission
		permissionID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.idempotent",
			Slug:         "documents.read.idempotent",
			Description:  sql.NullString{Valid: true, String: "Read permission"},
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
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Id: &permissionID},
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
		require.Equal(t, permissionID, res.Body.Data[0].Id)

		// Verify permission still exists in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, permissionID, finalPermissions[0].ID)
	})

	t.Run("create permission on-the-fly using slug", func(t *testing.T) {
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
		})
		require.NoError(t, err)

		// Use a slug that doesn't exist yet
		newPermissionSlug := "documents.create.onthefly"
		createFlag := true

		req := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{
					Slug:   &newPermissionSlug,
					Create: &createFlag,
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
