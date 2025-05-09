package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_delete_api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestDeleteProtection(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
		Caches:      h.Caches,
	})

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

	// Test case for deleting an API with delete protection enabled
	t.Run("delete protected API", func(t *testing.T) {
		// Create an API with delete protection
		api, err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			Name:             "Protected API",
			WorkspaceID:      workspace.ID,
			DeleteProtection: true,
		})
		require.NoError(t, err)

		// Ensure API exists and has delete protection
		apiBeforeDelete, err := db.Query.FindApiById(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.True(t, apiBeforeDelete.DeleteProtection)

		// Attempt to delete the API
		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should get a 429 Too Many Requests error for delete protected APIs
		require.Equal(t, 429, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "protected from deletion")

		// Verify API was NOT deleted
		apiAfterDelete, err := db.Query.FindApiById(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.False(t, apiAfterDelete.DeletedAtM.Valid, "API should not have been deleted")
	})
}
