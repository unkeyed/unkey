package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_key"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestUpdateKeySuccess(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	// Create API using helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	// Create key using helper
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        ptr.P("test"),
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		KeyId:      keyResponse.KeyID,
		Name:       nullable.NewNullableWithValue("test2"),
		ExternalId: nullable.NewNullableWithValue("test2"),
		Meta:       nullable.NewNullableWithValue(map[string]any{"test": "test"}),
		Expires:    nullable.NewNullableWithValue(time.Now().Add(time.Hour).UnixMilli()),
		Enabled:    ptr.P(true),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "Expected 200, got: %d", res.Status)
	require.NotNil(t, res.Body)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	t.Run("upsert ratelimit", func(t *testing.T) {
		t.Parallel()
		ratelimit := openapi.RatelimitRequest{
			AutoApply: false,
			Duration:  (time.Minute * 5).Milliseconds(),
			Limit:     100,
			Name:      "test",
		}

		req := handler.Request{
			KeyId:      keyResponse.KeyID,
			Ratelimits: ptr.P([]openapi.RatelimitRequest{ratelimit}),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "Expected 200, got: %d", res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		ratelimits, err := db.Query.ListRatelimitsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyResponse.KeyID, Valid: true})
		require.NoError(t, err)
		require.Len(t, ratelimits, 1)
		require.Equal(t, ratelimit.Name, ratelimits[0].Name)
		require.Equal(t, ratelimit.AutoApply, ratelimits[0].AutoApply)
		require.EqualValues(t, ratelimit.Duration, ratelimits[0].Duration)
		require.EqualValues(t, ratelimit.Limit, ratelimits[0].Limit)

		ratelimit = openapi.RatelimitRequest{
			AutoApply: true,
			Duration:  (time.Minute * 15).Milliseconds(),
			Limit:     100,
			Name:      "test",
		}

		req = handler.Request{
			KeyId:      keyResponse.KeyID,
			Ratelimits: ptr.P([]openapi.RatelimitRequest{ratelimit}),
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "Expected 200, got: %d", res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		ratelimits, err = db.Query.ListRatelimitsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyResponse.KeyID, Valid: true})
		require.NoError(t, err)
		require.Len(t, ratelimits, 1)
		require.Equal(t, ratelimit.Name, ratelimits[0].Name)
		require.Equal(t, ratelimit.AutoApply, ratelimits[0].AutoApply)
		require.EqualValues(t, ratelimit.Duration, ratelimits[0].Duration)
		require.EqualValues(t, ratelimit.Limit, ratelimits[0].Limit)
	})
}

func TestUpdateKeyUpdateAllFields(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	// Create API using helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	// Create key using helper
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: api.WorkspaceID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        ptr.P("test"),
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	h.CreateRole(seed.CreateRoleRequest{WorkspaceID: api.WorkspaceID, Name: "admin"})
	h.CreateRole(seed.CreateRoleRequest{WorkspaceID: api.WorkspaceID, Name: "user"})

	req := handler.Request{
		KeyId:      keyResponse.KeyID,
		Name:       nullable.NewNullableWithValue("newName"),
		ExternalId: nullable.NewNullableWithValue("newExternalId"),
		Meta:       nullable.NewNullableWithValue(map[string]any{"new": "meta"}),
		Expires:    nullable.NewNullNullable[int64](),
		Enabled:    ptr.P(true),
		Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
			Remaining: nullable.NewNullableWithValue(int64(100)),
			Refill: nullable.NewNullableWithValue(openapi.UpdateCreditsRefill{
				Interval: openapi.Daily,
				Amount:   50,
			}),
		}),
		Permissions: &[]string{"read", "write"},
		Roles:       &[]string{"admin", "user"},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "Expected 200, got: %d", res.Status)
	require.NotNil(t, res.Body)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	// Verify key was updated
	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.Equal(t, "newName", key.Name.String)
	require.True(t, key.IdentityID.Valid, "Should have identity ID set")

	// Verify credits were created
	credits, err := db.Query.FindCreditsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyResponse.KeyID, Valid: true})
	require.NoError(t, err)
	require.Equal(t, int32(100), credits.Remaining)
	require.True(t, credits.RefillAmount.Valid)
	require.Equal(t, int32(50), credits.RefillAmount.Int32)
	require.False(t, credits.RefillDay.Valid, "daily interval should not have refill day")

	// Verify identity was created with correct external ID
	identity, err := db.Query.FindIdentity(ctx, h.DB.RO(), db.FindIdentityParams{
		Identity:    key.IdentityID.String,
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "newExternalId", identity.ExternalID)
}

func TestKeyUpdateCreditsInvalidatesCache(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:       h.Logger,
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	// Create a test API and key with random initial credits using testutil helper
	apiName := "Test API"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
		Name:        &apiName,
	})

	keyName := "test-key"
	initialCredits := int32(100)
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
		Credits: &seed.CreditRequest{
			Remaining: initialCredits,
		},
	})

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	authBefore, _, err := h.Keys.Get(ctx, &zen.Session{}, key.Key)
	require.NoError(t, err)

	err = authBefore.Verify(ctx, keys.WithCredits(1))
	require.NoError(t, err)

	require.NotNil(t, authBefore.KeyCredits)
	require.Equal(t, int32(initialCredits)-1, authBefore.KeyCredits.Remaining)

	// Update the key's credits
	newCredits := int64(50)

	req := handler.Request{
		KeyId: key.KeyID,
		Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
			Refill:    nullable.NewNullNullable[openapi.UpdateCreditsRefill](),
			Remaining: nullable.NewNullableWithValue(newCredits),
		}),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)

	// Verify the key again to check if cache was properly invalidated
	authAfter, _, err := h.Keys.Get(ctx, &zen.Session{}, key.Key)
	require.NoError(t, err)

	err = authAfter.Verify(ctx, keys.WithCredits(1))
	require.NoError(t, err)

	require.NotNil(t, authAfter.KeyCredits)
	require.EqualValues(t, newCredits-1, authAfter.KeyCredits.Remaining)

	t.Run("update legacy credits to specific value", func(t *testing.T) {
		// Create key with legacy credits
		initialCredits := int32(100)
		legacyKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeySpaceID:              api.KeyAuthID.String,
			Name:                    ptr.P("legacy-update-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		// Verify it's using legacy system
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), legacyKey.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, initialCredits, key.RemainingRequests.Int32)

		// Update to new value via updateKey
		newValue := int64(250)
		req := handler.Request{
			KeyId: legacyKey.KeyID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue(newValue),
				Refill:    nullable.NewNullNullable[openapi.UpdateCreditsRefill](),
			}),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify in database (still using legacy field)
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), legacyKey.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, newValue, key.RemainingRequests.Int32)
	})

	t.Run("update legacy credits to unlimited (null)", func(t *testing.T) {
		// Create key with legacy credits
		initialCredits := int32(150)
		legacyKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeySpaceID:              api.KeyAuthID.String,
			Name:                    ptr.P("legacy-unlimited-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		// Verify it's using legacy system
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), legacyKey.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)

		// Set to unlimited (null)
		req := handler.Request{
			KeyId: legacyKey.KeyID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullNullable[int64](),
				Refill:    nullable.NewNullNullable[openapi.UpdateCreditsRefill](),
			}),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify field is now null
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), legacyKey.KeyID)
		require.NoError(t, err)
		require.False(t, key.RemainingRequests.Valid, "RemainingRequests should be null for unlimited")
	})

	t.Run("update legacy credits with refill configuration", func(t *testing.T) {
		// Create key with legacy credits
		initialCredits := int32(100)
		legacyKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeySpaceID:              api.KeyAuthID.String,
			Name:                    ptr.P("legacy-refill-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		// Update with daily refill
		req := handler.Request{
			KeyId: legacyKey.KeyID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue(int64(200)),
				Refill: nullable.NewNullableWithValue(openapi.UpdateCreditsRefill{
					Interval: openapi.Daily,
					Amount:   50,
				}),
			}),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify in database (legacy fields)
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), legacyKey.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, 200, key.RemainingRequests.Int32)
		require.True(t, key.RefillAmount.Valid)
		require.EqualValues(t, 50, key.RefillAmount.Int32)
		require.False(t, key.RefillDay.Valid, "daily refill should not have refillDay")
	})

	t.Run("update legacy credits with monthly refill", func(t *testing.T) {
		// Create key with legacy credits
		initialCredits := int32(100)
		legacyKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeySpaceID:              api.KeyAuthID.String,
			Name:                    ptr.P("legacy-monthly-refill-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		// Update with monthly refill
		refillDay := 15
		req := handler.Request{
			KeyId: legacyKey.KeyID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue(int64(300)),
				Refill: nullable.NewNullableWithValue(openapi.UpdateCreditsRefill{
					Interval:  openapi.Monthly,
					Amount:    75,
					RefillDay: &refillDay,
				}),
			}),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify in database (legacy fields)
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), legacyKey.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, 300, key.RemainingRequests.Int32)
		require.True(t, key.RefillAmount.Valid)
		require.EqualValues(t, 75, key.RefillAmount.Int32)
		require.True(t, key.RefillDay.Valid)
		require.EqualValues(t, 15, key.RefillDay.Int16)
	})

	t.Run("clear legacy refill configuration", func(t *testing.T) {
		// Create key with legacy credits and refill
		initialCredits := int32(100)
		refillAmount := int32(50)
		legacyKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeySpaceID:              api.KeyAuthID.String,
			Name:                    ptr.P("legacy-clear-refill-test"),
			LegacyRemainingRequests: &initialCredits,
			LegacyRefillAmount:      &refillAmount,
		})

		// Verify initial state
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), legacyKey.KeyID)
		require.NoError(t, err)
		require.True(t, key.RefillAmount.Valid)

		// Clear refill by setting to null
		req := handler.Request{
			KeyId: legacyKey.KeyID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue(int64(150)),
				Refill:    nullable.NewNullNullable[openapi.UpdateCreditsRefill](),
			}),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify refill fields are cleared
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), legacyKey.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, 150, key.RemainingRequests.Int32)
		require.False(t, key.RefillAmount.Valid, "refill amount should be cleared")
		require.False(t, key.RefillDay.Valid, "refill day should be cleared")
	})

}
