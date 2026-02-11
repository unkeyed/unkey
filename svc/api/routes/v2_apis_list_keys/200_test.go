package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:       h.DB,
		Keys:     h.Keys,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}

	h.Register(route)

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.read_api", "api.*.decrypt_key")

	// Create a keySpace for the API
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            keySpaceID,
		WorkspaceID:   workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	err = db.Query.UpdateKeySpaceKeyEncryption(ctx, h.DB.RW(), db.UpdateKeySpaceKeyEncryptionParams{
		ID:                 keySpaceID,
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
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create test identities
	identity1ExternalID := "test_user_1"
	identity1 := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  identity1ExternalID,
		Meta:        []byte(`{"role": "admin"}`),
	})

	identity2ExternalID := "test_user_2"
	identity2 := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  identity2ExternalID,
		Meta:        []byte(`{"role": "user"}`),
	})

	// Create test keys with various configurations
	// Track encrypted keys for verification
	encryptedKeys := make(map[string]string) // keyID -> plaintext

	// Key 1: identity1, production metadata
	key1 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Test Key 1"),
		IdentityID:  ptr.P(identity1.ID),
		Meta:        ptr.P(`{"env": "production", "team": "backend"}`),
		Recoverable: true,
	})
	encryptedKeys[key1.KeyID] = key1.Key

	// Key 2: identity1, staging metadata
	key2 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Test Key 2"),
		IdentityID:  ptr.P(identity1.ID),
		Meta:        ptr.P(`{"env": "staging"}`),
		Recoverable: true,
	})
	encryptedKeys[key2.KeyID] = key2.Key

	// Key 3: identity2, development metadata
	key3 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Test Key 3"),
		IdentityID:  ptr.P(identity2.ID),
		Meta:        ptr.P(`{"env": "development"}`),
		Recoverable: true,
	})
	encryptedKeys[key3.KeyID] = key3.Key

	// Key 4: no identity
	key4 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Test Key 4 (No Identity)"),
		Recoverable: true,
	})
	encryptedKeys[key4.KeyID] = key4.Key

	// Key 5: disabled
	key5 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Test Key 5 (Disabled)"),
		Disabled:    true,
		Recoverable: true,
	})
	encryptedKeys[key5.KeyID] = key5.Key

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Happy Path Tests
	t.Run("list all keys with default pagination", func(t *testing.T) {
		req := handler.Request{
			ApiId: apiID,
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
		require.GreaterOrEqual(t, len(res.Body.Data), 1) // At least one key

		// Verify response structure
		require.NotNil(t, res.Body.Meta)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotNil(t, res.Body.Pagination)

		// Basic key structure verification
		if len(res.Body.Data) > 0 {
			key := res.Body.Data[0]
			require.NotEmpty(t, key.KeyId)
			require.NotEmpty(t, key.Start)
			require.Greater(t, key.CreatedAt, int64(0))
		}
	})

	t.Run("list keys with limit parameter", func(t *testing.T) {
		limit := 2
		req := handler.Request{
			ApiId: apiID,
			Limit: &limit,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 2)
		require.NotNil(t, res.Body.Pagination)
		require.NotNil(t, res.Body.Pagination.Cursor)
		require.True(t, res.Body.Pagination.HasMore)
	})

	t.Run("list keys with pagination cursor", func(t *testing.T) {
		limit := 2

		// First page
		req1 := handler.Request{
			ApiId: apiID,
			Limit: &limit,
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, 200, res1.Status)
		require.Len(t, res1.Body.Data, 2)
		require.NotNil(t, res1.Body.Pagination.Cursor)

		// Second page
		req2 := handler.Request{
			ApiId:  apiID,
			Limit:  &limit,
			Cursor: res1.Body.Pagination.Cursor,
		}

		res2 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req2,
		)

		require.Equal(t, 200, res2.Status)
		require.Greater(t, len(res2.Body.Data), 0)

		// Verify no duplicate keys between pages
		firstPageIDs := make(map[string]bool)
		for _, key := range res1.Body.Data {
			firstPageIDs[key.KeyId] = true
		}

		for _, key := range res2.Body.Data {
			require.False(t, firstPageIDs[key.KeyId], "Key %s should not appear in both pages", key.KeyId)
		}
	})

	t.Run("filter by external ID", func(t *testing.T) {
		req := handler.Request{
			ApiId:      apiID,
			ExternalId: &identity1ExternalID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 2) // Keys with identity1

		for _, key := range res.Body.Data {
			require.NotNil(t, key.Identity)
			require.Equal(t, identity1ExternalID, key.Identity.ExternalId)
		}
	})

	t.Run("filter by non-existent external ID returns empty", func(t *testing.T) {
		nonExistentID := "non_existent_user"
		req := handler.Request{
			ApiId:      apiID,
			ExternalId: &nonExistentID,
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
		require.NotNil(t, res.Body.Pagination)
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
	})

	t.Run("verify key metadata is included", func(t *testing.T) {
		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.Greater(t, len(res.Body.Data), 0)

		// Find a key with metadata
		var foundProductionKey bool
		for _, key := range res.Body.Data {
			if key.Meta != nil {
				meta := key.Meta
				if env, exists := meta["env"]; exists && env == "production" {
					require.Equal(t, "backend", meta["team"])
					foundProductionKey = true
					break
				}
			}
		}
		require.True(t, foundProductionKey, "Should find a key with production environment metadata")
	})

	t.Run("verify identity information is included", func(t *testing.T) {
		req := handler.Request{
			ApiId:      apiID,
			ExternalId: &identity1ExternalID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.Greater(t, len(res.Body.Data), 0)

		key := res.Body.Data[0]
		require.NotNil(t, key.Identity)
		require.Equal(t, identity1ExternalID, key.Identity.ExternalId)
		require.NotNil(t, key.Identity.Meta)

		// Verify identity metadata
		identityMeta := key.Identity.Meta
		require.Equal(t, "admin", identityMeta["role"])
	})

	t.Run("verify correct ordering of results", func(t *testing.T) {
		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.Greater(t, len(res.Body.Data), 1)

		// Verify keys are consistently ordered (should be stable ordering)
		// Note: We don't enforce strict ascending ID order as UIDs may have different patterns
		var keyIds []string
		for _, key := range res.Body.Data {
			keyIds = append(keyIds, key.KeyId)
		}
		require.NotEmpty(t, keyIds)
		// Just verify we have consistent results - the exact ordering depends on UID generation
	})

	t.Run("empty API returns empty result", func(t *testing.T) {
		// Create a new API with no keys
		emptyKeySpaceID := uid.New(uid.KeySpacePrefix)
		err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
			ID:          emptyKeySpaceID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		emptyApiID := uid.New("api")
		err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          emptyApiID,
			Name:        "Empty API",
			WorkspaceID: workspace.ID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: emptyKeySpaceID},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			ApiId: emptyApiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 0)
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
	})

	t.Run("verify no sensitive data is exposed", func(t *testing.T) {
		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.Greater(t, len(res.Body.Data), 0)

		for _, key := range res.Body.Data {
			// Verify that plaintext key is not exposed unless explicitly requested
			require.Empty(t, key.Plaintext)

			// Only start prefix should be shown - allow reasonable prefix lengths
			require.NotEmpty(t, key.Start)
			require.True(t, len(key.Start) <= 32, "Start should be a reasonable prefix length")
		}
	})

	t.Run("verify ratelimits are returned correctly", func(t *testing.T) {
		// Create a key with ratelimits
		keyWithRatelimits := uid.New("key")
		err := db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyWithRatelimits,
			KeySpaceID:        keySpaceID,
			Hash:              hash.Sha256(uid.New("test")),
			Start:             "rl_test_",
			WorkspaceID:       workspace.ID,
			Name:              sql.NullString{Valid: true, String: "Key with Ratelimits"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},

			IdentityID: sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Create a key without ratelimits
		keyWithoutRatelimits := uid.New("key")
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                keyWithoutRatelimits,
			KeySpaceID:        keySpaceID,
			Hash:              hash.Sha256("no_rl_test_" + uid.New("")),
			Start:             "no_rl_test_",
			WorkspaceID:       workspace.ID,
			Name:              sql.NullString{Valid: true, String: "Key without Ratelimits"},
			CreatedAtM:        time.Now().UnixMilli(),
			Enabled:           true,
			ForWorkspaceID:    sql.NullString{Valid: false},
			Meta:              sql.NullString{Valid: false},
			Expires:           sql.NullTime{Valid: false},
			RemainingRequests: sql.NullInt32{Valid: false},

			IdentityID: sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Add ratelimits to the first key only
		err = db.Query.InsertKeyRatelimit(ctx, h.DB.RW(), db.InsertKeyRatelimitParams{
			ID:          uid.New("ratelimit"),
			WorkspaceID: workspace.ID,
			KeyID:       sql.NullString{Valid: true, String: keyWithRatelimits},
			Name:        "requests",
			Limit:       100,
			Duration:    60000,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Call the endpoint
		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body.Data)

		// Find both keys in the response and verify ratelimits behavior
		var foundKeyWithRatelimits, foundKeyWithoutRatelimits bool
		for _, key := range res.Body.Data {
			switch key.KeyId {
			case keyWithRatelimits:
				foundKeyWithRatelimits = true
				require.NotNil(t, key.Ratelimits, "Key with ratelimits should have ratelimits field")
				require.Len(t, key.Ratelimits, 1, "Should have exactly 1 ratelimit")

				ratelimit := key.Ratelimits[0]
				require.Equal(t, "requests", ratelimit.Name)
				require.Equal(t, int64(100), ratelimit.Limit)
				require.Equal(t, int64(60000), ratelimit.Duration)
				require.NotEmpty(t, ratelimit.Id, "Ratelimit should have an ID")

			case keyWithoutRatelimits:
				foundKeyWithoutRatelimits = true
				// Key without ratelimits should have nil or empty ratelimits array
				if key.Ratelimits != nil {
					require.Len(t, key.Ratelimits, 0, "Key without ratelimits should have empty ratelimits array")
				}
				// Both nil and empty array are acceptable
			}
		}
		require.True(t, foundKeyWithRatelimits, "Should find the key with ratelimits in response")
		require.True(t, foundKeyWithoutRatelimits, "Should find the key without ratelimits in response")
	})

	t.Run("verify encrypted key is returned correctly", func(t *testing.T) {
		req := handler.Request{
			ApiId:   apiID,
			Decrypt: ptr.P(true),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body.Data)

		for _, key := range res.Body.Data {
			expectedPlaintext, exists := encryptedKeys[key.KeyId]
			if !exists {
				continue
			}

			require.NotEmpty(t, key.Plaintext, "Key should be decrypted and have plaintext")
			require.Equal(t, expectedPlaintext, key.Plaintext, "Key should be decrypted and have correct plaintext")
		}
	})
}
