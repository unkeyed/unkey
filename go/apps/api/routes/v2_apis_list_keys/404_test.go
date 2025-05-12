package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_list_keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestNotFoundErrors(t *testing.T) {
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

	// Create workspaces
	workspace1 := h.Resources().UserWorkspace
	workspace2 := h.CreateWorkspace("other-workspace")

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace1.ID, "api.*.read_key", "api.*.read_api")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for non-existent API
	t.Run("non-existent API", func(t *testing.T) {
		req := handler.Request{
			ApiId: "api_does_not_exist",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "not found")
	})

	// Test case for API in different workspace
	t.Run("API in different workspace", func(t *testing.T) {
		// Create API in a different workspace
		otherApi, err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			Name:        "API in different workspace",
			WorkspaceID: workspace2.ID,
		})
		require.NoError(t, err)

		req := handler.Request{
			ApiId: otherApi.ID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "not found")
	})

	// Test case for deleted API
	t.Run("deleted API", func(t *testing.T) {
		// Create and then delete an API
		api, err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			Name:        "API to be deleted",
			WorkspaceID: workspace1.ID,
		})
		require.NoError(t, err)

		// Delete the API
		_, err = db.Query.DeleteApi(ctx, h.DB.RW(), api.ID)
		require.NoError(t, err)

		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "not found")
	})

	// Test case for API without KeyAuth
	t.Run("API without KeyAuth", func(t *testing.T) {
		// Create API without KeyAuth
		api, err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			Name:        "API without KeyAuth",
			WorkspaceID: workspace1.ID,
		})
		require.NoError(t, err)

		req := handler.Request{
			ApiId: api.ID,
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
		require.Equal(t, res.Body.Error.Detail, "not set up")
	})
}
