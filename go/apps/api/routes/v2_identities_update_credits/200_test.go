package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_credits"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
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
		UsageLimiter: h.UsageLimiter,
		KeyCache:     h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	initialCredits := int32(rand.IntN(50))

	externalId := uid.New("")
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalId,
		Meta:        nil,
		Ratelimits:  nil,
		Credits: &seed.CreditRequest{
			Remaining: initialCredits,
		},
	})

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.update_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// This also tests that we have the correct data for the identity.
	t.Run("update identity set to unlimited credits", func(t *testing.T) {
		nullValue := nullable.Nullable[int64]{}
		nullValue.SetNull()

		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Set,
			Value:     nullValue,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Remaining.IsNull())

		_, err := db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.True(t, db.IsNotFound(err))
	})

	setTo := int64(rand.IntN(50) + 1)
	t.Run(fmt.Sprintf("set to fixed value of %d", setTo), func(t *testing.T) {
		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(setTo),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, remaining, setTo)

		credits, err := db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, credits.Remaining, setTo)
	})

	increaseBy := int64(rand.IntN(50) + 1)
	t.Run(fmt.Sprintf("increase credits by %d", increaseBy), func(t *testing.T) {
		// Get current credits before decrement
		credits, err := db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		currentCredits := int64(credits.Remaining)

		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(increaseBy),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, remaining, currentCredits+increaseBy)

		credits, err = db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, credits.Remaining, currentCredits+increaseBy)
	})

	decreaseBy := int64(rand.IntN(50) + 1)
	t.Run(fmt.Sprintf("decrease credits by %d", decreaseBy), func(t *testing.T) {
		// Get current credits before decrement
		credits, err := db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		currentCredits := int64(credits.Remaining)

		// If we are decreasing credits into the negative, it will be automatically set to 0
		shouldBeRemaining := int64(0)
		if currentCredits-decreaseBy > 0 {
			shouldBeRemaining = currentCredits - decreaseBy
		}

		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(decreaseBy),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, remaining, shouldBeRemaining)

		credits, err = db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, credits.Remaining, shouldBeRemaining)
	})

	t.Run("counter cache invalidation after credit update", func(t *testing.T) {
		// Create a new key with initial credits for this test
		initialCredits := int32(100)

		// Create a test API and key with random initial credits using testutil helper
		apiName := "Test API"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
			Name:        &apiName,
		})

		externalId := uid.New("")
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  externalId,
			Meta:        nil,
			Ratelimits:  nil,
			Credits: &seed.CreditRequest{
				Remaining: initialCredits,
			},
		})

		cacheTestKey := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  &identity.ID,
		})

		authBefore, _, err := h.Keys.Get(ctx, &zen.Session{}, cacheTestKey.Key)
		require.NoError(t, err)

		err = authBefore.Verify(ctx, keys.WithCredits(1))
		require.NoError(t, err)

		require.NotNil(t, authBefore.IdentityCredits)
		require.EqualValues(t, initialCredits-1, authBefore.IdentityCredits.Remaining)

		// Update the key's credits
		newCredits := int64(50)
		updateReq := handler.Request{
			Identity:  identity.ID,
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

		require.NotNil(t, authAfter.IdentityCredits)
		require.EqualValues(t, newCredits-1, authAfter.IdentityCredits.Remaining)
	})

	t.Run("set to zero credits", func(t *testing.T) {
		externalId := uid.New("")
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  externalId,
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(int64(0)),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(0), remaining)

		// Verify in database - should have credits record with 0 (not deleted)
		credits, err := db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, 0, credits.Remaining)
	})

	t.Run("set from unlimited to limited", func(t *testing.T) {
		// Create identity without credits (unlimited)
		externalId := uid.New("")
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  externalId,
			// No credits - unlimited
		})

		// Verify unlimited (no credits record)
		_, err := db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.True(t, db.IsNotFound(err))

		// Set to specific value
		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(int64(500)),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(500), remaining)

		// Verify in database
		credits, err := db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, 500, credits.Remaining)
	})

	t.Run("increment preserves refill configuration", func(t *testing.T) {
		// Create identity with credits and refill config using the helper
		externalId := uid.New("")
		refillDay := int16(15)
		refillAmount := int32(50)
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  externalId,
			Credits: &seed.CreditRequest{
				Remaining:    100,
				RefillAmount: &refillAmount,
				RefillDay:    &refillDay,
			},
		})

		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(int64(10)),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify credits updated
		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(110), remaining)

		// Verify refill config is preserved
		require.NotNil(t, res.Body.Data.Refill)
		refill := res.Body.Data.Refill
		require.Equal(t, int64(50), refill.Amount)
		require.Equal(t, openapi.Monthly, refill.Interval)
		require.NotNil(t, refill.RefillDay)
		require.Equal(t, int(15), *refill.RefillDay)
	})

	t.Run("decrement preserves refill configuration", func(t *testing.T) {
		// Create identity with credits and daily refill
		creditID := uid.New("credit")
		identityID := uid.New(uid.IdentityPrefix)
		externalId := uid.New("")

		err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalId,
			WorkspaceID: workspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        []byte("{}"),
		})
		require.NoError(t, err)

		err = db.Query.InsertCredit(ctx, h.DB.RW(), db.InsertCreditParams{
			ID:           creditID,
			WorkspaceID:  workspace.ID,
			IdentityID:   sql.NullString{String: identityID, Valid: true},
			Remaining:    200,
			RefillAmount: sql.NullInt32{Int32: 25, Valid: true},
			RefillDay:    sql.NullInt16{Valid: false}, // Daily refill
			CreatedAt:    time.Now().UnixMilli(),
			UpdatedAt:    sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
		require.NoError(t, err)

		req := handler.Request{
			Identity:  externalId,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(int64(50)),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify credits updated
		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(150), remaining)

		// Verify refill config is preserved
		require.NotNil(t, res.Body.Data.Refill)
		refill := res.Body.Data.Refill
		require.Equal(t, int64(25), refill.Amount)
		require.Equal(t, openapi.Daily, refill.Interval)
		require.Nil(t, refill.RefillDay)
	})

	t.Run("multiple keys using same identity credits", func(t *testing.T) {
		// Create identity with credits
		externalId := uid.New("")
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  externalId,
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

		// Create API
		apiName := "Multi Key Test API"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
			Name:        &apiName,
		})

		// Create multiple keys linked to same identity
		key1 := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  &identity.ID,
		})

		key2 := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  &identity.ID,
		})

		key3 := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  &identity.ID,
		})

		// Verify initial state for all keys
		auth1, _, err := h.Keys.Get(ctx, &zen.Session{}, key1.Key)
		require.NoError(t, err)
		require.NotNil(t, auth1.IdentityCredits)
		require.EqualValues(t, 100, auth1.IdentityCredits.Remaining)

		auth2, _, err := h.Keys.Get(ctx, &zen.Session{}, key2.Key)
		require.NoError(t, err)
		require.NotNil(t, auth2.IdentityCredits)
		require.EqualValues(t, 100, auth2.IdentityCredits.Remaining)

		// Update credits
		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Set,
			Value:     nullable.NewNullableWithValue(int64(250)),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify all keys see the update (cache invalidated)
		auth1After, _, err := h.Keys.Get(ctx, &zen.Session{}, key1.Key)
		require.NoError(t, err)
		require.NotNil(t, auth1After.IdentityCredits)
		require.EqualValues(t, 250, auth1After.IdentityCredits.Remaining)

		auth2After, _, err := h.Keys.Get(ctx, &zen.Session{}, key2.Key)
		require.NoError(t, err)
		require.NotNil(t, auth2After.IdentityCredits)
		require.EqualValues(t, 250, auth2After.IdentityCredits.Remaining)

		auth3After, _, err := h.Keys.Get(ctx, &zen.Session{}, key3.Key)
		require.NoError(t, err)
		require.NotNil(t, auth3After.IdentityCredits)
		require.EqualValues(t, 250, auth3After.IdentityCredits.Remaining)
	})

	t.Run("decrement to exactly zero", func(t *testing.T) {
		externalId := uid.New("")
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  externalId,
			Credits: &seed.CreditRequest{
				Remaining: 50,
			},
		})

		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(int64(50)),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(0), remaining)

		// Verify in database
		credits, err := db.Query.FindCreditsByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		require.NoError(t, err)
		require.EqualValues(t, 0, credits.Remaining)
	})

	t.Run("increment very large value", func(t *testing.T) {
		externalId := uid.New("")
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  externalId,
			Credits: &seed.CreditRequest{
				Remaining: 100,
			},
		})

		// Increment by large value
		req := handler.Request{
			Identity:  identity.ID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(int64(1000000)),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		remaining, err := res.Body.Data.Remaining.Get()
		require.NoError(t, err)
		require.Equal(t, int64(1000100), remaining)
	})
}
