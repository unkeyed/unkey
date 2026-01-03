package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_apis_delete_api"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
)

func TestNotFoundErrors(t *testing.T) {
	ctx := context.Background()
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

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.delete_api")

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
}
