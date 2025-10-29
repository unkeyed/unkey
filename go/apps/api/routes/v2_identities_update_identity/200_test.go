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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:       h.Logger,
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.update_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
	}

	// Setup test data
	ctx := context.Background()

	workspaceID := h.Resources().UserWorkspace.ID
	identityID := uid.New(uid.IdentityPrefix)
	otherIdentityID := uid.New(uid.IdentityPrefix)
	externalID := "test_user_123"
	otherExternalID := "test_user_456"

	// Create initial metadata
	metaMap := map[string]interface{}{
		"name":    "Test User",
		"email":   "test@example.com",
		"plan":    "free",
		"credits": 50,
	}
	metaBytes, err := json.Marshal(metaMap)
	require.NoError(t, err)

	// Insert test identities
	err = db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(t, err)

	err = db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          otherIdentityID,
		ExternalID:  otherExternalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	})
	require.NoError(t, err)

	// Insert test ratelimits for the first identity
	ratelimitID1 := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
		ID:          ratelimitID1,
		WorkspaceID: workspaceID,
		IdentityID:  sql.NullString{String: identityID, Valid: true},
		Name:        "api_calls",
		Limit:       100,
		Duration:    60000, // 1 minute
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	ratelimitID2 := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
		ID:          ratelimitID2,
		WorkspaceID: workspaceID,
		IdentityID:  sql.NullString{String: identityID, Valid: true},
		Name:        "special_feature",
		Limit:       10,
		Duration:    3600000, // 1 hour
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	t.Run("update metadata", func(t *testing.T) {
		newMeta := map[string]interface{}{
			"joined": "2023-01-01",
			"active": true,
		}

		req := handler.Request{
			Identity: otherExternalID,
			Meta:     &newMeta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, otherExternalID, res.Body.Data.ExternalId)

		// Verify metadata
		require.NotNil(t, res.Body.Data.Meta)
		meta := *res.Body.Data.Meta
		assert.Equal(t, "2023-01-01", meta["joined"])
		assert.Equal(t, true, meta["active"])

		// Verify no ratelimits
		require.Nil(t, res.Body.Data.Ratelimits)
	})

	t.Run("update ratelimits - add new, update existing, delete one", func(t *testing.T) {
		// This will:
		// 1. Update 'api_calls' limit from 100 to 200
		// 2. Add a new 'new_feature' limit
		// 3. Delete 'special_feature' limit (by not including it)
		ratelimits := []openapi.RatelimitRequest{
			{
				Name:      "api_calls",
				Limit:     200,
				Duration:  60000,
				AutoApply: true,
			},
			{
				Name:      "new_feature",
				Limit:     5,
				Duration:  86400000, // 1 day
				AutoApply: false,
			},
		}

		req := handler.Request{
			Identity:   externalID,
			Ratelimits: &ratelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify exactly 2 ratelimits (should have removed 'special_feature')
		require.NotNil(t, res.Body.Data.Ratelimits)
		require.Len(t, *res.Body.Data.Ratelimits, 2)

		// Check ratelimit values
		var apiCallsLimit, newFeatureLimit *openapi.RatelimitResponse
		for i := range *res.Body.Data.Ratelimits {
			switch (*res.Body.Data.Ratelimits)[i].Name {
			case "api_calls":
				apiCallsLimit = &(*res.Body.Data.Ratelimits)[i]
			case "new_feature":
				newFeatureLimit = &(*res.Body.Data.Ratelimits)[i]
			}
		}

		require.NotNil(t, apiCallsLimit, "api_calls ratelimit not found")
		require.NotNil(t, newFeatureLimit, "new_feature ratelimit not found")

		// Verify updated limit
		assert.Equal(t, int64(200), apiCallsLimit.Limit)
		assert.Equal(t, int64(60000), apiCallsLimit.Duration)

		// Verify new limit
		assert.Equal(t, int64(5), newFeatureLimit.Limit)
		assert.Equal(t, int64(86400000), newFeatureLimit.Duration)

		// Verify 'special_feature' was removed
		for _, rl := range *res.Body.Data.Ratelimits {
			assert.NotEqual(t, "special_feature", rl.Name, "special_feature should have been removed")
		}
	})

	t.Run("remove all ratelimits", func(t *testing.T) {
		// Empty array should remove all ratelimits
		emptyRatelimits := []openapi.RatelimitRequest{}

		req := handler.Request{
			Identity:   externalID,
			Ratelimits: &emptyRatelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify no ratelimits
		require.Nil(t, res.Body.Data.Ratelimits)
	})

	t.Run("clear metadata", func(t *testing.T) {
		// Empty map should clear metadata
		emptyMeta := map[string]interface{}{}

		req := handler.Request{
			Identity: externalID,
			Meta:     &emptyMeta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify response
		require.Equal(t, externalID, res.Body.Data.ExternalId)

		// Verify empty metadata
		require.NotNil(t, res.Body.Data.Meta)
		assert.Empty(t, *res.Body.Data.Meta)
	})

	t.Run("update both metadata and ratelimits", func(t *testing.T) {
		newMeta := map[string]interface{}{
			"plan":    "enterprise",
			"credits": 1000,
		}

		ratelimits := []openapi.RatelimitRequest{
			{
				Name:      "enterprise_feature",
				Limit:     50,
				Duration:  3600000,
				AutoApply: true,
			},
		}

		req := handler.Request{
			Identity:   externalID,
			Meta:       &newMeta,
			Ratelimits: &ratelimits,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify metadata
		require.NotNil(t, res.Body.Data.Meta)
		meta := *res.Body.Data.Meta
		assert.Equal(t, "enterprise", meta["plan"])
		assert.Equal(t, float64(1000), meta["credits"])

		// Verify ratelimits
		require.NotNil(t, res.Body.Data.Ratelimits)
		require.Len(t, *res.Body.Data.Ratelimits, 1)
		rlimits := *res.Body.Data.Ratelimits
		assert.Equal(t, "enterprise_feature", rlimits[0].Name)
		assert.Equal(t, int64(50), rlimits[0].Limit)
		assert.Equal(t, int64(3600000), rlimits[0].Duration)
	})
}

func TestCreditsUpdateBehavior(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:       h.Logger,
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.update_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
	}

	ctx := context.Background()
	workspaceID := h.Resources().UserWorkspace.ID

	// Helper function to create identity with credits
	createIdentityWithCredits := func(externalID string, remaining int32, refillAmount sql.NullInt32, refillDay sql.NullInt16) string {
		identityID := uid.New(uid.IdentityPrefix)
		err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalID,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        []byte("{}"),
		})
		require.NoError(t, err)

		creditID := uid.New("credit")
		err = db.Query.InsertCredit(ctx, h.DB.RW(), db.InsertCreditParams{
			ID:           creditID,
			WorkspaceID:  workspaceID,
			IdentityID:   sql.NullString{String: identityID, Valid: true},
			Remaining:    remaining,
			RefillAmount: refillAmount,
			RefillDay:    refillDay,
			CreatedAt:    time.Now().UnixMilli(),
			UpdatedAt:    sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
		require.NoError(t, err)

		return externalID
	}

	// Helper function to create identity without credits
	createIdentityWithoutCredits := func(externalID string) string {
		identityID := uid.New(uid.IdentityPrefix)
		err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalID,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        []byte("{}"),
		})
		require.NoError(t, err)
		return externalID
	}

	t.Run("add credits to identity without credits", func(t *testing.T) {
		identityID := createIdentityWithoutCredits("test_add_credits")

		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue[int64](100),
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify credits were added
		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(100), res.Body.Data.Credits.Remaining.MustGet())
		assert.Nil(t, res.Body.Data.Credits.Refill)
	})

	t.Run("update existing credits remaining only", func(t *testing.T) {
		identityID := createIdentityWithCredits("test_update_remaining", 50,
			sql.NullInt32{Int32: 100, Valid: true}, // refill amount
			sql.NullInt16{Int16: 15, Valid: true})  // refill day

		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue[int64](200),
				// Not specifying refill - should preserve existing refill config
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify credits updated
		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(200), res.Body.Data.Credits.Remaining.MustGet())

		// Verify refill preserved
		require.NotNil(t, res.Body.Data.Credits.Refill)
		refill := *res.Body.Data.Credits.Refill
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(100), refill.Amount)
		assert.Equal(t, openapi.Monthly, refill.Interval)
		assert.NotNil(t, refill.RefillDay)
		assert.Equal(t, 15, *refill.RefillDay)
	})

	t.Run("add refill to credits without refill", func(t *testing.T) {
		identityID := createIdentityWithCredits("test_add_refill", 50,
			sql.NullInt32{Valid: false}, // no refill amount
			sql.NullInt16{Valid: false}) // no refill day

		refillDay := 10
		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Refill: nullable.NewNullableWithValue(openapi.UpdateCreditsRefill{
					Amount:    150,
					Interval:  openapi.Monthly,
					RefillDay: &refillDay,
				}),
				// Not specifying remaining - should preserve existing
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify remaining preserved
		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(50), res.Body.Data.Credits.Remaining.MustGet())

		// Verify refill added
		require.NotNil(t, res.Body.Data.Credits.Refill)
		refill := *res.Body.Data.Credits.Refill
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(150), refill.Amount)
		assert.Equal(t, openapi.Monthly, refill.Interval)
		assert.NotNil(t, refill.RefillDay)
		assert.Equal(t, 10, *refill.RefillDay)
	})

	t.Run("change refill configuration", func(t *testing.T) {
		identityID := createIdentityWithCredits("test_change_refill", 75,
			sql.NullInt32{Int32: 100, Valid: true}, // refill amount
			sql.NullInt16{Int16: 15, Valid: true})  // refill day

		refillDay := 1
		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue[int64](300),
				Refill: nullable.NewNullableWithValue(openapi.UpdateCreditsRefill{
					Amount:    500,
					Interval:  openapi.Monthly,
					RefillDay: &refillDay,
				}),
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify everything changed
		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(300), res.Body.Data.Credits.Remaining.MustGet())

		require.NotNil(t, res.Body.Data.Credits.Refill)
		refill := res.Body.Data.Credits.Refill
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(500), refill.Amount)
		assert.Equal(t, openapi.Monthly, refill.Interval)
		assert.NotNil(t, refill.RefillDay)
		assert.Equal(t, 1, *refill.RefillDay)
	})

	t.Run("remove refill by setting to null", func(t *testing.T) {
		identityID := createIdentityWithCredits("test_remove_refill", 100,
			sql.NullInt32{Int32: 200, Valid: true}, // refill amount
			sql.NullInt16{Int16: 20, Valid: true})  // refill day

		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Refill: nullable.NewNullNullable[openapi.UpdateCreditsRefill](), // explicitly null
				// Not specifying remaining - should preserve
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify remaining preserved
		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(100), res.Body.Data.Credits.Remaining.MustGet())

		// Verify refill removed
		assert.Nil(t, res.Body.Data.Credits.Refill)
	})

	t.Run("daily refill configuration", func(t *testing.T) {
		identityID := createIdentityWithCredits("test_daily_refill", 50,
			sql.NullInt32{Valid: false}, // no refill
			sql.NullInt16{Valid: false})

		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Refill: nullable.NewNullableWithValue(openapi.UpdateCreditsRefill{
					Amount:   25,
					Interval: openapi.Daily,
					// RefillDay should be nil for daily
				}),
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify refill
		require.NotNil(t, res.Body.Data.Credits)
		require.NotNil(t, res.Body.Data.Credits.Refill)
		refill := *res.Body.Data.Credits.Refill
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(25), refill.Amount)
		assert.Equal(t, openapi.Daily, refill.Interval)
		assert.Nil(t, refill.RefillDay) // Should be nil for daily
	})

	t.Run("remove all credits by setting to null", func(t *testing.T) {
		identityID := createIdentityWithCredits("test_remove_credits", 100,
			sql.NullInt32{Int32: 200, Valid: true}, // refill amount
			sql.NullInt16{Int16: 15, Valid: true})  // refill day

		req := handler.Request{
			Identity: identityID,
			Credits:  nullable.NewNullNullable[openapi.UpdateCredits](), // explicitly null
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify credits removed
		assert.Nil(t, res.Body.Data.Credits)
	})

	t.Run("set unlimited credits", func(t *testing.T) {
		identityID := createIdentityWithoutCredits("test_unlimited_credits")

		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullNullable[int64](), // null means unlimited
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify unlimited credits
		require.Nil(t, res.Body.Data.Credits)
	})

	t.Run("update credits with metadata in same request", func(t *testing.T) {
		identityID := createIdentityWithCredits("test_credits_and_meta", 50,
			sql.NullInt32{Valid: false},
			sql.NullInt16{Valid: false})

		meta := map[string]interface{}{
			"plan": "pro",
			"tier": 2,
		}

		req := handler.Request{
			Identity: identityID,
			Meta:     &meta,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue[int64](1000),
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify credits
		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(1000), res.Body.Data.Credits.Remaining.MustGet())

		// Verify metadata
		require.NotNil(t, res.Body.Data.Meta)
		assert.Equal(t, "pro", (*res.Body.Data.Meta)["plan"])
		assert.Equal(t, float64(2), (*res.Body.Data.Meta)["tier"])
	})

	t.Run("not specifying credits preserves existing", func(t *testing.T) {
		identityID := createIdentityWithCredits("test_preserve_credits", 777,
			sql.NullInt32{Int32: 888, Valid: true},
			sql.NullInt16{Int16: 25, Valid: true})

		meta := map[string]interface{}{
			"updated": true,
		}

		req := handler.Request{
			Identity: identityID,
			Meta:     &meta,
			// Credits not specified - should be preserved
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		// Verify credits preserved
		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(777), res.Body.Data.Credits.Remaining.MustGet())

		require.NotNil(t, res.Body.Data.Credits.Refill)
		refill := *res.Body.Data.Credits.Refill
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(888), refill.Amount)
		assert.Equal(t, openapi.Monthly, refill.Interval)
		assert.NotNil(t, refill.RefillDay)
		assert.Equal(t, 25, *refill.RefillDay)
	})

	t.Run("complex scenario - change everything", func(t *testing.T) {
		// Start with monthly refill
		identityID := createIdentityWithCredits("test_complex_change", 100,
			sql.NullInt32{Int32: 50, Valid: true},
			sql.NullInt16{Int16: 10, Valid: true})

		// Change to daily refill with different amount and remaining
		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue[int64](500),
				Refill: nullable.NewNullableWithValue(openapi.UpdateCreditsRefill{
					Amount:   20,
					Interval: openapi.Daily,
				}),
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(500), res.Body.Data.Credits.Remaining.MustGet())

		require.NotNil(t, res.Body.Data.Credits.Refill)
		refill := *res.Body.Data.Credits.Refill
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(20), refill.Amount)
		assert.Equal(t, openapi.Daily, refill.Interval)
		assert.Nil(t, refill.RefillDay)
	})

	t.Run("edge case - zero credits", func(t *testing.T) {
		identityID := createIdentityWithoutCredits("test_zero_credits")

		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue[int64](0),
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotNil(t, res.Body.Data.Credits)
		require.True(t, res.Body.Data.Credits.Remaining.IsSpecified())
		assert.Equal(t, int64(0), res.Body.Data.Credits.Remaining.MustGet())
	})

	t.Run("edge case - refill with invalid day for monthly", func(t *testing.T) {
		identityID := createIdentityWithoutCredits("test_invalid_refill_day")

		refillDay := 32 // Invalid day
		req := handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue[int64](100),
				Refill: nullable.NewNullableWithValue(openapi.UpdateCreditsRefill{
					Amount:    50,
					Interval:  openapi.Monthly,
					RefillDay: &refillDay,
				}),
			}),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		// Invalid refill day should return 400
		require.Equal(t, 400, res.Status)
	})

	t.Run("three state handling - not set vs null vs value", func(t *testing.T) {
		// Create identity with full credits configuration
		identityID := createIdentityWithCredits("test_three_state", 100,
			sql.NullInt32{Int32: 50, Valid: true},
			sql.NullInt16{Int16: 15, Valid: true})

		// Test 1: Update only metadata (credits not set) - should preserve all credits
		meta := map[string]interface{}{"test": "value"}
		req := handler.Request{
			Identity: identityID,
			Meta:     &meta,
			// Credits field not set at all - should preserve
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Credits)
		assert.Equal(t, int64(100), res.Body.Data.Credits.Remaining.MustGet())

		// Test 2: Set refill to null but don't touch remaining
		req = handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Refill: nullable.NewNullNullable[openapi.UpdateCreditsRefill](), // explicitly null
				// Remaining not set - should preserve
			}),
		}
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Credits)
		assert.Equal(t, int64(100), res.Body.Data.Credits.Remaining.MustGet()) // preserved
		assert.Nil(t, res.Body.Data.Credits.Refill)                            // removed

		// Test 3: Update remaining but don't touch refill (which is now null)
		req = handler.Request{
			Identity: identityID,
			Credits: nullable.NewNullableWithValue(openapi.UpdateCredits{
				Remaining: nullable.NewNullableWithValue[int64](200),
				// Refill not set - should preserve (as null)
			}),
		}
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.Credits)
		assert.Equal(t, int64(200), res.Body.Data.Credits.Remaining.MustGet())
		assert.Nil(t, res.Body.Data.Credits.Refill) // still null
	})
}
