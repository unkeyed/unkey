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
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_key"
	"golang.org/x/sync/errgroup"
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

	authBefore, _, err := h.Keys.Get(ctx, &zen.Session{}, hash.Sha256(key.Key))
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
	authAfter, _, err := h.Keys.Get(ctx, &zen.Session{}, hash.Sha256(key.Key))
	require.NoError(t, err)

	err = authAfter.Verify(ctx, keys.WithCredits(1))
	require.NoError(t, err)

	require.True(t, authAfter.Key.RemainingRequests.Valid)
	require.Equal(t, int32(newCredits)-1, authAfter.Key.RemainingRequests.Int32)
}

// TestUpdateKeyConcurrentWithSameExternalId tests that concurrent updates
// to different keys with the same new externalId don't deadlock.
// This was previously possible due to gap locks when inserting identities.
// The fix uses INSERT ... ON DUPLICATE KEY UPDATE (upsert) to avoid deadlocks.
func TestUpdateKeyConcurrentWithSameExternalId(t *testing.T) {
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

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	// Create multiple keys without identities
	numKeys := 10
	keyIDs := make([]string, numKeys)
	for i := range numKeys {
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        ptr.P(fmt.Sprintf("key-%d", i)),
		})
		keyIDs[i] = keyResponse.KeyID
	}

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	externalID := "shared_identity_deadlock_test"

	g := errgroup.Group{}
	for _, keyID := range keyIDs {
		g.Go(func() error {
			req := handler.Request{
				KeyId:      keyID,
				ExternalId: nullable.NewNullableWithValue(externalID),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if res.Status != 200 {
				return fmt.Errorf("key %s: unexpected status %d", keyID, res.Status)
			}
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err, "All concurrent updates should succeed without deadlock")

	// Verify all keys reference the same identity
	var sharedIdentityID string
	for i, keyID := range keyIDs {
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.True(t, key.IdentityID.Valid, "Key should have identity after update")

		if i == 0 {
			sharedIdentityID = key.IdentityID.String
		} else {
			require.Equal(t, sharedIdentityID, key.IdentityID.String,
				"All keys updated with same externalId should share the same identity")
		}
	}

	// Verify only one identity was created
	identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		ExternalID:  externalID,
		Deleted:     false,
	})
	require.NoError(t, err)
	require.Equal(t, sharedIdentityID, identity.ID)
}

// TestUpdateKeyConcurrentRatelimits tests that concurrent updates to the
// same key's ratelimits don't deadlock. The handler uses SELECT ... FOR UPDATE
// on the key row to serialize concurrent modifications.
func TestUpdateKeyConcurrentRatelimits(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	// Create a single key that all concurrent requests will update
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        ptr.P("concurrent-ratelimit-test-key"),
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	numConcurrent := 10

	g := errgroup.Group{}
	for i := range numConcurrent {
		g.Go(func() error {
			// All concurrent requests modify the SAME ratelimits on the SAME key
			ratelimits := []openapi.RatelimitRequest{
				{Name: "shared_limit_a", Limit: int64(100 + i), Duration: 60000, AutoApply: true},
				{Name: "shared_limit_b", Limit: int64(200 + i), Duration: 60000, AutoApply: true},
				{Name: "shared_limit_c", Limit: int64(300 + i), Duration: 60000, AutoApply: true},
			}
			req := handler.Request{
				KeyId:      keyResponse.KeyID,
				Ratelimits: ptr.P(ratelimits),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if res.Status != 200 {
				return fmt.Errorf("request %d: unexpected status %d", i, res.Status)
			}
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err, "All concurrent updates should succeed without deadlock")
}
