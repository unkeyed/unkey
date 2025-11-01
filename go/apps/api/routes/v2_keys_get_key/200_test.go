package handler_test

import (
	"context"
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
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestGetKeyByKeyID(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:       h.Logger,
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		Vault:        h.Vault,
		LiveKeyCache: h.Caches.LiveKeyByID,
	}

	h.Register(route)

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	// Create a test API with encrypted keys using testutil helper
	apiName := "Test API"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		Name:          &apiName,
		EncryptedKeys: true,
	})

	// Create test identity with ratelimit using testutil helper
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "test_user",
		Meta:        []byte(`{"role": "admin"}`),
		Ratelimits: []seed.CreateRatelimitRequest{
			{
				WorkspaceID: workspace.ID,
				Name:        "api_calls",
				Limit:       100,
				Duration:    60000,
			},
		},
	})

	// Create test key with identity and encryption using testutil helper
	keyName := "test-key"
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
		IdentityID:  &identity.ID,
	})
	keyID := key.KeyID
	// key := keyResponse.Key

	// Add encryption for the key since API has encrypted keys enabled
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
			KeyId:   keyID,
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.KeyId, keyID)
	})

	t.Run("get key by keyId with decrypting", func(t *testing.T) {
		req := handler.Request{
			KeyId:   keyID,
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
	route := &handler.Handler{
		Logger:       h.Logger,
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		Vault:        h.Vault,
		LiveKeyCache: h.Caches.LiveKeyByID,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// Create test API using testutil helper
	apiName := "Test API"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
		Name:        &apiName,
	})

	// Create root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("key with complex meta data", func(t *testing.T) {
		// Create test key with complex meta using testutil helper
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
		metaString := string(metaBytes)
		keyName := "complex-meta-key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
			Meta:        &metaString,
		})
		keyID := keyResponse.KeyID

		req := handler.Request{
			KeyId:   keyID,
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
		futureDate := time.Now().Add(24 * time.Hour).Truncate(time.Hour)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Expires:     &futureDate,
		})

		req := handler.Request{
			KeyId:   keyResponse.KeyID,
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Expires)
		require.Equal(t, futureDate.UnixMilli(), *res.Body.Data.Expires)
	})

	t.Run("key with credits and daily refill", func(t *testing.T) {
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:  workspace.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    ptr.P(int32(50)),
			RefillAmount: ptr.P(int32(100)),
		})

		req := handler.Request{
			KeyId:   keyResponse.KeyID,
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
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:  workspace.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    ptr.P(int32(50)),
			RefillAmount: ptr.P(int32(100)),
			RefillDay:    ptr.P(int16(1)),
		})

		req := handler.Request{
			KeyId:   keyResponse.KeyID,
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
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Permissions: []seed.CreatePermissionRequest{{
				Name:        "read_data",
				Slug:        "read_data",
				Description: nil,
				WorkspaceID: workspace.ID,
			}, {
				Name:        "write_data",
				Slug:        "write_data",
				Description: nil,
				WorkspaceID: workspace.ID,
			}},
			Roles: []seed.CreateRoleRequest{{
				Name:        "data_admin",
				Description: nil,
				WorkspaceID: workspace.ID,
			}},
		})

		req := handler.Request{
			KeyId:   keyResponse.KeyID,
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
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "api_calls",
					WorkspaceID: workspace.ID,
					AutoApply:   false,
					Duration:    60000, // 1minute
					Limit:       100,
					IdentityID:  nil,
					KeyID:       nil,
				},
				{
					Name:        "data_transfer",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    3600000, // 1 hour
					Limit:       1000,
					IdentityID:  nil,
					KeyID:       nil,
				},
			},
		})

		req := handler.Request{
			KeyId:   keyResponse.KeyID,
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
			switch rl.Name {
			case "api_calls":
				apiCallsRL = &rl
			case "data_transfer":
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
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Disabled:    true,
		})

		req := handler.Request{
			KeyId:   keyResponse.KeyID,
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.False(t, res.Body.Data.Enabled)
	})
}
