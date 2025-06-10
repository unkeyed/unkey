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
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	// Create a workspace and root key
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("key not found", func(t *testing.T) {
		// Create a permission that exists
		permissionID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.404keynotfound",
			Slug:         "documents.read.404keynotfound",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		// Use a non-existent key ID
		nonExistentKeyID := uid.New(uid.KeyPrefix)

		req := handler.Request{
			KeyId: nonExistentKeyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Name   *string `json:"name,omitempty"`
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
		require.Contains(t, res.Body.Error.Detail, "key was not found")
	})

	t.Run("permission not found by ID", func(t *testing.T) {
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
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Use a non-existent permission ID
		nonExistentPermissionID := uid.New(uid.TestPrefix)

		req := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Name   *string `json:"name,omitempty"`
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
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	t.Run("permission not found by name", func(t *testing.T) {
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
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Use a non-existent permission name
		nonExistentPermissionName := "nonexistent.permission.name"

		req := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Name   *string `json:"name,omitempty"`
			}{
				{Name: &nonExistentPermissionName},
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
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	t.Run("permission from different workspace by ID", func(t *testing.T) {
		// Create another workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)
		err := db.Query.InsertWorkspace(ctx, h.DB.RW(), db.InsertWorkspaceParams{
			ID:        otherWorkspaceID,
			OrgID:     uid.New("test_org"),
			Name:      "Other Workspace",
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a permission in the other workspace
		otherPermissionID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: otherPermissionID,
			WorkspaceID:  otherWorkspaceID,
			Name:         "other.workspace.permission.404",
			Slug:         "other.workspace.permission.404",
			Description:  sql.NullString{Valid: true, String: "Permission in other workspace"},
		})
		require.NoError(t, err)

		// Create a test keyring in our workspace
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err = db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        workspace.ID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test key in our workspace
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
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Name   *string `json:"name,omitempty"`
			}{
				{Id: &otherPermissionID},
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
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	t.Run("key from different workspace", func(t *testing.T) {
		// Create another workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)
		err := db.Query.InsertWorkspace(ctx, h.DB.RW(), db.InsertWorkspaceParams{
			ID:        otherWorkspaceID,
			OrgID:     uid.New("test_org"),
			Name:      "Other Workspace",
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test keyring in the other workspace
		otherKeyAuthID := uid.New(uid.KeyAuthPrefix)
		err = db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 otherKeyAuthID,
			WorkspaceID:        otherWorkspaceID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test key in the other workspace
		otherKeyID := uid.New(uid.KeyPrefix)
		otherKeyString := "test_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                otherKeyID,
			KeyringID:         otherKeyAuthID,
			Hash:              hash.Sha256(otherKeyString),
			Start:             otherKeyString[:4],
			WorkspaceID:       otherWorkspaceID,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Name:              sql.NullString{Valid: true, String: "Other Workspace Key"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			IdentityID:        sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},
			RatelimitAsync:    sql.NullBool{Valid: false},
			RatelimitLimit:    sql.NullInt32{Valid: false},
			RatelimitDuration: sql.NullInt64{Valid: false},
			Environment:       sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Create a permission in our workspace
		permissionID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.404keydifferentws",
			Slug:         "documents.read.404keydifferentws",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: otherKeyID,
			Permissions: []struct {
				Create *bool   `json:"create,omitempty"`
				Id     *string `json:"id,omitempty"`
				Name   *string `json:"name,omitempty"`
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
		require.Contains(t, res.Body.Error.Detail, "key was not found")
	})
}
