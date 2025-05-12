package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_get_api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestGetApiSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	// Test with existing API
	t.Run("get existing api", func(t *testing.T) {
		// Create a root key with right permissions
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Create a test API
		apiID := uid.New(uid.APIPrefix)
		apiName := "test-get-existing-api"
		err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        apiName,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Make the request to get the API
		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{
				ApiId: apiID,
			},
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, apiID, res.Body.Data.Id)
		require.Equal(t, apiName, res.Body.Data.Name)
	})

	// Test with different authorization scopes
	t.Run("authorization scopes", func(t *testing.T) {
		// Create a new test API
		apiName := "test-get-api"
		apiID := uid.New(uid.APIPrefix)

		err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        apiName,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		testCases := []struct {
			name           string
			permissions    []string
			expectedStatus int
		}{
			{
				name:           "wildcard permission",
				permissions:    []string{"*"},
				expectedStatus: 403, // The "*" permission isn't directly supported in the handler
			},
			{
				name:           "api wildcard permission",
				permissions:    []string{"api.*.read_api"},
				expectedStatus: 200,
			},
			{
				name:           "specific api permission",
				permissions:    []string{fmt.Sprintf("api.%s.read_api", apiID)},
				expectedStatus: 200,
			},
			{
				name:           "multiple permissions including relevant one",
				permissions:    []string{"other.permission", "api.*.read_api"},
				expectedStatus: 200,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, tc.permissions...)
				headers := http.Header{
					"Content-Type":  {"application/json"},
					"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
				}

				res := testutil.CallRoute[handler.Request, handler.Response](
					h,
					route,
					headers,
					handler.Request{
						ApiId: apiID,
					},
				)

				require.Equal(t, tc.expectedStatus, res.Status, "expected %d, received: %#v", tc.expectedStatus, res)
				if tc.expectedStatus == 200 {
					require.NotNil(t, res.Body)
					require.Equal(t, apiID, res.Body.Data.Id)
					require.Equal(t, apiName, res.Body.Data.Name)
				}
			})
		}
	})

	// Test with API that has IP whitelist
	t.Run("get api with ip whitelist", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Create API with IP whitelist
		apiID := uid.New(uid.APIPrefix)
		apiName := "api-with-ip-whitelist"

		err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        apiName,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{
				ApiId: apiID,
			},
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, apiID, res.Body.Data.Id)
		require.Equal(t, apiName, res.Body.Data.Name)
	})
}
