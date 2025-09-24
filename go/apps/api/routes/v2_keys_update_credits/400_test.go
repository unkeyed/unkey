package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_credits"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestKeyUpdateCreditsBadRequest(t *testing.T) {
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

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	// Create a test API and key using testutil helper
	apiName := "Test API"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
		Name:        &apiName,
	})

	keyName := "test-key"
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeyAuthID:   api.KeyAuthID.String,
		Name:        &keyName,
	})
	keyID := keyResponse.KeyID

	// Create root key with read permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing keyId", func(t *testing.T) {
		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/keys.updateCredits' failed to validate schema")
	})

	t.Run("empty keyId string", func(t *testing.T) {
		req := handler.Request{
			KeyId: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("increment with null throws error", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Increment,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("can't increment unlimited key", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(int64(1)),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("decrement with null throws error", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Decrement,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("can't decrement unlimited key", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(int64(1)),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})
}
