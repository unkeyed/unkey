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
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFound(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
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

	t.Run("non-existent key ID", func(t *testing.T) {
		// Create a permission to reference
		permissionID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.notfound",
			Slug:         "documents.read.notfound",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		// Use non-existent key ID
		nonExistentKeyID := uid.New(uid.KeyPrefix)

		req := handler.Request{
			KeyId: nonExistentKeyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Id: &permissionID},
			},
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

	t.Run("non-existent permission ID", func(t *testing.T) {
		// Create a test keyring and key
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

		// Use non-existent permission ID
		nonExistentPermissionID := uid.New(uid.TestPrefix)

		req := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Id: &nonExistentPermissionID},
			},
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
		require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Permission with ID '%s' was not found", nonExistentPermissionID))
	})

	t.Run("non-existent permission name", func(t *testing.T) {
		// Create a test keyring and key
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

		nonExistentPermissionName := "nonexistent.permission.name"

		req := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Slug: &nonExistentPermissionName},
			},
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
		require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Permission with slug '%s' was not found", nonExistentPermissionName))
	})

	t.Run("key from different workspace (isolation)", func(t *testing.T) {
		// Create another workspace
		otherWorkspace := h.CreateWorkspace()

		// Create keyring and key in the other workspace
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        otherWorkspace.ID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		keyID := uid.New(uid.KeyPrefix)
		keyString := "test_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              hash.Sha256(keyString),
			Start:             keyString[:4],
			WorkspaceID:       otherWorkspace.ID, // Different workspace
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

		// Create permission in the authorized workspace
		permissionID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.isolation",
			Slug:         "documents.read.isolation",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID, // Key from different workspace
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Id: &permissionID},
			},
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

	t.Run("permission from different workspace (isolation)", func(t *testing.T) {
		// Create another workspace
		otherWorkspace := h.CreateWorkspace()

		// Create a test keyring and key in the authorized workspace
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

		// Create permission in the other workspace
		permissionID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  otherWorkspace.ID, // Different workspace
			Name:         "documents.read.otherworkspace",
			Slug:         "documents.read.otherworkspace",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Id: &permissionID}, // Permission from different workspace
			},
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
		require.Contains(t, res.Body.Error.Detail, "Permission 'documents.read.otherworkspace' was not found")
	})

	t.Run("multiple permissions with early failure", func(t *testing.T) {
		// Create a test keyring and key
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

		// Create a valid permission for the second item
		validPermissionID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: validPermissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.valid",
			Slug:         "documents.read.valid",
			Description:  sql.NullString{Valid: true, String: "Valid permission"},
		})
		require.NoError(t, err)

		// Use non-existent permission ID as first item
		nonExistentPermissionID := uid.New(uid.TestPrefix)

		req := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Slug   *string `json:"slug,omitempty"`
			}{
				{Id: &nonExistentPermissionID}, // This should fail first
				{Id: &validPermissionID},       // This should not be processed
			},
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
		require.Contains(t, res.Body.Error.Detail, fmt.Sprintf("Permission with ID '%s' was not found", nonExistentPermissionID))
	})
}
