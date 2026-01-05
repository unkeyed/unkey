package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_add_roles"
)

func TestValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace and root key
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.add_role_to_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create a test API and key for valid requests using testutil helper
	defaultPrefix := "test"
	defaultBytes := int32(16)
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		DefaultPrefix: &defaultPrefix,
		DefaultBytes:  &defaultBytes,
	})

	keyName := "Test Key"
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
	})
	validKeyID := keyResponse.KeyID

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

	// Test case for empty roles array
	t.Run("empty roles array", func(t *testing.T) {
		req := handler.Request{
			KeyId: validKeyID,
			Roles: []string{},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
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
			"keyId": validKeyID,
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

	// Test case for role with empty string id
	t.Run("role with empty string", func(t *testing.T) {
		req := handler.Request{
			KeyId: validKeyID,
			Roles: []string{},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
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
