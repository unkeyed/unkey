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
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_get_api"
)

func TestGetApiSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
		Caches: h.Caches,
	}

	h.Register(route)

	// Test with existing API
	t.Run("get existing api", func(t *testing.T) {
		// Create a root key with right permissions
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		apiName := "test-get-existing-api"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        &apiName,
		})

		// Make the request to get the API
		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{
				ApiId: api.ID,
			},
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, api.ID, res.Body.Data.Id)
		require.Equal(t, apiName, res.Body.Data.Name)
	})

	// Test with different authorization scopes
	t.Run("authorization scopes", func(t *testing.T) {
		apiName := "test-get-existing-api"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        &apiName,
		})

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
				permissions:    []string{fmt.Sprintf("api.%s.read_api", api.ID)},
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
						ApiId: api.ID,
					},
				)

				require.Equal(t, tc.expectedStatus, res.Status, "expected %d, received: %#v", tc.expectedStatus, res)
				if tc.expectedStatus == 200 {
					require.NotNil(t, res.Body)
					require.Equal(t, api.ID, res.Body.Data.Id)
					require.Equal(t, apiName, res.Body.Data.Name)
				}
			})
		}
	})

	// Test API with very long name
	t.Run("get api with very long name", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Create API with a very long name
		apiName := "this-is-a-very-long-api-name-for-testing-the-limits-of-what-the-system-can-handle-when-dealing-with-extremely-verbose-identifiers-that-might-challenge-database-storage-ui-rendering-and-overall-system-performance-with-edge-cases"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        &apiName,
		})
		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{
				ApiId: api.ID,
			},
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, api.ID, res.Body.Data.Id)
		require.Equal(t, apiName, res.Body.Data.Name, "The long name should be returned exactly as stored")
	})

	// Test API with special characters in name
	t.Run("get api with special characters in name", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Create API with special characters in name
		apiName := "special!@#$%^&*()_+-=[]{}|;:,.<>?/~` characters"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        &apiName,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{
				ApiId: api.ID,
			},
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, api.ID, res.Body.Data.Id)
		require.Equal(t, apiName, res.Body.Data.Name, "Special characters should be preserved in the name")
	})

	// Test API with Unicode characters in name
	t.Run("get api with unicode characters in name", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Create API with Unicode characters in name
		apiName := "Unicode 测试 API 名称 🔑 🔒 ✅"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        &apiName,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{
				ApiId: api.ID,
			},
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, api.ID, res.Body.Data.Id)
		require.Equal(t, apiName, res.Body.Data.Name, "Unicode characters should be preserved in the name")
	})

	// Test retrieving a recently created API with timestamp verification
	t.Run("get recently created api with timestamp verification", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Record current time just before API creation
		creationTime := time.Now().UnixMilli()

		// Create a new API
		apiName := "recent-api"
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        &apiName,
			CreatedAt:   &creationTime,
		})

		// Immediately retrieve the API
		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{
				ApiId: api.ID,
			},
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, api.ID, res.Body.Data.Id)
		require.Equal(t, apiName, res.Body.Data.Name)

		// Verify in database that timestamp is correct
		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, creationTime, api.CreatedAtM, "Creation timestamp should match")
	})

	// Test API with delete protection and verify all fields
	t.Run("get api with complete data verification", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		apiName := "complete-verification-api"
		creationTime := time.Now().UnixMilli()
		createdApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   h.Resources().UserWorkspace.ID,
			Name:          &apiName,
			EncryptedKeys: true,
			CreatedAt:     &creationTime,
			DefaultPrefix: ptr.P("test_"),
			DefaultBytes:  ptr.P(int32(16)),
		})

		// Set delete protection after API creation
		err := db.Query.UpdateApiDeleteProtection(ctx, h.DB.RW(), db.UpdateApiDeleteProtectionParams{
			ApiID:            createdApi.ID,
			DeleteProtection: sql.NullBool{Valid: true, Bool: true},
		})
		require.NoError(t, err)

		// Retrieve the API
		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{
				ApiId: createdApi.ID,
			},
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, createdApi.ID, res.Body.Data.Id)
		require.Equal(t, apiName, res.Body.Data.Name)

		// Verify database record matches exactly what's returned
		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), createdApi.ID)
		require.NoError(t, err)

		// Verify core fields
		require.Equal(t, api.ID, res.Body.Data.Id)
		require.Equal(t, api.Name, res.Body.Data.Name)

		// More detailed verification would be done here if the response included more fields
		// For now, verify the database has the expected values
		require.Equal(t, h.Resources().UserWorkspace.ID, api.WorkspaceID)
		require.True(t, api.AuthType.Valid)
		require.Equal(t, db.ApisAuthTypeKey, api.AuthType.ApisAuthType)
		require.True(t, api.KeyAuthID.Valid)
		require.Equal(t, createdApi.KeyAuthID.String, api.KeyAuthID.String)
		require.Equal(t, creationTime, api.CreatedAtM)
		require.True(t, api.DeleteProtection.Valid)
		require.True(t, api.DeleteProtection.Bool)
		require.False(t, api.DeletedAtM.Valid)
	})
}
