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
		LiveKeyCache: h.Caches.LiveKeyByID,
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
		Meta:       nullable.NewNullableWithValue(map[string]interface{}{"test": "test"}),
		Expires:    nullable.NewNullableWithValue(time.Now().Add(time.Hour).UnixMilli()),
		Enabled:    ptr.P(true),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "Expected 200, got: %d", res.Status)
	require.NotNil(t, res.Body)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	t.Run("upsert ratelimit", func(t *testing.T) {
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
		LiveKeyCache: h.Caches.LiveKeyByID,
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
		Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
			Remaining: nullable.NewNullableWithValue(int64(100)),
			Refill: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsRefill{
				Interval: openapi.UpdateKeyCreditsRefillIntervalDaily,
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
	require.Equal(t, int32(100), key.RemainingRequests.Int32)
	require.Equal(t, int32(50), key.RefillAmount.Int32)

	// Verify identity was created with correct external ID
	identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
		IdentityID:  key.IdentityID.String,
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
		LiveKeyCache: h.Caches.LiveKeyByID,
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
		Remaining:   &initialCredits,
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

	require.True(t, authBefore.Key.RemainingRequests.Valid)
	require.Equal(t, initialCredits-1, authBefore.Key.RemainingRequests.Int32)

	// Update the key's credits
	newCredits := int64(50)

	req := handler.Request{
		KeyId: key.KeyID,
		Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
			Refill:    nullable.NewNullNullable[openapi.UpdateKeyCreditsRefill](),
			Remaining: nullable.NewNullableWithValue[int64](newCredits),
		}),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)

	// Verify the key again to check if cache was properly invalidated
	authAfter, _, err := h.Keys.Get(ctx, &zen.Session{}, key.Key)
	require.NoError(t, err)

	err = authAfter.Verify(ctx, keys.WithCredits(1))
	require.NoError(t, err)

	require.True(t, authAfter.Key.RemainingRequests.Valid)
	require.Equal(t, int32(newCredits)-1, authAfter.Key.RemainingRequests.Int32)

}
