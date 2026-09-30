package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_get_api"
)

func TestGetApiInsufficientPermissions(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:     h.DB,
		Caches: h.Caches,
	}

	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

	testCases := []struct {
		name        string
		permissions []string
	}{
		{
			name:        "no permissions",
			permissions: []string{},
		},
		{
			name:        "unrelated permission",
			permissions: []string{"unrelated.permission"},
		},
		{
			name:        "wrong api permission",
			permissions: []string{"api.*.create_api"},
		},
		{
			name:        "wrong scope for specific api",
			permissions: []string{fmt.Sprintf("api.%s.create_api", api.ID)},
		},
		{
			name:        "permission for different api",
			permissions: []string{fmt.Sprintf("api.%s.read_api", uid.New(uid.APIPrefix))},
		},
		{
			name:        "multiple insufficient permissions",
			permissions: []string{"key.*.create", "ratelimit.*.create", "identity.*.read"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
				h,
				route,
				headers,
				handler.Request{
					ApiId: api.ID,
				},
			)

			// Read authorization runs after the lookup so the URN arm can use the
			// keyspace, and an insufficient-permissions failure is masked as a
			// not-found error. Callers without read access must not be able to
			// distinguish an existing API from a non-existent one.
			require.Equal(t, 404, res.Status, "expected 404, received: %#v", res)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/data/api_not_found", res.Body.Error.Type)
			require.Equal(t, "The requested API does not exist or has been deleted.", res.Body.Error.Detail)
		})
	}

	// Test with a valid key but from wrong workspace
	t.Run("key from wrong workspace", func(t *testing.T) {
		// Create another workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)

		// Create key for other workspace with sufficient permissions
		rootKey := h.CreateRootKey(otherWorkspaceID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				ApiId: api.ID,
			},
		)

		require.Equal(t, 404, res.Status, "expected 404, received: %#v", res)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/workspace_not_found", res.Body.Error.Type)
		require.Equal(t, "The provided root key is invalid. The requested workspace does not exist.", res.Body.Error.Detail)
	})
}
