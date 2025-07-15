package handler_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_key"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func Test_UpdateKey_Success(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

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

	// Create key using helper
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
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
		Enabled:    &[]bool{true}[0],
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "Expected 200, got: %d", res.Status)
	require.NotNil(t, res.Body)
	require.NotEmpty(t, res.Body.Meta.RequestId)
}

func Test_UpdateKey_UpdateAllFields(t *testing.T) {
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

	// Create key using helper
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
		Name:        ptr.P("test"),
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		KeyId:      keyResponse.KeyID,
		Name:       nullable.NewNullableWithValue("newName"),
		ExternalId: nullable.NewNullableWithValue("newExternalId"),
		Meta:       nullable.NewNullableWithValue(map[string]interface{}{"new": "meta"}),
		Expires:    nullable.NewNullNullable[int64](),
		Enabled:    &[]bool{true}[0],
		Credits: &openapi.KeyCreditsData{
			Remaining: nullable.NewNullableWithValue(int64(100)),
			Refill: &openapi.KeyCreditsRefill{
				Interval: openapi.Daily,
				Amount:   50,
			},
		},
		Permissions: &[]string{"read", "write"},
		Roles:       &[]string{"admin", "user"},
	}

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	log.Printf("Response body %#v", res.Body)
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
		ID: key.IdentityID.String,
	})
	require.NoError(t, err)
	require.Equal(t, "newExternalId", identity.ExternalID)
}
