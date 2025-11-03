package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_delete_api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestSuccess(t *testing.T) {
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

	// Test case for deleting an API without keys
	t.Run("delete api without keys", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		// Ensure API exists before deletion
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)

		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

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
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify API is marked as deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err) // Should still find it, just marked as deleted
		require.True(t, apiAfterDelete.DeletedAtM.Valid)
	})

	// Test case for deleting an API with active keys
	t.Run("delete api with active keys", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		createKey := h.CreateKey(seed.CreateKeyRequest{
			KeySpaceID:  api.KeyAuthID.String,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})

		// Ensure API exists before deletion
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

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
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify API is marked as deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.True(t, apiAfterDelete.DeletedAtM.Valid)

		// Check that the key is still accessible (soft delete doesn't cascade to keys)
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), createKey.KeyID)
		require.NoError(t, err)
		require.Equal(t, createKey.KeyID, key.ID)
	})

	// Test case for deleting an API immediately after creation
	t.Run("delete api immediately after creation", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		// Verify the API was created
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.Equal(t, api.Name, apiBeforeDelete.Name)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

		// Immediately delete the API without any delay
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
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify API is marked as deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.True(t, apiAfterDelete.DeletedAtM.Valid)
	})
}
