package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_delete_api"
)

func TestDeleteAPIHidesUnreadableExistence(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*#read", workspace.ID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	absent := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{ApiId: uid.New(uid.APIPrefix)})
	require.Equal(t, http.StatusNotFound, absent.Status)
	existing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{ApiId: api.ID})
	require.Equal(t, absent.Status, existing.Status, "received: %s", existing.RawBody)
	require.Equal(t, absent.Body.Error, existing.Body.Error)
	require.NotContains(t, existing.RawBody, api.KeyAuthID.String)
	notDeleted, err := db.Query.FindApiByID(context.Background(), h.DB.RO(), api.ID)
	require.NoError(t, err)
	require.False(t, notDeleted.DeletedAtM.Valid)
}

func TestNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*/keyspaces/*#delete", workspace.ID))

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for non-existent API
	t.Run("non-existent API", func(t *testing.T) {
		req := handler.Request{
			ApiId: uid.New(uid.APIPrefix),
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
		require.Equal(t, "The requested API does not exist or has been deleted.", res.Body.Error.Detail)
	})

	// Test case for API that's already deleted
	t.Run("already deleted API", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		err := db.Query.SoftDeleteApi(ctx, h.DB.RW(), db.SoftDeleteApiParams{
			ApiID: api.ID,
			Now:   sql.NullInt64{Valid: true, Int64: h.Clock.Now().UnixMilli()},
		})
		require.NoError(t, err)

		// Try to delete it again
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
		require.Equal(t, "The requested API does not exist or has been deleted.", res.Body.Error.Detail)
	})

	t.Run("API belongs to different workspace", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: otherWorkspace.ID})

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
		require.Equal(t, "The requested API does not exist or has been deleted.", res.Body.Error.Detail)
	})
}
