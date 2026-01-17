package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_delete_api"
)

func TestDeleteProtection(t *testing.T) {
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

	// Test case for deleting an API with delete protection enabled
	t.Run("delete protected API", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		err := db.Query.UpdateApiDeleteProtection(ctx, h.DB.RW(), db.UpdateApiDeleteProtectionParams{
			ApiID:            api.ID,
			DeleteProtection: sql.NullBool{Valid: true, Bool: true},
		})
		require.NoError(t, err)

		// Ensure API exists and has delete protection
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.True(t, apiBeforeDelete.DeleteProtection.Valid)
		require.True(t, apiBeforeDelete.DeleteProtection.Bool)

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

		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "This API has delete protection enabled. Disable it before attempting to delete.", res.Body.Error.Detail)

		// Verify API was NOT deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.False(t, apiAfterDelete.DeletedAtM.Valid, "API should not have been deleted")
	})
}
