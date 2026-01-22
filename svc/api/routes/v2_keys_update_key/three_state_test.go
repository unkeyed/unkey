package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_key"
)

// TestThreeStateUpdateLogic tests that all fields properly support the three-state logic:
// 1. Set to specific value
// 2. Set to NULL (clear value)
// 3. Undefined (keep existing value)
func TestThreeStateUpdateLogic(t *testing.T) {
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

	// Create root key with update permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("Name field three-state logic", func(t *testing.T) {
		// Create key with initial name
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        ptr.P("initial-name"),
		})

		// Test 1: Set to specific value
		req := handler.Request{
			KeyId: keyResponse.KeyID,
			Name:  nullable.NewNullableWithValue("updated-name"),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the name was updated
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.Name.Valid)
		require.Equal(t, "updated-name", key.Name.String)

		// Test 2: Set to NULL (clear value)
		req = handler.Request{
			KeyId:   keyResponse.KeyID,
			Name:    nullable.NewNullNullable[string](),
			Enabled: ptr.P(true), // Add a valid field to avoid empty request
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the name was cleared
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.False(t, key.Name.Valid)

		// Test 3: Undefined (keep existing value)
		// First set a value
		req = handler.Request{
			KeyId: keyResponse.KeyID,
			Name:  nullable.NewNullableWithValue("preserved-name"),
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Now update something else without specifying name
		req = handler.Request{
			KeyId:   keyResponse.KeyID,
			Enabled: ptr.P(false),
			// Name is not specified, should keep existing value
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the name was preserved
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.Name.Valid)
		require.Equal(t, "preserved-name", key.Name.String)
		require.False(t, key.Enabled) // But enabled should be updated
	})

	t.Run("Meta field three-state logic", func(t *testing.T) {
		t.Parallel()
		// Create key with initial meta
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Meta:        ptr.P(`{"initial": "value"}`),
		})

		// Test 1: Set to specific value
		req := handler.Request{
			KeyId: keyResponse.KeyID,
			Meta:  nullable.NewNullableWithValue(map[string]interface{}{"updated": "value"}),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the meta was updated
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.Meta.Valid)
		require.Contains(t, key.Meta.String, "updated")

		// Test 2: Set to NULL (clear value)
		req = handler.Request{
			KeyId: keyResponse.KeyID,
			Meta:  nullable.NewNullNullable[map[string]interface{}](),
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the meta was cleared
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.False(t, key.Meta.Valid)

		// Test 3: Undefined (keep existing value)
		// First set a value
		req = handler.Request{
			KeyId: keyResponse.KeyID,
			Meta:  nullable.NewNullableWithValue(map[string]interface{}{"preserved": "value"}),
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Now update something else without specifying meta
		req = handler.Request{
			KeyId:   keyResponse.KeyID,
			Enabled: ptr.P(true),
			// Meta is not specified, should keep existing value
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the meta was preserved
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.Meta.Valid)
		require.Contains(t, key.Meta.String, "preserved")
	})

	t.Run("Expires field three-state logic", func(t *testing.T) {
		t.Parallel()
		// Create key with initial expiration
		futureTime := time.Now().Add(24 * time.Hour)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Expires:     ptr.P(futureTime),
		})

		// Test 1: Set to specific value
		newExpireTime := time.Now().Add(48 * time.Hour)
		req := handler.Request{
			KeyId:   keyResponse.KeyID,
			Expires: nullable.NewNullableWithValue(newExpireTime.UnixMilli()),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the expiration was updated
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.Expires.Valid)
		require.Equal(t, newExpireTime.UnixMilli(), key.Expires.Time.UnixMilli())

		// Test 2: Set to NULL (clear value - make permanent)
		req = handler.Request{
			KeyId:   keyResponse.KeyID,
			Expires: nullable.NewNullNullable[int64](),
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the expiration was cleared
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.False(t, key.Expires.Valid)

		// Test 3: Undefined (keep existing value)
		// First set a value
		preservedTime := time.Now().Add(72 * time.Hour)
		req = handler.Request{
			KeyId:   keyResponse.KeyID,
			Expires: nullable.NewNullableWithValue(preservedTime.UnixMilli()),
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Now update something else without specifying expires
		req = handler.Request{
			KeyId:   keyResponse.KeyID,
			Enabled: ptr.P(false),
			// Expires is not specified, should keep existing value
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the expiration was preserved
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.Expires.Valid)
		require.Equal(t, preservedTime.UnixMilli(), key.Expires.Time.UnixMilli())
	})

	t.Run("Identity field three-state logic", func(t *testing.T) {
		t.Parallel()
		// Create key without identity initially
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        ptr.P("identity-test-key"),
		})

		// Test 1: Set to specific value (create new identity)
		req := handler.Request{
			KeyId:      keyResponse.KeyID,
			ExternalId: nullable.NewNullableWithValue("updated-user"),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the identity was updated
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.IdentityID.Valid)

		// Check that the identity exists
		identity, err := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			IdentityID:  key.IdentityID.String,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		require.NoError(t, err)
		require.Equal(t, "updated-user", identity.ExternalID)

		// Test 2: Set to NULL (clear identity association)
		req = handler.Request{
			KeyId:      keyResponse.KeyID,
			ExternalId: nullable.NewNullNullable[string](),
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the identity was cleared
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.False(t, key.IdentityID.Valid)

		// Test 3: Undefined (keep existing value)
		// First set a value
		req = handler.Request{
			KeyId:      keyResponse.KeyID,
			ExternalId: nullable.NewNullableWithValue("preserved-user"),
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Now update something else without specifying external ID
		req = handler.Request{
			KeyId:   keyResponse.KeyID,
			Enabled: ptr.P(true),
			// ExternalId is not specified, should keep existing value
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify the identity was preserved
		key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)
		require.True(t, key.IdentityID.Valid)

		// Check that the identity exists with correct external ID
		identity, err = db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			IdentityID:  key.IdentityID.String,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		require.NoError(t, err)
		require.Equal(t, "preserved-user", identity.ExternalID)
	})

	t.Run("Credits field three-state logic", func(t *testing.T) {
		t.Parallel()
		// Test Case 1: Create key with credits and refill -> Set credits object to null => everything is gone
		t.Run("Set credits object to null clears everything", func(t *testing.T) {
			t.Parallel()
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID:  h.Resources().UserWorkspace.ID,
				KeySpaceID:   api.KeyAuthID.String,
				Name:         ptr.P("credits-null-test"),
				Remaining:    ptr.P(int32(100)),
				RefillAmount: ptr.P(int32(50)),
				RefillDay:    ptr.P(int16(15)),
			})

			// Verify initial state
			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 100, key.RemainingRequests.Int32)
			require.True(t, key.RefillAmount.Valid)
			require.EqualValues(t, 50, key.RefillAmount.Int32)
			require.True(t, key.RefillDay.Valid)
			require.EqualValues(t, 15, key.RefillDay.Int16)

			// Set credits to null
			req := handler.Request{
				KeyId:   keyResponse.KeyID,
				Credits: nullable.NewNullNullable[openapi.UpdateKeyCreditsData](),
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			// Verify everything is cleared
			key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.False(t, key.RemainingRequests.Valid, "remaining should be cleared")
			require.False(t, key.RefillAmount.Valid, "refill amount should be cleared")
			require.False(t, key.RefillDay.Valid, "refill day should be cleared")
		})

		// Test Case 2: Set credits.remaining to null => everything is gone (remaining + refill)
		t.Run("Set credits.remaining to null clears everything", func(t *testing.T) {
			t.Parallel()
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID:  h.Resources().UserWorkspace.ID,
				KeySpaceID:   api.KeyAuthID.String,
				Name:         ptr.P("remaining-null-test"),
				Remaining:    ptr.P(int32(200)),
				RefillAmount: ptr.P(int32(75)),
				RefillDay:    ptr.P(int16(10)),
			})

			// Verify initial state
			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 200, key.RemainingRequests.Int32)
			require.True(t, key.RefillAmount.Valid)
			require.EqualValues(t, 75, key.RefillAmount.Int32)
			require.True(t, key.RefillDay.Valid)
			require.EqualValues(t, 10, key.RefillDay.Int16)

			// Set credits.remaining to null
			req := handler.Request{
				KeyId: keyResponse.KeyID,
				Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
					Remaining: nullable.NewNullNullable[int64](),
				}),
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			// Verify everything is cleared
			key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.False(t, key.RemainingRequests.Valid, "remaining should be cleared")
			require.False(t, key.RefillAmount.Valid, "refill amount should be cleared")
			require.False(t, key.RefillDay.Valid, "refill day should be cleared")
		})

		// Test Case 3: Set credits.refill to null => ONLY clears refill (amount and day), keeps remaining
		t.Run("Set credits.refill to null only clears refill, keeps remaining", func(t *testing.T) {
			t.Parallel()
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID:  h.Resources().UserWorkspace.ID,
				KeySpaceID:   api.KeyAuthID.String,
				Name:         ptr.P("refill-null-test"),
				Remaining:    ptr.P(int32(300)),
				RefillAmount: ptr.P(int32(100)),
				RefillDay:    ptr.P(int16(25)),
			})

			// Verify initial state
			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 300, key.RemainingRequests.Int32)
			require.True(t, key.RefillAmount.Valid)
			require.EqualValues(t, 100, key.RefillAmount.Int32)
			require.True(t, key.RefillDay.Valid)
			require.EqualValues(t, 25, key.RefillDay.Int16)

			// Set credits.refill to null
			req := handler.Request{
				KeyId: keyResponse.KeyID,
				Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
					Refill: nullable.NewNullNullable[openapi.UpdateKeyCreditsRefill](),
				}),
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			// Verify only refill is cleared, remaining stays
			key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid, "remaining should be preserved")
			require.EqualValues(t, 300, key.RemainingRequests.Int32, "remaining value should be unchanged")
			require.False(t, key.RefillAmount.Valid, "refill amount should be cleared")
			require.False(t, key.RefillDay.Valid, "refill day should be cleared")
		})

		// Test Case 4: Update refill without touching remaining
		t.Run("Update refill without touching remaining", func(t *testing.T) {
			t.Parallel()
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID:  h.Resources().UserWorkspace.ID,
				KeySpaceID:   api.KeyAuthID.String,
				Name:         ptr.P("update-refill-only"),
				Remaining:    ptr.P(int32(500)),
				RefillAmount: ptr.P(int32(50)),
				RefillDay:    ptr.P(int16(5)),
			})

			// Update only refill
			req := handler.Request{
				KeyId: keyResponse.KeyID,
				Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
					Refill: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsRefill{
						Amount:    150,
						Interval:  openapi.UpdateKeyCreditsRefillIntervalMonthly,
						RefillDay: ptr.P(20),
					}),
				}),
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			// Verify refill updated, remaining unchanged
			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 500, key.RemainingRequests.Int32, "remaining should be unchanged")
			require.True(t, key.RefillAmount.Valid)
			require.EqualValues(t, 150, key.RefillAmount.Int32, "refill amount should be updated")
			require.True(t, key.RefillDay.Valid)
			require.EqualValues(t, 20, key.RefillDay.Int16, "refill day should be updated")
		})

		// Test Case 5: Update remaining without touching refill
		t.Run("Update remaining without touching refill", func(t *testing.T) {
			t.Parallel()
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID:  h.Resources().UserWorkspace.ID,
				KeySpaceID:   api.KeyAuthID.String,
				Name:         ptr.P("update-remaining-only"),
				Remaining:    ptr.P(int32(100)),
				RefillAmount: ptr.P(int32(200)),
				RefillDay:    ptr.P(int16(12)),
			})

			// Update only remaining
			req := handler.Request{
				KeyId: keyResponse.KeyID,
				Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
					Remaining: nullable.NewNullableWithValue(int64(999)),
				}),
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			// Verify remaining updated, refill unchanged
			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 999, key.RemainingRequests.Int32, "remaining should be updated")
			require.True(t, key.RefillAmount.Valid)
			require.EqualValues(t, 200, key.RefillAmount.Int32, "refill amount should be unchanged")
			require.True(t, key.RefillDay.Valid)
			require.EqualValues(t, 12, key.RefillDay.Int16, "refill day should be unchanged")
		})

		// Test Case 6: Undefined credits (not specified) keeps everything
		t.Run("Undefined credits keeps everything", func(t *testing.T) {
			t.Parallel()
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID:  h.Resources().UserWorkspace.ID,
				KeySpaceID:   api.KeyAuthID.String,
				Name:         ptr.P("undefined-credits-test"),
				Remaining:    ptr.P(int32(777)),
				RefillAmount: ptr.P(int32(111)),
				RefillDay:    ptr.P(int16(28)),
			})

			// Update something else without specifying credits
			req := handler.Request{
				KeyId:   keyResponse.KeyID,
				Enabled: ptr.P(false),
				// Credits not specified at all
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			// Verify credits unchanged
			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.False(t, key.Enabled, "enabled should be updated")
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 777, key.RemainingRequests.Int32, "remaining should be unchanged")
			require.True(t, key.RefillAmount.Valid)
			require.EqualValues(t, 111, key.RefillAmount.Int32, "refill amount should be unchanged")
			require.True(t, key.RefillDay.Valid)
			require.EqualValues(t, 28, key.RefillDay.Int16, "refill day should be unchanged")
		})

		// Test Case 7: Daily refill (no refillDay)
		t.Run("Daily refill without refillDay", func(t *testing.T) {
			t.Parallel()
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: h.Resources().UserWorkspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				Name:        ptr.P("daily-refill-test"),
				Remaining:   ptr.P(int32(50)),
			})

			// Set daily refill
			req := handler.Request{
				KeyId: keyResponse.KeyID,
				Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
					Refill: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsRefill{
						Amount:   25,
						Interval: openapi.UpdateKeyCreditsRefillIntervalDaily,
						// No RefillDay for daily
					}),
				}),
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			// Verify daily refill set correctly
			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 50, key.RemainingRequests.Int32)
			require.True(t, key.RefillAmount.Valid)
			require.EqualValues(t, 25, key.RefillAmount.Int32)
			require.False(t, key.RefillDay.Valid, "refill day should be null for daily interval")
		})

		// Test Case 8: Complex sequence of updates
		t.Run("Complex sequence of updates", func(t *testing.T) {
			t.Parallel()
			keyResponse := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: h.Resources().UserWorkspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				Name:        ptr.P("complex-sequence"),
			})

			// Step 1: Add credits with refill
			req := handler.Request{
				KeyId: keyResponse.KeyID,
				Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
					Remaining: nullable.NewNullableWithValue(int64(1000)),
					Refill: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsRefill{
						Amount:    500,
						Interval:  openapi.UpdateKeyCreditsRefillIntervalMonthly,
						RefillDay: ptr.P(1),
					}),
				}),
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			// Step 2: Clear refill only
			req = handler.Request{
				KeyId: keyResponse.KeyID,
				Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
					Refill: nullable.NewNullNullable[openapi.UpdateKeyCreditsRefill](),
				}),
			}
			res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 1000, key.RemainingRequests.Int32, "remaining should still be 1000")
			require.False(t, key.RefillAmount.Valid, "refill should be cleared")

			// Step 3: Add back refill without changing remaining
			req = handler.Request{
				KeyId: keyResponse.KeyID,
				Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
					Refill: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsRefill{
						Amount:   250,
						Interval: openapi.UpdateKeyCreditsRefillIntervalDaily,
					}),
				}),
			}
			res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.True(t, key.RemainingRequests.Valid)
			require.EqualValues(t, 1000, key.RemainingRequests.Int32, "remaining should still be 1000")
			require.True(t, key.RefillAmount.Valid)
			require.EqualValues(t, 250, key.RefillAmount.Int32, "refill should be 250")

			// Step 4: Clear everything
			req = handler.Request{
				KeyId:   keyResponse.KeyID,
				Credits: nullable.NewNullNullable[openapi.UpdateKeyCreditsData](),
			}
			res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)

			key, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
			require.NoError(t, err)
			require.False(t, key.RemainingRequests.Valid, "remaining should be cleared")
			require.False(t, key.RefillAmount.Valid, "refill should be cleared")
			require.False(t, key.RefillDay.Valid, "refill day should be cleared")
		})
	})

	t.Run("All fields simultaneously", func(t *testing.T) {
		t.Parallel()
		// Create key with initial values
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        ptr.P("initial-name"),
			Meta:        ptr.P(`{"initial": "value"}`),
			Expires:     ptr.P(time.Now().Add(24 * time.Hour)),
		})

		// First add an identity to the key
		req := handler.Request{
			KeyId:      keyResponse.KeyID,
			ExternalId: nullable.NewNullableWithValue("initial-user"),
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Update all fields with different three-state behaviors
		req = handler.Request{
			KeyId:      keyResponse.KeyID,
			Name:       nullable.NewNullableWithValue("updated-name"),   // Set to value
			ExternalId: nullable.NewNullNullable[string](),              // Set to NULL
			Meta:       nullable.NewNullableWithValue(map[string]any{}), // Set to empty object
			Expires:    nullable.NewNullNullable[int64](),               // Set to NULL
			Enabled:    ptr.P(false),                                    // Set to specific value
		}

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// Verify all fields were updated correctly
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
		require.NoError(t, err)

		// Name should be updated
		require.True(t, key.Name.Valid)
		require.Equal(t, "updated-name", key.Name.String)

		// Identity should be cleared
		require.False(t, key.IdentityID.Valid)

		// Meta should be set to empty object
		require.True(t, key.Meta.Valid)
		require.Equal(t, "{}", key.Meta.String)

		// Expires should be cleared
		require.False(t, key.Expires.Valid)

		// Enabled should be set to false
		require.False(t, key.Enabled)
	})
}
