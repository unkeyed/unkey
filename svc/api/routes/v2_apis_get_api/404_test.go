package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_get_api"
)

func TestGetApiNotFound(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:     h.DB,
		Keys:   h.Keys,
		Caches: h.Caches,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test with non-existent API ID
	t.Run("non-existent api id", func(t *testing.T) {
		nonExistentApiID := uid.New(uid.APIPrefix)

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				ApiId: nonExistentApiID,
			},
		)

		require.Equal(t, 404, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/api_not_found", res.Body.Error.Type)
		require.Equal(t, "The requested API does not exist or has been deleted.", res.Body.Error.Detail)
	})

	// Test with API from different workspace
	t.Run("api from different workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)

		diffApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: otherWorkspaceID})

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				ApiId: diffApi.ID,
			},
		)

		require.Equal(t, 404, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/api_not_found", res.Body.Error.Type)
		require.Equal(t, "The requested API does not exist or has been deleted.", res.Body.Error.Detail)
	})

	// Test with soft-deleted API
	t.Run("deleted api", func(t *testing.T) {
		diffApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		// Verify it exists
		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), diffApi.ID)
		require.NoError(t, err)
		require.Equal(t, diffApi.ID, api.ID)
		require.False(t, api.DeletedAtM.Valid)

		// Mark API as deleted by setting DeletedAtM
		err = db.Query.SoftDeleteApi(ctx, h.DB.RW(), db.SoftDeleteApiParams{
			ApiID: diffApi.ID,
			Now:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		require.NoError(t, err)

		// Verify it's marked as deleted
		deletedApi, err := db.Query.FindApiByID(ctx, h.DB.RO(), diffApi.ID)
		require.NoError(t, err)
		require.True(t, deletedApi.DeletedAtM.Valid)

		// Attempt to get the deleted API
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				ApiId: diffApi.ID,
			},
		)

		require.Equal(t, 404, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/api_not_found", res.Body.Error.Type)
		require.Equal(t, "The requested API does not exist or has been deleted.", res.Body.Error.Detail)
	})

	// Test with empty API ID
	t.Run("empty api id", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				ApiId: "",
			},
		)

		require.Equal(t, 400, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "POST request body for '/v2/apis.getApi' failed to validate schema")
	})
}
