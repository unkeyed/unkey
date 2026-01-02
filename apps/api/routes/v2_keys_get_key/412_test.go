package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_keys_get_key"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
)

func TestPreconditionError(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
		Vault:  h.Vault,
	}

	h.Register(route)

	// Create API using testutil helper
	apiName := "test-api"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        &apiName,
	})

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key", "api.*.decrypt_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	key := h.CreateKey(seed.CreateKeyRequest{
		KeySpaceID:  api.KeyAuthID.String,
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	// Test case for API ID with special characters
	t.Run("Try getting a recoverable key without being opt-in", func(t *testing.T) {
		req := handler.Request{
			Decrypt: ptr.P(true),
			KeyId:   key.KeyID,
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("api not set up for key encryption", func(t *testing.T) {
		h := testutil.NewHarness(t)

		apiName := "test-api"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        &apiName,
		})

		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key", "api.*.decrypt_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		key := h.CreateKey(seed.CreateKeyRequest{
			KeySpaceID:  api.KeyAuthID.String,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})

		route := &handler.Handler{
			Logger:    h.Logger,
			DB:        h.DB,
			Keys:      h.Keys,
			Auditlogs: h.Auditlogs,
			Vault:     h.Vault,
		}
		h.Register(route)

		req := handler.Request{
			Decrypt: ptr.P(true),
			KeyId:   key.KeyID,
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](
			h,
			route,
			headers,
			req,
		)
		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "does not support key encryption")
	})

	t.Run("vault missing when decrypt requested", func(t *testing.T) {
		h := testutil.NewHarness(t)

		// Create API using testutil helper
		apiName := "test-api"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        &apiName,
		})

		// Create a root key with appropriate permissions
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key", "api.*.decrypt_key")

		// Set up request headers
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		key := h.CreateKey(seed.CreateKeyRequest{
			KeySpaceID:  api.KeyAuthID.String,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})

		// Create route with nil vault
		routeNoVault := &handler.Handler{
			Logger:    h.Logger,
			DB:        h.DB,
			Keys:      h.Keys,
			Auditlogs: h.Auditlogs,
			Vault:     nil, // No vault
		}
		h.Register(routeNoVault)

		req := handler.Request{
			KeyId:   key.KeyID,
			Decrypt: ptr.P(true),
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, routeNoVault, headers, req)
		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Vault hasn't been set up")
	})
}
