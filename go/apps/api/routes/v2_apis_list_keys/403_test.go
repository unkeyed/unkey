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

func TestAuthorizationErrors(t *testing.T) {
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

	// Create an API for testing
	api, err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		Name:        "Test API",
		WorkspaceID: workspace.ID,
	})
	require.NoError(t, err)

	// Create a KeyAuth for the API
	keyAuth, err := db.Query.InsertKeyAuth(ctx, h.DB.RW(), db.InsertKeyAuthParams{
		ApiID: api.ID,
	})
	require.NoError(t, err)

	// Update the API with KeyAuthID
	_, err = db.Query.UpdateApiSetKeyAuthId(ctx, h.DB.RW(), api.ID, keyAuth.ID)
	require.NoError(t, err)

	// Test case for insufficient permissions - missing read_key
	t.Run("missing read_key permission", func(t *testing.T) {
		// Create a root key with only read_api but no read_key permission
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
		require.Contains(t, res.Body.Error.Detail, "insufficient permissions")
	})

	// Test case for insufficient permissions - missing read_api
	t.Run("missing read_api permission", func(t *testing.T) {
		// Create a root key with only read_key but no read_api permission
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key")

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
		require.Contains(t, res.Body.Error.Detail, "insufficient permissions")
	})

	// Test case for permission for different API
	t.Run("permission for different API", func(t *testing.T) {
		// Create a root key with permissions for a specific different API
		differentApiId := "api_different"
		rootKey := h.CreateRootKey(
			workspace.ID,
			fmt.Sprintf("api.%s.read_key", differentApiId),
			fmt.Sprintf("api.%s.read_api", differentApiId),
		)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: api.ID, // Using the test API, not the one we have permission for
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
		require.Contains(t, res.Body.Error.Detail, "insufficient permissions")
	})

	// Test case for decrypt permission
	t.Run("missing decrypt permission", func(t *testing.T) {
		// Create a root key with read permissions but no decrypt permission
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		decrypt := true
		req := handler.Request{
			ApiId:   api.ID,
			Decrypt: &decrypt,
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
		require.Contains(t, res.Body.Error.Detail, "insufficient permissions")
	})
}
