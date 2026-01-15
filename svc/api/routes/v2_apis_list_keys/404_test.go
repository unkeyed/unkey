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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

func TestNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:   h.Logger,
		DB:       h.DB,
		Keys:     h.Keys,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}

	h.Register(route)

	// Create workspaces
	workspace1 := h.Resources().UserWorkspace
	workspace2 := h.CreateWorkspace()

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace1.ID, "api.*.read_key", "api.*.read_api")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for non-existent API
	t.Run("non-existent API", func(t *testing.T) {
		req := handler.Request{
			ApiId: "api_does_not_exist_123",
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
	})

	// Test case for API in different workspace
	t.Run("API in different workspace", func(t *testing.T) {
		// Create a keySpace for the API in the different workspace
		otherKeySpaceID := uid.New(uid.KeySpacePrefix)
		err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
			ID:            otherKeySpaceID,
			WorkspaceID:   workspace2.ID,
			CreatedAtM:    time.Now().UnixMilli(),
			DefaultPrefix: sql.NullString{Valid: false},
			DefaultBytes:  sql.NullInt32{Valid: false},
		})
		require.NoError(t, err)

		// Create API in the different workspace
		otherApiID := uid.New("api")
		err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          otherApiID,
			Name:        "API in different workspace",
			WorkspaceID: workspace2.ID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: otherKeySpaceID},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			ApiId: otherApiID,
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
	})

	// Test case for deleted API
	t.Run("deleted API", func(t *testing.T) {
		// Create a keySpace for the API
		deletedKeySpaceID := uid.New(uid.KeySpacePrefix)
		err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
			ID:            deletedKeySpaceID,
			WorkspaceID:   workspace1.ID,
			CreatedAtM:    time.Now().UnixMilli(),
			DefaultPrefix: sql.NullString{Valid: false},
			DefaultBytes:  sql.NullInt32{Valid: false},
		})
		require.NoError(t, err)

		// Create API to be deleted
		deletedApiID := uid.New("api")
		err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          deletedApiID,
			Name:        "API to be deleted",
			WorkspaceID: workspace1.ID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: deletedKeySpaceID},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Soft delete the API
		err = db.Query.SoftDeleteApi(ctx, h.DB.RW(), db.SoftDeleteApiParams{
			ApiID: deletedApiID,
			Now:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		require.NoError(t, err)

		req := handler.Request{
			ApiId: deletedApiID,
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
	})

	// Test case for API without KeySpace (should return 404 Not Found)
	t.Run("API without KeySpace", func(t *testing.T) {
		// Create API without KeySpace
		noKeySpaceApiID := uid.New("api")
		err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          noKeySpaceApiID,
			Name:        "API without KeySpace",
			WorkspaceID: workspace1.ID,
			AuthType:    db.NullApisAuthType{Valid: false}, // No auth type
			KeyAuthID:   sql.NullString{Valid: false},      // No KeySpace
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			ApiId: noKeySpaceApiID,
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
	})

	// Test case for invalid API ID format
	t.Run("invalid API ID format", func(t *testing.T) {
		req := handler.Request{
			ApiId: "invalid_format", // Doesn't start with 'api_'
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
	})

	// Test case for empty API ID
	t.Run("empty API ID", func(t *testing.T) {
		req := handler.Request{
			ApiId: "",
		}

		// Empty API ID might be caught by validation (400) or return 404
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should be either 400 (validation) or 404 (not found)
		require.True(t, res.Status == 400 || res.Status == 404)
	})

	// Test case for verifying error response structure
	t.Run("verify error response structure", func(t *testing.T) {
		req := handler.Request{
			ApiId: "api_definitely_does_not_exist",
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
		require.NotEmpty(t, res.Body.Error.Detail)
		require.Equal(t, 404, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Error.Title)

		// Verify meta information is included
		require.NotNil(t, res.Body.Meta)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	// Test case for API that exists but has no keys (should return 200 with empty array)
	t.Run("API exists but has no keys", func(t *testing.T) {
		// Create API with no keys using testutil helper
		apiName := "API with no keys"
		emptyApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: workspace1.ID,
			Name:        &apiName,
		})
		emptyApiID := emptyApi.ID

		req := handler.Request{
			ApiId: emptyApiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		// Should return 200 with empty data array, not 404
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 0)
		require.NotNil(t, res.Body.Pagination)
		require.False(t, res.Body.Pagination.HasMore)
	})

	// Test case for very long API ID
	t.Run("very long API ID", func(t *testing.T) {
		longApiId := "api_" + string(make([]byte, 500)) // Very long API ID
		req := handler.Request{
			ApiId: longApiId,
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
	})

	// Test case for API ID with special characters
	t.Run("API ID with special characters", func(t *testing.T) {
		req := handler.Request{
			ApiId: "api_special!@#$%^&*()",
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
	})
}
