package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_get_api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestGetApiInsufficientPermissions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	// Create a test API
	apiID := uid.New(uid.APIPrefix)
	err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "test-api-permissions",
		WorkspaceID: h.Resources().UserWorkspace.ID,
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

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
			permissions: []string{fmt.Sprintf("api.%s.create_api", apiID)},
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

			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
				h,
				route,
				headers,
				handler.Request{
					ApiId: apiID,
				},
			)

			require.Equal(t, 403, res.Status, "expected 403, received: %#v", res)
			require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
			require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions:")
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
				ApiId: apiID,
			},
		)

		require.Equal(t, 404, res.Status, "expected 404, received: %#v", res)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/workspace_not_found", res.Body.Error.Type)
		require.Equal(t, "The provided root key is invalid. The requested workspace does not exist.", res.Body.Error.Detail)
	})
}
