package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_key"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
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
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
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
			KeyAuthID:   api.KeyAuthID.String,
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
		// Create key with initial meta
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
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
		// Create key with initial expiration
		futureTime := time.Now().Add(24 * time.Hour)
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
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
		// Create key without identity initially
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
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
			ID: key.IdentityID.String,
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
			ID: key.IdentityID.String,
		})
		require.NoError(t, err)
		require.Equal(t, "preserved-user", identity.ExternalID)
	})

	t.Run("All fields simultaneously", func(t *testing.T) {
		// Create key with initial values
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
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
			Name:       nullable.NewNullableWithValue("updated-name"),           // Set to value
			ExternalId: nullable.NewNullNullable[string](),                      // Set to NULL
			Meta:       nullable.NewNullableWithValue(map[string]interface{}{}), // Set to empty object
			Expires:    nullable.NewNullNullable[int64](),                       // Set to NULL
			Enabled:    ptr.P(false),                                            // Set to specific value
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
