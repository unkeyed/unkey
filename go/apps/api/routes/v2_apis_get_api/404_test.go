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

func TestGetApiNotFound(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

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
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/api_not_found", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "The requested API does not exist or has been deleted")
	})

	// Test with API from different workspace
	t.Run("api from different workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)

		// Create API in the different workspace
		apiID := uid.New(uid.APIPrefix)
		err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        "other-workspace-api",
			WorkspaceID: otherWorkspaceID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				ApiId: apiID,
			},
		)

		require.Equal(t, 404, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/api_not_found", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "The requested API does not exist or has been deleted")
	})

	// Note: We can't easily test with deleted API since we don't have direct access to delete APIs

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
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "POST request body for '/v2/apis.getApi' failed to validate schema")
	})
}
