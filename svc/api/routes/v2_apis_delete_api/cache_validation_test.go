package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_delete_api"
)

func TestCacheInvalidation(t *testing.T) {
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

	// Test case for verifying cache invalidation
	t.Run("verify cache invalidation", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		// Get API to ensure it's in the cache
		_, hit, err := h.Caches.LiveApiByID.SWR(ctx, cache.ScopedKey{WorkspaceID: workspace.ID, Key: api.ID}, func(ctx context.Context) (db.FindLiveApiByIDRow, error) {
			return db.Query.FindLiveApiByID(ctx, h.DB.RO(), api.ID)
		}, caches.DefaultFindFirstOp)
		require.NoError(t, err)
		require.Equal(t, cache.Hit, hit)
		// Delete the API
		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)

		// Verify API is soft-deleted in the database
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.True(t, apiAfterDelete.DeletedAtM.Valid)

		// Verify the API is deleted in the cache
		_, hit = h.Caches.LiveApiByID.Get(ctx, cache.ScopedKey{WorkspaceID: workspace.ID, Key: api.ID})
		require.Equal(t, cache.Null, hit)
	})
}
