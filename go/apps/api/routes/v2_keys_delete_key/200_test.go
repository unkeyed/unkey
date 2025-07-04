package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_delete_key"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestKeyDeleteSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	// Create a keyAuth (keyring) for the API
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            keyAuthID,
		WorkspaceID:   workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	// Create a test API
	apiID := uid.New("api")
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "Test API",
		WorkspaceID: workspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	softDeleteKeyID := uid.New(uid.KeyPrefix)
	softDeletekey, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     "test",
		ByteLength: 16,
	})

	err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
		ID:                softDeleteKeyID,
		KeyringID:         keyAuthID,
		Hash:              softDeletekey.Hash,
		Start:             softDeletekey.Start,
		WorkspaceID:       workspace.ID,
		ForWorkspaceID:    sql.NullString{Valid: false},
		Name:              sql.NullString{Valid: true, String: "test-key"},
		Expires:           sql.NullTime{Valid: false},
		CreatedAtM:        time.Now().UnixMilli(),
		Enabled:           true,
		IdentityID:        sql.NullString{Valid: false, String: ""},
		RemainingRequests: sql.NullInt32{Int32: 0, Valid: false},
	})
	require.NoError(t, err)

	hardDeleteKeyID := uid.New(uid.KeyPrefix)
	hardDeleteKey, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     "test",
		ByteLength: 16,
	})

	err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
		ID:                hardDeleteKeyID,
		KeyringID:         keyAuthID,
		Hash:              hardDeleteKey.Hash,
		Start:             hardDeleteKey.Start,
		WorkspaceID:       workspace.ID,
		ForWorkspaceID:    sql.NullString{Valid: false},
		Name:              sql.NullString{Valid: true, String: "test-key"},
		Expires:           sql.NullTime{Valid: false},
		CreatedAtM:        time.Now().UnixMilli(),
		Enabled:           true,
		IdentityID:        sql.NullString{Valid: false, String: ""},
		RemainingRequests: sql.NullInt32{Int32: 0, Valid: false},
	})
	require.NoError(t, err)

	encryption, err := h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: workspace.ID,
		Data:    hardDeleteKey.Key,
	})
	require.NoError(t, err)

	err = db.Query.InsertKeyEncryption(ctx, h.DB.RW(), db.InsertKeyEncryptionParams{
		WorkspaceID:     workspace.ID,
		KeyID:           hardDeleteKeyID,
		CreatedAt:       time.Now().UnixMilli(),
		Encrypted:       encryption.GetEncrypted(),
		EncryptionKeyID: encryption.GetKeyId(),
	})
	require.NoError(t, err)

	// Create permissions
	perm1ID := uid.New(uid.PermissionPrefix)
	err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: perm1ID,
		WorkspaceID:  workspace.ID,
		Name:         "read_data",
		Slug:         "read_data",
		CreatedAtM:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Assign permissions to key
	err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
		KeyID:        hardDeleteKeyID,
		PermissionID: perm1ID,
		WorkspaceID:  workspace.ID,
	})
	require.NoError(t, err)

	roleID := uid.New(uid.RolePrefix)
	err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: workspace.ID,
		Name:        "data_admin",
	})
	require.NoError(t, err)

	// Assign role to key
	err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
		KeyID:       hardDeleteKeyID,
		RoleID:      roleID,
		WorkspaceID: workspace.ID,
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create ratelimits for the key
	rl1ID := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertKeyRatelimit(ctx, h.DB.RW(), db.InsertKeyRatelimitParams{
		ID:          rl1ID,
		WorkspaceID: workspace.ID,
		KeyID:       sql.NullString{Valid: true, String: hardDeleteKeyID},
		Name:        "api_calls",
		Limit:       100,
		Duration:    60000, // 1 minute
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.delete_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("soft delete key", func(t *testing.T) {
		now := time.Now().UnixMilli()
		req := handler.Request{
			KeyId: softDeleteKeyID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), softDeleteKeyID)
		require.NoError(t, err)
		require.NotNil(t, key)
		require.Equal(t, key.DeletedAtM.Valid, true)
		require.Greater(t, key.DeletedAtM.Int64, now)
	})

	t.Run("permanently delete key", func(t *testing.T) {
		req := handler.Request{
			KeyId:     hardDeleteKeyID,
			Permanent: ptr.P(true),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.NoError(t, err)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		_, err := db.Query.FindKeyByID(ctx, h.DB.RO(), hardDeleteKeyID)
		require.Equal(t, sql.ErrNoRows, err)

		ratelimits, err := db.Query.ListRatelimitsByKeyID(ctx, h.DB.RO(), sql.NullString{String: hardDeleteKeyID, Valid: true})
		require.NoError(t, err)
		require.Len(t, ratelimits, 0)

		roles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), hardDeleteKeyID)
		require.NoError(t, err)
		require.Len(t, roles, 0)

		permissions, err := db.Query.ListPermissionsByKeyID(ctx, h.DB.RO(), db.ListPermissionsByKeyIDParams{
			KeyID: hardDeleteKeyID,
		})
		require.NoError(t, err)
		require.Len(t, permissions, 0)

		_, err = db.Query.FindKeyEncryptionByKeyID(ctx, h.DB.RO(), hardDeleteKeyID)
		require.Equal(t, sql.ErrNoRows, err)
	})
}
