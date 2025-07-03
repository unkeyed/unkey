package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_get_key"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestGetKeyByKeyID(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
		Vault:       h.Vault,
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

	err = db.Query.UpdateKeyringKeyEncryption(ctx, h.DB.RW(), db.UpdateKeyringKeyEncryptionParams{
		ID:                 keyAuthID,
		StoreEncryptedKeys: true,
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

	// Create test identities
	identityID := uid.New("identity")
	identityExternalID := "test_user"
	err = db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  identityExternalID,
		WorkspaceID: workspace.ID,
		Environment: "",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte(`{"role": "admin"}`),
	})
	require.NoError(t, err)

	ratelimitID := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
		ID:          ratelimitID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		IdentityID:  sql.NullString{String: identityID, Valid: true},
		Name:        "api_calls",
		Limit:       100,
		Duration:    60000, // 1 minute
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	keyID := uid.New(uid.KeyPrefix)
	key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     "test",
		ByteLength: 16,
	})

	insertParams := db.InsertKeyParams{
		ID:             keyID,
		KeyringID:      keyAuthID,
		Hash:           key.Hash,
		Start:          key.Start,
		WorkspaceID:    workspace.ID,
		ForWorkspaceID: sql.NullString{Valid: false},
		Name:           sql.NullString{Valid: true, String: "test-key"},
		Expires:        sql.NullTime{Valid: false},
		CreatedAtM:     time.Now().UnixMilli(),
		Enabled:        true,
		IdentityID:     sql.NullString{Valid: true, String: identityID},
	}

	err = db.Query.InsertKey(ctx, h.DB.RW(), insertParams)
	require.NoError(t, err)

	encryption, err := h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: workspace.ID,
		Data:    key.Key,
	})
	require.NoError(t, err)

	err = db.Query.InsertKeyEncryption(ctx, h.DB.RW(), db.InsertKeyEncryptionParams{
		WorkspaceID:     workspace.ID,
		KeyID:           keyID,
		CreatedAt:       time.Now().UnixMilli(),
		Encrypted:       encryption.GetEncrypted(),
		EncryptionKeyID: encryption.GetKeyId(),
	})
	require.NoError(t, err)

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.decrypt_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// This also tests that we have the correct data for the key.
	t.Run("get key by keyId without decrypting", func(t *testing.T) {
		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.KeyId, keyID)
	})

	t.Run("get key by keyId with decrypting", func(t *testing.T) {
		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(true),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, ptr.SafeDeref(res.Body.Data.Plaintext), key.Key)
	})

	t.Run("get key by plaintext key", func(t *testing.T) {
		req := handler.Request{
			Key: ptr.P(key.Key),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.KeyId, keyID)
	})

	t.Run("get key by plaintext key with decrypting", func(t *testing.T) {
		req := handler.Request{
			Key:     ptr.P(key.Key),
			Decrypt: ptr.P(true),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, ptr.SafeDeref(res.Body.Data.Plaintext), key.Key)
	})
}

func TestGetKey_AdditionalScenarios(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
		Vault:       h.Vault,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// Create keyAuth (keyring) for the API
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            keyAuthID,
		WorkspaceID:   workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	// Create test API
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

	// Create root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("key with complex meta data", func(t *testing.T) {
		keyID := uid.New(uid.KeyPrefix)
		key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
			Prefix:     "test",
			ByteLength: 16,
		})

		complexMeta := map[string]interface{}{
			"user_id":    12345,
			"plan":       "premium",
			"features":   []string{"analytics", "webhooks"},
			"created_by": "admin@example.com",
			"nested": map[string]string{
				"department": "engineering",
				"team":       "backend",
			},
		}
		metaBytes, _ := json.Marshal(complexMeta)

		err := db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			KeyringID:   keyAuthID,
			Hash:        key.Hash,
			Start:       key.Start,
			WorkspaceID: workspace.ID,
			Name:        sql.NullString{Valid: true, String: "complex-meta-key"},
			Meta:        sql.NullString{Valid: true, String: string(metaBytes)},
			CreatedAtM:  time.Now().UnixMilli(),
			Enabled:     true,
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Meta)

		// Verify meta data was properly unmarshaled
		metaMap := *res.Body.Data.Meta
		require.Equal(t, float64(12345), metaMap["user_id"]) // JSON numbers become float64
		require.Equal(t, "premium", metaMap["plan"])
	})

	t.Run("key with expiration date", func(t *testing.T) {
		keyID := uid.New(uid.KeyPrefix)
		key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
			Prefix:     "test",
			ByteLength: 16,
		})

		futureDate := time.Now().Add(24 * time.Hour).Truncate(time.Hour)
		err := db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			KeyringID:   keyAuthID,
			Hash:        key.Hash,
			Start:       key.Start,
			WorkspaceID: workspace.ID,
			Name:        sql.NullString{Valid: true, String: "expiring-key"},
			Expires:     sql.NullTime{Valid: true, Time: futureDate},
			CreatedAtM:  time.Now().UnixMilli(),
			Enabled:     true,
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Expires)
		require.Equal(t, futureDate.UnixMilli(), *res.Body.Data.Expires)
	})

	t.Run("key with credits and daily refill", func(t *testing.T) {
		keyID := uid.New(uid.KeyPrefix)
		key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
			Prefix:     "test",
			ByteLength: 16,
		})

		err := db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              key.Hash,
			Start:             key.Start,
			WorkspaceID:       workspace.ID,
			Name:              sql.NullString{Valid: true, String: "credits-key"},
			RemainingRequests: sql.NullInt32{Valid: true, Int32: 50},
			RefillAmount:      sql.NullInt32{Valid: true, Int32: 100},
			RefillDay:         sql.NullInt16{Valid: false, Int16: 0},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Credits)
		require.Equal(t, nullable.NewNullableWithValue(int64(50)), res.Body.Data.Credits.Remaining)
		require.NotNil(t, res.Body.Data.Credits.Refill)
		require.Equal(t, int64(100), res.Body.Data.Credits.Refill.Amount)
		require.Equal(t, "daily", string(res.Body.Data.Credits.Refill.Interval))
	})

	t.Run("key with monthly refill", func(t *testing.T) {
		keyID := uid.New(uid.KeyPrefix)
		key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
			Prefix:     "test",
			ByteLength: 16,
		})

		err := db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         keyAuthID,
			Hash:              key.Hash,
			Start:             key.Start,
			WorkspaceID:       workspace.ID,
			Name:              sql.NullString{Valid: true, String: "monthly-refill-key"},
			RemainingRequests: sql.NullInt32{Valid: true, Int32: 1000},
			RefillAmount:      sql.NullInt32{Valid: true, Int32: 2000},
			RefillDay:         sql.NullInt16{Valid: true, Int16: 1}, // 1st of month
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Credits)
		require.NotNil(t, res.Body.Data.Credits.Refill)
		require.Equal(t, "monthly", string(res.Body.Data.Credits.Refill.Interval))
		require.Equal(t, 1, *res.Body.Data.Credits.Refill.RefillDay)
	})

	t.Run("key with roles and permissions", func(t *testing.T) {
		keyID := uid.New(uid.KeyPrefix)
		key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
			Prefix:     "test",
			ByteLength: 16,
		})

		err := db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			KeyringID:   keyAuthID,
			Hash:        key.Hash,
			Start:       key.Start,
			WorkspaceID: workspace.ID,
			Name:        sql.NullString{Valid: true, String: "rbac-key"},
			CreatedAtM:  time.Now().UnixMilli(),
			Enabled:     true,
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

		perm2ID := uid.New(uid.PermissionPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: perm2ID,
			WorkspaceID:  workspace.ID,
			Name:         "write_data",
			Slug:         "write_data",
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create role
		roleID := uid.New(uid.RolePrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "data_admin",
		})
		require.NoError(t, err)

		// Assign permissions to key
		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: perm1ID,
			WorkspaceID:  workspace.ID,
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: perm2ID,
			WorkspaceID:  workspace.ID,
		})
		require.NoError(t, err)

		// Assign role to key
		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Permissions)
		require.NotNil(t, res.Body.Data.Roles)

		permissions := *res.Body.Data.Permissions
		require.Len(t, permissions, 2)
		require.Contains(t, permissions, "read_data")
		require.Contains(t, permissions, "write_data")

		roles := *res.Body.Data.Roles
		require.Len(t, roles, 1)
		require.Contains(t, roles, "data_admin")
	})

	t.Run("key with ratelimits", func(t *testing.T) {
		keyID := uid.New(uid.KeyPrefix)
		key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
			Prefix:     "test",
			ByteLength: 16,
		})

		err := db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			KeyringID:   keyAuthID,
			Hash:        key.Hash,
			Start:       key.Start,
			WorkspaceID: workspace.ID,
			Name:        sql.NullString{Valid: true, String: "ratelimited-key"},
			CreatedAtM:  time.Now().UnixMilli(),
			Enabled:     true,
		})
		require.NoError(t, err)

		// Create ratelimits for the key
		rl1ID := uid.New(uid.RatelimitPrefix)
		err = db.Query.InsertKeyRatelimit(ctx, h.DB.RW(), db.InsertKeyRatelimitParams{
			ID:          rl1ID,
			WorkspaceID: workspace.ID,
			KeyID:       sql.NullString{Valid: true, String: keyID},
			Name:        "api_calls",
			Limit:       100,
			Duration:    60000, // 1 minute
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		rl2ID := uid.New(uid.RatelimitPrefix)
		err = db.Query.InsertKeyRatelimit(ctx, h.DB.RW(), db.InsertKeyRatelimitParams{
			ID:          rl2ID,
			WorkspaceID: workspace.ID,
			KeyID:       sql.NullString{Valid: true, String: keyID},
			Name:        "data_transfer",
			Limit:       1000,
			Duration:    3600000, // 1 hour
			AutoApply:   true,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Ratelimits)

		ratelimits := *res.Body.Data.Ratelimits
		require.Len(t, ratelimits, 2)

		// Find each ratelimit and verify
		var apiCallsRL, dataTransferRL *openapi.RatelimitResponse
		for _, rl := range ratelimits {
			if rl.Name == "api_calls" {
				apiCallsRL = &rl
			} else if rl.Name == "data_transfer" {
				dataTransferRL = &rl
			}
		}

		require.NotNil(t, apiCallsRL)
		require.Equal(t, int64(100), apiCallsRL.Limit)
		require.Equal(t, int64(60000), apiCallsRL.Duration)
		require.False(t, apiCallsRL.AutoApply)

		require.NotNil(t, dataTransferRL)
		require.Equal(t, int64(1000), dataTransferRL.Limit)
		require.Equal(t, int64(3600000), dataTransferRL.Duration)
		require.True(t, dataTransferRL.AutoApply)
	})

	t.Run("disabled key", func(t *testing.T) {
		keyID := uid.New(uid.KeyPrefix)
		key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
			Prefix:     "test",
			ByteLength: 16,
		})

		err := db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			KeyringID:   keyAuthID,
			Hash:        key.Hash,
			Start:       key.Start,
			WorkspaceID: workspace.ID,
			Name:        sql.NullString{Valid: true, String: "disabled-key"},
			CreatedAtM:  time.Now().UnixMilli(),
			Enabled:     false, // Key is disabled
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.False(t, res.Body.Data.Enabled)
	})
}
