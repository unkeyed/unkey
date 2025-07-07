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
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_remove_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestAuthorizationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create test data
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

	permissionID := uid.New(uid.TestPrefix)
	err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		Name:         "documents.read.remove.auth403",
		Slug:         "documents.read.remove.auth403",
		Description:  sql.NullString{Valid: true, String: "Read documents permission"},
	})
	require.NoError(t, err)

	req := handler.Request{
		KeyId: keyID,
		Permissions: []struct {
			Id   *string `json:"id,omitempty"`
			Slug *string `json:"slug,omitempty"`
		}{
			{Id: &permissionID},
		},
	}

	t.Run("root key without required permissions", func(t *testing.T) {
		// Create root key without the required permission
		insufficientRootKey := h.CreateRootKey(workspace.ID, "some.other.permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", insufficientRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("root key with partial permissions", func(t *testing.T) {
		// Create root key with related but insufficient permission
		partialRootKey := h.CreateRootKey(workspace.ID, "api.read.update_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", partialRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("key belongs to different workspace", func(t *testing.T) {
		// Create another workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)
		err := db.Query.InsertWorkspace(ctx, h.DB.RW(), db.InsertWorkspaceParams{
			ID:        otherWorkspaceID,
			OrgID:     uid.New("test_org"),
			Name:      "Other Workspace",
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create keyring in other workspace
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

		// Create key in other workspace
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
		})
		require.NoError(t, err)

		// Create root key for original workspace (authorized for workspace.ID, not otherWorkspaceID)
		authorizedRootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")

		reqWithOtherKey := handler.Request{
			KeyId: otherKeyID,
			Permissions: []struct {
				Id   *string `json:"id,omitempty"`
				Slug *string `json:"slug,omitempty"`
			}{
				{Id: &permissionID},
			},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", authorizedRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			reqWithOtherKey,
		)

		require.Equal(t, 404, res.Status) // Key not found (because it belongs to different workspace)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "key was not found")
	})

	t.Run("permission belongs to different workspace", func(t *testing.T) {
		// Create another workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)
		err := db.Query.InsertWorkspace(ctx, h.DB.RW(), db.InsertWorkspaceParams{
			ID:        otherWorkspaceID,
			OrgID:     uid.New("test_org"),
			Name:      "Other Workspace",
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create permission in other workspace
		otherPermissionID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: otherPermissionID,
			WorkspaceID:  otherWorkspaceID,
			Name:         "other.permission.remove",
			Slug:         "other.permission.remove",
			Description:  sql.NullString{Valid: true, String: "Permission in other workspace"},
		})
		require.NoError(t, err)

		// Create root key for original workspace
		authorizedRootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")

		reqWithOtherPermission := handler.Request{
			KeyId: keyID,
			Permissions: []struct {
				Id   *string `json:"id,omitempty"`
				Slug *string `json:"slug,omitempty"`
			}{
				{Id: &otherPermissionID},
			},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", authorizedRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			reqWithOtherPermission,
		)

		require.Equal(t, 404, res.Status) // Permission not found (because it belongs to different workspace)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	t.Run("root key with no permissions", func(t *testing.T) {
		// Create root key with no permissions
		noPermissionsRootKey := h.CreateRootKey(workspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", noPermissionsRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})
}
