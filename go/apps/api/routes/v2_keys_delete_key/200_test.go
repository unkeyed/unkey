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
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestKeyDeleteSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	// Create a test API using testutil helper
	apiName := "Test API"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
		Name:        &apiName,
	})

	// Create a test key for soft delete using testutil helper
	softDeleteKeyName := "test-key"
	softDeleteKeyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &softDeleteKeyName,
	})
	softDeleteKeyID := softDeleteKeyResponse.KeyID

	// Create a test key for hard delete with all relationships using testutil helper
	hardDeleteKeyName := "test-key"
	hardDeleteKeyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &hardDeleteKeyName,
		Permissions: []seed.CreatePermissionRequest{
			{
				WorkspaceID: workspace.ID,
				Name:        "read_data",
				Slug:        "read_data",
			},
		},
		Roles: []seed.CreateRoleRequest{
			{
				WorkspaceID: workspace.ID,
				Name:        "data_admin",
			},
		},
		Ratelimits: []seed.CreateRatelimitRequest{
			{
				WorkspaceID: workspace.ID,
				Name:        "api_calls",
				Limit:       100,
				Duration:    60000,
			},
		},
	})
	hardDeleteKeyID := hardDeleteKeyResponse.KeyID

	// Add encryption to the hard delete key
	encryption, err := h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: workspace.ID,
		Data:    hardDeleteKeyResponse.Key,
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
