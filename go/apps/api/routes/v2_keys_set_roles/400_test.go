package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_roles"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		LiveKeyCache: h.Caches.LiveKeyByID,
	}

	h.Register(route)

	// Create a workspace and root key
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for missing keyId
	t.Run("missing keyId", func(t *testing.T) {
		req := map[string]interface{}{
			"roles": []map[string]interface{}{
				{"id": "role_123"},
			},
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for missing roles
	t.Run("missing roles", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": "key_123",
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for invalid keyId format
	t.Run("invalid keyId format", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": "ab", // too short
			"roles": []map[string]interface{}{
				{"id": "role_123"},
			},
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for malformed JSON body
	t.Run("malformed JSON body", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": "key_123",
			"roles": "invalid_not_array",
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

}
