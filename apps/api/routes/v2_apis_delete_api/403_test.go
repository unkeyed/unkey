package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_apis_delete_api"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestAuthorizationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

	// Test case for insufficient permissions - missing delete_api
	t.Run("missing delete_api permission", func(t *testing.T) {
		// Create a root key with only read_api but no delete_api permission
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	// Test case for permission for different API
	t.Run("permission for different API", func(t *testing.T) {
		// Create a root key with permissions for a specific different API
		differentApiId := "api_different"
		rootKey := h.CreateRootKey(
			workspace.ID,
			fmt.Sprintf("api.%s.delete_api", differentApiId),
		)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		rootKey := h.CreateRootKey(uid.New(uid.WorkspacePrefix), "api.*.delete_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Note: We mask wrong workspace errors as 404 not found
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "The provided root key is invalid. The requested workspace does not exist.", res.Body.Error.Detail)
	})
}
