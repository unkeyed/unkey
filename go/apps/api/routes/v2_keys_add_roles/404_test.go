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
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
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

	// Test case for key not found
	t.Run("key not found", func(t *testing.T) {
		req := handler.Request{
			KeyId: "key_nonexistent123456789", // Non-existent key ID
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: func() *string { s := "role_123"; return &s }()},
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

	// Test case for key in different workspace
	t.Run("key in different workspace", func(t *testing.T) {
		// Create a second workspace with a key
		workspace2 := h.CreateWorkspace()

		// Create a test keyring in workspace2
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        workspace2.ID,
			StoreEncryptedKeys: false,
			DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			CreatedAtM:         time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a test key in workspace2
		keyID := uid.New(uid.KeyPrefix)
		keyString := "test_" + uid.New("")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              hash.Sha256(keyString),
			Start:             keyString[:4],
			WorkspaceID:       workspace2.ID,
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
			KeyId: keyID, // Key from workspace2, but accessed with workspace1 root key
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: func() *string { s := "role_123"; return &s }()},
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

	// Test case for role not found by ID
	t.Run("role not found by ID", func(t *testing.T) {
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

		nonExistentRoleId := "role_nonexistent123456789"
		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &nonExistentRoleId}, // Non-existent role ID
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
		require.Contains(t, res.Body.Error.Detail, "Role with ID")
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for role not found by name
	t.Run("role not found by name", func(t *testing.T) {
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

		nonExistentRoleName := "nonexistent_role"
		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Name: &nonExistentRoleName}, // Non-existent role name
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
		require.Contains(t, res.Body.Error.Detail, "Role with name")
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for role in different workspace
	t.Run("role in different workspace", func(t *testing.T) {
		// Create a second workspace with a role
		workspace2 := h.CreateWorkspace()
		roleID := uid.New(uid.TestPrefix)
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace2.ID,
			Name:        "admin_diff_workspace",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		// Create a test keyring in workspace1
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

		// Create a test key in workspace1
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
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &roleID}, // Role from different workspace
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
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for role by name in different workspace
	t.Run("role by name in different workspace", func(t *testing.T) {
		// Create a second workspace with a role
		workspace2 := h.CreateWorkspace()
		roleID := uid.New(uid.TestPrefix)
		roleName := "admin_by_name_diff_workspace"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace2.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		// Create a test keyring in workspace1
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

		// Create a test key in workspace1
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
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Name: &roleName}, // Role by name from different workspace
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
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for multiple roles with one not found
	t.Run("multiple roles one not found", func(t *testing.T) {
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

		// Create one valid role
		validRoleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      validRoleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_multiple_roles",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		invalidRoleId := "role_nonexistent123456789"

		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &validRoleID},   // Valid role
				{Id: &invalidRoleId}, // Invalid role - should cause 404
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
		require.Contains(t, res.Body.Error.Detail, "Role with ID")
		require.Contains(t, res.Body.Error.Detail, invalidRoleId)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	// Test case for deleted key
	t.Run("deleted key", func(t *testing.T) {
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

		// Create a role
		roleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_deleted_key",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		// Use non-existent key ID to simulate deleted key
		deletedKeyID := "key_deleted123456789"

		req := handler.Request{
			KeyId: deletedKeyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &roleID},
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

	// Test case for deleted role
	t.Run("deleted role", func(t *testing.T) {
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

		// Create a role
		roleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "admin_deleted_role",
			Description: sql.NullString{Valid: true, String: "Admin role"},
		})
		require.NoError(t, err)

		// Use non-existent role ID to simulate deleted role
		deletedRoleID := "role_deleted123456789"

		req := handler.Request{
			KeyId: keyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{Id: &deletedRoleID},
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
		require.Contains(t, res.Body.Error.Detail, "Role with ID")
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})
}
