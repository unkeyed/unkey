package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_list_keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
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

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.read_api")

	externalID := uid.New("test_external_id")
	// Create a test identity
	err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          uid.New("test_identity"),
		ExternalID:  externalID,
		WorkspaceID: workspace.ID,
		Environment: "",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte{},
	})
	require.NoError(t, err)

	// Create test keys
	testKeys := []db.InsertKeyParams{
		{
			Start:       "key1_",
			KeyAuthID:   keyAuth.ID,
			WorkspaceID: workspace.ID,
			Name:        db.NewNullString("Test Key 1"),
			IdentityID:  db.NewNullString(identity.ID),
		},
		{
			Start:       "key2_",
			KeyAuthID:   keyAuth.ID,
			WorkspaceID: workspace.ID,
			Name:        db.NewNullString("Test Key 2"),
			IdentityID:  db.NewNullString(identity.ID),
		},
		{
			Start:       "key3_",
			KeyAuthID:   keyAuth.ID,
			WorkspaceID: workspace.ID,
			Name:        db.NewNullString("Test Key 3"),
		},
	}

	for _, params := range testKeys {
		_, err := db.Query.InsertKey(ctx, h.DB.RW(), params)
		require.NoError(t, err)
	}

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for listing all keys
	t.Run("list all keys", func(t *testing.T) {
		req := handler.Request{
			ApiId: api.ID,
			Limit: 100,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data.Keys, 3)
		require.Equal(t, 3, res.Body.Data.Total)
	})

	// Test case for limiting results
	t.Run("limit results", func(t *testing.T) {
		req := handler.Request{
			ApiId: api.ID,
			Limit: 2,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data.Keys, 2)
		require.Equal(t, 3, res.Body.Data.Total)
		require.NotNil(t, res.Body.Data.Cursor)
	})

	// Test case for filtering by external ID
	t.Run("filter by external ID", func(t *testing.T) {
		externalID := "test-external-id"
		req := handler.Request{
			ApiId:      api.ID,
			ExternalId: &externalID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data.Keys, 2)
		for _, key := range res.Body.Data.Keys {
			require.NotNil(t, key.Identity)
			require.Equal(t, externalID, key.Identity.ExternalId)
		}
	})

	// Test case for pagination with cursor
	t.Run("pagination with cursor", func(t *testing.T) {
		// First page
		req1 := handler.Request{
			ApiId: api.ID,
			Limit: 2,
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, 200, res1.Status)
		require.NotNil(t, res1.Body.Data.Cursor)

		// Second page
		req2 := handler.Request{
			ApiId:  api.ID,
			Limit:  2,
			Cursor: res1.Body.Data.Cursor,
		}

		res2 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req2,
		)

		require.Equal(t, 200, res2.Status)
		require.NotNil(t, res2.Body)
		require.NotNil(t, res2.Body.Data)
		require.Len(t, res2.Body.Data.Keys, 1)
		require.Equal(t, 3, res2.Body.Data.Total)
		require.Nil(t, res2.Body.Data.Cursor) // No more pages
	})
}
