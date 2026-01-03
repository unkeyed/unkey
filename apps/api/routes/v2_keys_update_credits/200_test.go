package handler_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_keys_update_credits"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestKeyUpdateCreditsSuccess(t *testing.T) {
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
	initialCredits := int32(rand.IntN(50))
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
		Remaining:   &initialCredits,
	})
	keyID := keyResponse.KeyID

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// This also tests that we have the correct data for the key.
	t.Run("update key set to unlimited credits", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Set,
			Value:     nullable.Nullable[int64]{},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.Remaining.IsNull(), true)

		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotNil(t, key)
		require.Equal(t, key.RemainingRequests.Valid, false)
		require.Equal(t, key.RefillAmount.Valid, false)
		require.Equal(t, key.RefillDay.Valid, false)
	})

	setTo := int64(rand.IntN(50) + 1)
	t.Run(fmt.Sprintf("set to fixed value of %d", setTo), func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(setTo),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, remaining, setTo)

		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotNil(t, key)
		require.Equal(t, key.RemainingRequests.Valid, true)
		require.EqualValues(t, key.RemainingRequests.Int32, setTo)
	})

	increaseBy := int64(rand.IntN(50) + 1)
	t.Run(fmt.Sprintf("increase credits by %d", increaseBy), func(t *testing.T) {
		// Get current credits before decrement
		currentKey, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.True(t, currentKey.RemainingRequests.Valid)
		currentCredits := int64(currentKey.RemainingRequests.Int32)

		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(increaseBy),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, remaining, currentCredits+increaseBy)

		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotNil(t, key)
		require.Equal(t, key.RemainingRequests.Valid, true)
		require.EqualValues(t, key.RemainingRequests.Int32, currentCredits+increaseBy)
	})

	decreaseBy := int64(rand.IntN(50) + 1)
	t.Run(fmt.Sprintf("decrease credits by %d", decreaseBy), func(t *testing.T) {
		// Get current credits before decrement
		currentKey, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.True(t, currentKey.RemainingRequests.Valid)
		currentCredits := int64(currentKey.RemainingRequests.Int32)

		// If we are decreasing credits into the negative, it will be automatically set to 0
		shouldBeRemaining := int64(0)
		if currentCredits-decreaseBy > 0 {
			shouldBeRemaining = currentCredits - decreaseBy
		}

		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(decreaseBy),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, remaining, shouldBeRemaining)

		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.NotNil(t, key)
		require.Equal(t, key.RemainingRequests.Valid, true)
		require.EqualValues(t, key.RemainingRequests.Int32, shouldBeRemaining)
	})

	t.Run("counter cache invalidation after credit update", func(t *testing.T) {
		// Create a new key with initial credits for this test
		initialCredits := int32(100)
		cacheTestKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
			Remaining:   &initialCredits,
		})

		authBefore, _, err := h.Keys.Get(ctx, &zen.Session{}, hash.Sha256(cacheTestKey.Key))
		require.NoError(t, err)

		err = authBefore.Verify(ctx, keys.WithCredits(1))
		require.NoError(t, err)

		require.True(t, authBefore.Key.RemainingRequests.Valid)
		require.Equal(t, initialCredits-1, authBefore.Key.RemainingRequests.Int32)

		// Update the key's credits
		newCredits := int64(50)
		updateReq := handler.Request{
			KeyId:     cacheTestKey.KeyID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(newCredits),
		}

		updateRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, updateReq)
		require.Equal(t, 200, updateRes.Status)
		require.NotNil(t, updateRes.Body)

		// Verify the response shows updated credits
		updatedRemaining, err := updateRes.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, newCredits, updatedRemaining)

		// Verify the key again to check if cache was properly invalidated
		authAfter, _, err := h.Keys.Get(ctx, &zen.Session{}, hash.Sha256(cacheTestKey.Key))
		require.NoError(t, err)

		err = authAfter.Verify(ctx, keys.WithCredits(1))
		require.NoError(t, err)

		require.True(t, authAfter.Key.RemainingRequests.Valid)
		require.Equal(t, int32(newCredits)-1, authAfter.Key.RemainingRequests.Int32)

	})
}
