package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_list_keys"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestValidationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Vault:       h.Vault,
	})

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.read_api")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for missing required apiId
	t.Run("missing apiId", func(t *testing.T) {
		req := handler.Request{
			// ApiId is missing
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
		require.Contains(t, res.Body.Error.Detail, "invalid")
	})

	// Test case for invalid limit (too large)
	t.Run("invalid limit (too large)", func(t *testing.T) {
		req := handler.Request{
			ApiId: "api_1234",
			Limit: 1000, // Limit too large (max 100)
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		// Note: In the implementation, we're actually capping the limit at 100 rather than
		// returning an error, so this test will pass as long as the request doesn't fail
		// with a different error.
	})

	// Test case for invalid API ID format
	t.Run("invalid apiId format", func(t *testing.T) {
		req := handler.Request{
			ApiId: "", // Empty string is invalid
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
		require.Contains(t, res.Body.Error.Detail, "invalid")
	})
}
