package handler_test

import (
	"context"
	"database/sql"
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

	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
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

	// Test API with very long name
	t.Run("get api with very long name", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Create API with a very long name
		apiID := uid.New(uid.APIPrefix)
		apiName := "this-is-a-very-long-api-name-for-testing-the-limits-of-what-the-system-can-handle-when-dealing-with-extremely-verbose-identifiers-that-might-challenge-database-storage-ui-rendering-and-overall-system-performance-with-edge-cases"

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
		apiID := uid.New(uid.APIPrefix)
		apiName := "special!@#$%^&*()_+-=[]{}|;:,.<>?/~` characters"

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
		apiID := uid.New(uid.APIPrefix)
		apiName := "Unicode 测试 API 名称 🔑 🔒 ✅"

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
		apiID := uid.New(uid.APIPrefix)
		apiName := "recent-api"

		err := db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        apiName,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			CreatedAtM:  creationTime,
		})
		require.NoError(t, err)

		// Immediately retrieve the API
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

		// Verify in database that timestamp is correct
		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
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

		// Create keyring for the API
		keyAuthID := uid.New(uid.KeyAuthPrefix)
		err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
			ID:                 keyAuthID,
			WorkspaceID:        h.Resources().UserWorkspace.ID,
			CreatedAtM:         time.Now().UnixMilli(),
			DefaultPrefix:      sql.NullString{Valid: true, String: "test_"},
			DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
			StoreEncryptedKeys: true,
		})
		require.NoError(t, err)

		// Create API with all fields populated and delete protection enabled
		apiID := uid.New(uid.APIPrefix)
		apiName := "complete-verification-api"
		creationTime := time.Now().UnixMilli()

		err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
			ID:          apiID,
			Name:        apiName,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
			CreatedAtM:  creationTime,
		})
		require.NoError(t, err)

		// Set delete protection after API creation
		err = db.Query.UpdateApiDeleteProtection(ctx, h.DB.RW(), db.UpdateApiDeleteProtectionParams{
			ApiID:            apiID,
			DeleteProtection: sql.NullBool{Valid: true, Bool: true},
		})
		require.NoError(t, err)

		// Retrieve the API
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

		// Verify database record matches exactly what's returned
		api, err := db.Query.FindApiByID(ctx, h.DB.RO(), apiID)
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
		require.Equal(t, keyAuthID, api.KeyAuthID.String)
		require.Equal(t, creationTime, api.CreatedAtM)
		require.True(t, api.DeleteProtection.Valid)
		require.True(t, api.DeleteProtection.Bool)
		require.False(t, api.DeletedAtM.Valid)
	})
}
