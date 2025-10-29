package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_credits"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/zen"
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
		Credits: &seed.CreditRequest{
			Remaining: initialCredits,
		},
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
		nullValue := nullable.Nullable[int64]{}
		nullValue.SetNull()

		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Set,
			Value:     nullValue,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		t.Logf("Request Operation: %v, Value.IsNull(): %v", req.Operation, req.Value.IsNull())
		t.Logf("Remaining: %+v, IsNull: %v, IsSpecified: %v", res.Body.Data.Remaining, res.Body.Data.Remaining.IsNull(), res.Body.Data.Remaining.IsSpecified())
		t.Logf("Refill: %+v", res.Body.Data.Refill)
		require.True(t, res.Body.Data.Remaining.IsNull())

		_, err := db.Query.FindCreditsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyID, Valid: true})
		require.True(t, db.IsNotFound(err))
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

		credits, err := db.Query.FindCreditsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, credits.Remaining, setTo)
	})

	increaseBy := int64(rand.IntN(50) + 1)
	t.Run(fmt.Sprintf("increase credits by %d", increaseBy), func(t *testing.T) {
		// Get current credits before decrement
		credits, err := db.Query.FindCreditsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyID, Valid: true})
		require.NoError(t, err)
		currentCredits := int64(credits.Remaining)

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

		credits, err = db.Query.FindCreditsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, credits.Remaining, currentCredits+increaseBy)
	})

	decreaseBy := int64(rand.IntN(50) + 1)
	t.Run(fmt.Sprintf("decrease credits by %d", decreaseBy), func(t *testing.T) {
		// Get current credits before decrement
		credits, err := db.Query.FindCreditsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyID, Valid: true})
		require.NoError(t, err)
		currentCredits := int64(credits.Remaining)

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

		credits, err = db.Query.FindCreditsByKeyID(ctx, h.DB.RO(), sql.NullString{String: keyID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, credits.Remaining, shouldBeRemaining)
	})

	t.Run("counter cache invalidation after credit update", func(t *testing.T) {
		// Create a new key with initial credits for this test
		initialCredits := int32(100)
		cacheTestKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
			Credits: &seed.CreditRequest{
				Remaining: initialCredits,
			},
		})

		authBefore, _, err := h.Keys.Get(ctx, &zen.Session{}, cacheTestKey.Key)
		require.NoError(t, err)

		err = authBefore.Verify(ctx, keys.WithCredits(1))
		require.NoError(t, err)

		require.NotNil(t, authBefore.KeyCredits)
		require.EqualValues(t, initialCredits-1, authBefore.KeyCredits.Remaining)

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
		authAfter, _, err := h.Keys.Get(ctx, &zen.Session{}, cacheTestKey.Key)
		require.NoError(t, err)

		err = authAfter.Verify(ctx, keys.WithCredits(1))
		require.NoError(t, err)

		require.NotNil(t, authAfter.KeyCredits)
		require.EqualValues(t, newCredits-1, authAfter.KeyCredits.Remaining)
	})

	// Test legacy credit system operations
	t.Run("set legacy credits to specific value", func(t *testing.T) {
		// Create key with legacy credits
		initialCredits := int32(100)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeyAuthID:               api.KeyAuthID.String,
			Name:                    ptr.P("legacy-set-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		// Verify it's using legacy system
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, initialCredits, key.RemainingRequests.Int32)

		// Update to a new value
		newValue := int64(250)
		req := handler.Request{
			KeyId:     keyResponse.KeyID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(newValue),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, newValue, remaining)

		// Verify in database (still using legacy field)
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, newValue, key.RemainingRequests.Int32)
	})

	t.Run("set legacy credits to unlimited (null)", func(t *testing.T) {
		initialCredits := int32(75)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeyAuthID:               api.KeyAuthID.String,
			Name:                    ptr.P("legacy-null-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		// Set to null (unlimited)
		nullValue := nullable.Nullable[int64]{}
		nullValue.SetNull()

		req := handler.Request{
			KeyId:     keyResponse.KeyID,
			Operation: openapi.Set,
			Value:     nullValue,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.True(t, res.Body.Data.Remaining.IsNull())

		// Verify legacy field is cleared
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.False(t, key.RemainingRequests.Valid)
	})

	t.Run("increment legacy credits", func(t *testing.T) {
		initialCredits := int32(rand.IntN(100) + 50)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeyAuthID:               api.KeyAuthID.String,
			Name:                    ptr.P("legacy-increment-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		increaseBy := int64(rand.IntN(50) + 1)
		req := handler.Request{
			KeyId:     keyResponse.KeyID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(increaseBy),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(initialCredits)+increaseBy, remaining)

		// Verify in database
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, int64(initialCredits)+increaseBy, key.RemainingRequests.Int32)
	})

	t.Run("decrement legacy credits", func(t *testing.T) {
		initialCredits := int32(rand.IntN(100) + 100)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeyAuthID:               api.KeyAuthID.String,
			Name:                    ptr.P("legacy-decrement-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		decreaseBy := int64(rand.IntN(50) + 1)
		req := handler.Request{
			KeyId:     keyResponse.KeyID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(decreaseBy),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(initialCredits)-decreaseBy, remaining)

		// Verify in database
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, int64(initialCredits)-decreaseBy, key.RemainingRequests.Int32)
	})

	t.Run("decrement legacy credits below zero sets to zero", func(t *testing.T) {
		initialCredits := int32(10)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeyAuthID:               api.KeyAuthID.String,
			Name:                    ptr.P("legacy-decrement-negative-test"),
			LegacyRemainingRequests: &initialCredits,
		})

		decreaseBy := int64(50) // More than current
		req := handler.Request{
			KeyId:     keyResponse.KeyID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(decreaseBy),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(0), remaining)

		// Verify in database
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.RemainingRequests.Valid)
		require.EqualValues(t, 0, key.RemainingRequests.Int32)
	})

	t.Run("legacy credits with refill configuration", func(t *testing.T) {
		initialCredits := int32(100)
		refillAmount := int32(200)
		refillDay := int16(15)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:             workspace.ID,
			KeyAuthID:               api.KeyAuthID.String,
			Name:                    ptr.P("legacy-refill-test"),
			LegacyRemainingRequests: &initialCredits,
			LegacyRefillAmount:      &refillAmount,
			LegacyRefillDay:         &refillDay,
		})

		// Verify refill configuration exists
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.RefillAmount.Valid)
		require.EqualValues(t, refillAmount, key.RefillAmount.Int32)
		require.True(t, key.RefillDay.Valid)
		require.EqualValues(t, refillDay, key.RefillDay.Int16)

		// Update credits - refill should be preserved
		newValue := int64(150)
		req := handler.Request{
			KeyId:     keyResponse.KeyID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(newValue),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify response includes refill
		require.NotNil(t, res.Body.Data.Refill)
		require.Equal(t, int64(refillAmount), res.Body.Data.Refill.Amount)
		require.Equal(t, openapi.Monthly, res.Body.Data.Refill.Interval)
		require.NotNil(t, res.Body.Data.Refill.RefillDay)
		require.Equal(t, int(refillDay), *res.Body.Data.Refill.RefillDay)

		// Verify refill still exists in database
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.RefillAmount.Valid)
		require.EqualValues(t, refillAmount, key.RefillAmount.Int32)
		require.True(t, key.RefillDay.Valid)
		require.EqualValues(t, refillDay, key.RefillDay.Int16)
	})
}
