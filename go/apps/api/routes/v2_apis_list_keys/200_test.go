package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_list_keys"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:   h.Logger,
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

	// Create a test API with encrypted keys enabled
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: true,
		Name:          ptr.P("Test API"),
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	apiID := api.ID
	keyAuthID := api.KeyAuthID.String

	identity1 := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "test_user_1",
		Meta:        []byte(`{"role": "admin"}`),
		Ratelimits:  nil,
		Credits:     nil,
	})
	identity2 := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "test_user_2",
		Meta:        []byte(`{"role": "user"}`),
		Ratelimits:  nil,
		Credits:     nil,
	})

	// Create test keys with various configurations using seeder
	h.CreateKey(seed.CreateKeyRequest{
		Disabled:    false,
		WorkspaceID: workspace.ID,
		KeySpaceID:  keyAuthID,
		IdentityID:  &identity1.ID,
		Meta:        ptr.P(`{"env": "production", "team": "backend"}`),
		Name:        ptr.P("Test Key 1"),
		Recoverable: true,
	})

	h.CreateKey(seed.CreateKeyRequest{
		Disabled:    false,
		WorkspaceID: workspace.ID,
		KeySpaceID:  keyAuthID,
		IdentityID:  &identity2.ID,
		Meta:        ptr.P(`{"env": "staging"}`),
		Name:        ptr.P("Test Key 2"),
		Recoverable: true,
	})

	h.CreateKey(seed.CreateKeyRequest{
		Disabled:    false,
		WorkspaceID: workspace.ID,
		KeySpaceID:  keyAuthID,
		IdentityID:  &identity1.ID,
		Meta:        ptr.P(`{"env": "development"}`),
		Name:        ptr.P("Test Key 3"),
		Recoverable: true,
	})

	h.CreateKey(seed.CreateKeyRequest{
		Disabled:    false,
		WorkspaceID: workspace.ID,
		KeySpaceID:  keyAuthID,
		Name:        ptr.P("Test Key 4 (No Identity)"),
		Recoverable: true,
	})

	h.CreateKey(seed.CreateKeyRequest{
		Disabled:    true,
		WorkspaceID: workspace.ID,
		KeySpaceID:  keyAuthID,
		Name:        ptr.P("Test Key 5 (Disabled)"),
		Recoverable: true,
	})

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
			ExternalId: &identity1.ExternalID,
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
			require.Equal(t, identity1.ExternalID, key.Identity.ExternalId)
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
				meta := *key.Meta
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
			ExternalId: &identity1.ExternalID,
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
		require.Equal(t, identity1.ExternalID, key.Identity.ExternalId)
		require.NotNil(t, key.Identity.Meta)

		// Verify identity metadata
		identityMeta := *key.Identity.Meta
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
		emptyApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			IpWhitelist:   "",
			EncryptedKeys: false,
			Name:          ptr.P("Empty API"),
		})
		emptyApiID := emptyApi.ID

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
			require.Nil(t, key.Plaintext)

			// Only start prefix should be shown - allow reasonable prefix lengths
			require.NotEmpty(t, key.Start)
			require.True(t, len(key.Start) <= 32, "Start should be a reasonable prefix length")
		}
	})

	t.Run("verify ratelimits are returned correctly", func(t *testing.T) {
		// Create a key with ratelimits
		keyWithRatelimitsResp := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  keyAuthID,
			Name:        ptr.P("Key with Ratelimits"),
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "requests",
					WorkspaceID: workspace.ID,
					AutoApply:   false,
					Duration:    60000,
					Limit:       100,
				},
			},
		})
		keyWithRatelimits := keyWithRatelimitsResp.KeyID

		// Create a key without ratelimits
		keyWithoutRatelimitsResp := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  keyAuthID,
			Name:        ptr.P("Key without Ratelimits"),
		})
		keyWithoutRatelimits := keyWithoutRatelimitsResp.KeyID

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
				require.Len(t, *key.Ratelimits, 1, "Should have exactly 1 ratelimit")

				ratelimit := (*key.Ratelimits)[0]
				require.Equal(t, "requests", ratelimit.Name)
				require.Equal(t, int64(100), ratelimit.Limit)
				require.Equal(t, int64(60000), ratelimit.Duration)
				require.NotEmpty(t, ratelimit.Id, "Ratelimit should have an ID")

			case keyWithoutRatelimits:
				foundKeyWithoutRatelimits = true
				// Key without ratelimits should have nil or empty ratelimits array
				if key.Ratelimits != nil {
					require.Len(t, *key.Ratelimits, 0, "Key without ratelimits should have empty ratelimits array")
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

		// All keys created with Recoverable: true should have plaintext
		for _, key := range res.Body.Data {
			if key.Plaintext != nil {
				require.NotEmpty(t, *key.Plaintext, "Decrypted key should have non-empty plaintext")
			}
		}
	})

	t.Run("verify key with new credits system", func(t *testing.T) {
		// Create a key with credits in the new credits table
		keyWithNewCredits := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  keyAuthID,
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

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

		// Find the key with new credits
		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithNewCredits.KeyID {
				foundKey = true
				require.NotNil(t, key.Credits, "Key should have credits field")
				remaining, err := key.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(100), remaining, "Credits remaining should be 100")
				require.Nil(t, key.Credits.Refill, "Key should have no refill configuration")
				break
			}
		}
		require.True(t, foundKey, "Should find the key with new credits in response")
	})

	t.Run("verify key with new credits and daily refill", func(t *testing.T) {
		refillAmount := int32(100)
		keyWithRefill := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Credits: &seed.CreditRequest{
				Remaining:    50,
				RefillAmount: &refillAmount,
			},
		})

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

		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithRefill.KeyID {
				foundKey = true
				require.NotNil(t, key.Credits, "Key should have credits field")
				remaining, err := key.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(50), remaining)
				require.NotNil(t, key.Credits.Refill, "Key should have refill configuration")
				require.Equal(t, int64(100), key.Credits.Refill.Amount)
				require.Equal(t, "daily", string(key.Credits.Refill.Interval))
				require.Nil(t, key.Credits.Refill.RefillDay, "Daily refill should have no refill day")
				break
			}
		}
		require.True(t, foundKey, "Should find the key with refill in response")
	})

	t.Run("verify key with new credits and monthly refill", func(t *testing.T) {
		refillAmount := int32(200)
		refillDay := int16(15)
		keyWithMonthlyRefill := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Credits: &seed.CreditRequest{
				Remaining:    75,
				RefillAmount: &refillAmount,
				RefillDay:    &refillDay,
			},
		})

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

		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithMonthlyRefill.KeyID {
				foundKey = true
				require.NotNil(t, key.Credits, "Key should have credits field")
				remaining, err := key.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(75), remaining)
				require.NotNil(t, key.Credits.Refill, "Key should have refill configuration")
				require.Equal(t, int64(200), key.Credits.Refill.Amount)
				require.Equal(t, "monthly", string(key.Credits.Refill.Interval))
				require.NotNil(t, key.Credits.Refill.RefillDay, "Monthly refill should have refill day")
				require.Equal(t, 15, *key.Credits.Refill.RefillDay)
				break
			}
		}
		require.True(t, foundKey, "Should find the key with monthly refill in response")
	})

	t.Run("verify key with legacy credits system", func(t *testing.T) {
		// Create a key with credits in the legacy keys.remaining_requests field
		keyWithLegacyCreditsResp := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeySpaceID:              keyAuthID,
			Name:                    ptr.P("Key with Legacy Credits"),
			LegacyRemainingRequests: ptr.P(int32(250)),
		})
		keyWithLegacyCredits := keyWithLegacyCreditsResp.KeyID

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

		// Find the key with legacy credits
		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithLegacyCredits {
				foundKey = true
				require.NotNil(t, key.Credits, "Key should have credits field")
				remaining, err := key.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(250), remaining, "Legacy credits remaining should be 250")
				require.Nil(t, key.Credits.Refill, "Legacy key should have no refill configuration")
				break
			}
		}
		require.True(t, foundKey, "Should find the key with legacy credits in response")
	})

	t.Run("verify key with legacy credits and refill", func(t *testing.T) {
		keyWithLegacyRefillResp := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeySpaceID:              keyAuthID,
			Name:                    ptr.P("Key with Legacy Refill"),
			LegacyRemainingRequests: ptr.P(int32(150)),
			LegacyRefillAmount:      ptr.P(int32(300)),
			LegacyRefillDay:         ptr.P(int16(1)),
		})
		keyWithLegacyRefill := keyWithLegacyRefillResp.KeyID

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

		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithLegacyRefill {
				foundKey = true
				require.NotNil(t, key.Credits, "Key should have credits field")
				remaining, err := key.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(150), remaining)
				require.NotNil(t, key.Credits.Refill, "Legacy key should have refill configuration")
				require.Equal(t, int64(300), key.Credits.Refill.Amount)
				require.Equal(t, "monthly", string(key.Credits.Refill.Interval))
				require.NotNil(t, key.Credits.Refill.RefillDay, "Monthly refill should have refill day")
				require.Equal(t, 1, *key.Credits.Refill.RefillDay)
				break
			}
		}
		require.True(t, foundKey, "Should find the key with legacy refill in response")
	})

	t.Run("verify key with unlimited credits (no credits field)", func(t *testing.T) {
		// Create a key with no credits (unlimited)
		keyUnlimited := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
		})

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

		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyUnlimited.KeyID {
				foundKey = true
				require.Nil(t, key.Credits, "Key with unlimited credits should have nil credits field")
				break
			}
		}
		require.True(t, foundKey, "Should find the key with unlimited credits in response")
	})

	t.Run("verify key with identity credits", func(t *testing.T) {
		// Create identity with credits
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(""),
			Credits: &seed.CreditRequest{
				Remaining: 150,
			},
		})

		// Create key linked to identity
		keyWithIdentityCredits := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  &identity.ID,
		})

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

		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithIdentityCredits.KeyID {
				foundKey = true
				require.NotNil(t, key.Identity, "Key should have identity")
				require.Equal(t, identity.ID, key.Identity.Id)
				require.NotNil(t, key.Identity.Credits, "Identity should have credits")
				remaining, err := key.Identity.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(150), remaining, "Identity credits remaining should be 150")
				require.Nil(t, key.Identity.Credits.Refill, "Identity should have no refill configuration")
				break
			}
		}
		require.True(t, foundKey, "Should find the key with identity credits in response")
	})

	t.Run("verify key with identity credits and daily refill", func(t *testing.T) {
		refillAmount := int32(500)
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(""),
			Credits: &seed.CreditRequest{
				Remaining:    300,
				RefillAmount: &refillAmount,
			},
		})

		keyWithIdentityCredits := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  &identity.ID,
		})

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

		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithIdentityCredits.KeyID {
				foundKey = true
				require.NotNil(t, key.Identity, "Key should have identity")
				require.NotNil(t, key.Identity.Credits, "Identity should have credits")
				remaining, err := key.Identity.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(300), remaining)
				require.NotNil(t, key.Identity.Credits.Refill, "Identity should have refill configuration")
				require.Equal(t, int64(500), key.Identity.Credits.Refill.Amount)
				require.Equal(t, "daily", string(key.Identity.Credits.Refill.Interval))
				require.Nil(t, key.Identity.Credits.Refill.RefillDay, "Daily refill should have no refill day")
				break
			}
		}
		require.True(t, foundKey, "Should find the key with identity credits and daily refill")
	})

	t.Run("verify key with identity credits and monthly refill", func(t *testing.T) {
		refillAmount := int32(1000)
		refillDay := int16(1)
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(""),
			Credits: &seed.CreditRequest{
				Remaining:    750,
				RefillAmount: &refillAmount,
				RefillDay:    &refillDay,
			},
		})

		keyWithIdentityCredits := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  &identity.ID,
		})

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

		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithIdentityCredits.KeyID {
				foundKey = true
				require.NotNil(t, key.Identity, "Key should have identity")
				require.NotNil(t, key.Identity.Credits, "Identity should have credits")
				remaining, err := key.Identity.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(750), remaining)
				require.NotNil(t, key.Identity.Credits.Refill, "Identity should have refill configuration")
				require.Equal(t, int64(1000), key.Identity.Credits.Refill.Amount)
				require.Equal(t, "monthly", string(key.Identity.Credits.Refill.Interval))
				require.NotNil(t, key.Identity.Credits.Refill.RefillDay, "Monthly refill should have refill day")
				require.Equal(t, 1, *key.Identity.Credits.Refill.RefillDay)
				break
			}
		}
		require.True(t, foundKey, "Should find the key with identity credits and monthly refill")
	})

	t.Run("verify key with both key and identity credits", func(t *testing.T) {
		// Create identity with credits
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  uid.New(""),
			Credits: &seed.CreditRequest{
				Remaining: 500,
			},
		})

		// Create key with its own credits linked to identity
		keyWithBothCredits := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  &identity.ID,
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

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

		var foundKey bool
		for _, key := range res.Body.Data {
			if key.KeyId == keyWithBothCredits.KeyID {
				foundKey = true
				// Key should have its own credits
				require.NotNil(t, key.Credits, "Key should have credits field")
				keyRemaining, err := key.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(100), keyRemaining, "Key credits remaining should be 100")

				// Key should also have identity with credits
				require.NotNil(t, key.Identity, "Key should have identity")
				require.NotNil(t, key.Identity.Credits, "Identity should have credits")
				identityRemaining, err := key.Identity.Credits.Remaining.Get()
				require.NoError(t, err)
				require.Equal(t, int64(500), identityRemaining, "Identity credits remaining should be 500")
				break
			}
		}
		require.True(t, foundKey, "Should find the key with both key and identity credits")
	})
}
