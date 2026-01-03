package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_apis_list_keys"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestAuthorizationErrors(t *testing.T) {
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

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a keySpace for the API
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            keySpaceID,
		WorkspaceID:   workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	err = db.Query.UpdateKeySpaceKeyEncryption(ctx, h.DB.RW(), db.UpdateKeySpaceKeyEncryptionParams{
		ID:                 keySpaceID,
		StoreEncryptedKeys: true,
	})
	require.NoError(t, err)

	// Create a test API
	apiID := uid.New("api")
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "Test API",
		WorkspaceID: workspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Test case for insufficient permissions - missing read_key
	t.Run("missing read_key permission", func(t *testing.T) {
		// Create a root key with only read_api but no read_key permission
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	// Test case for insufficient permissions - missing read_api
	t.Run("missing read_api permission", func(t *testing.T) {
		// Create a root key with only read_key but no read_api permission
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	// Test case for permission for different API
	t.Run("permission for different API", func(t *testing.T) {
		// Create a root key with permissions for a specific different API
		differentApiId := "api_different_123"
		rootKey := h.CreateRootKey(
			workspace.ID,
			fmt.Sprintf("api.%s.read_key", differentApiId),
			fmt.Sprintf("api.%s.read_api", differentApiId),
		)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID, // Using the test API, not the one we have permission for
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	// Test case for decrypt permission
	t.Run("missing decrypt permission", func(t *testing.T) {
		// Create a root key with read permissions but no decrypt permission
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		decrypt := true
		req := handler.Request{
			ApiId:   apiID,
			Decrypt: &decrypt,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	// Test case for no permissions at all
	t.Run("no permissions", func(t *testing.T) {
		// Create a root key with no relevant permissions
		rootKey := h.CreateRootKey(workspace.ID, "workspace.read")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	// Test case for cross-workspace access attempt
	t.Run("cross workspace access", func(t *testing.T) {
		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		// Create a root key for the different workspace with full permissions
		rootKey := h.CreateRootKey(differentWorkspace.ID, "api.*.read_key", "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID, // API belongs to the original workspace
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should return 404 (not found) rather than 403 for security reasons
		// The system masks cross-workspace access as "not found"
		require.True(t, res.Status == 403 || res.Status == 404)
	})

	// Test case for wildcard permissions (should work)
	t.Run("wildcard permissions should work", func(t *testing.T) {
		// Create a root key with explicit wildcard API permissions for both required actions
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		// Should not be 403 - wildcard API permissions should work
		require.NotEqual(t, 403, res.Status)
		// Should be 200 (success) assuming the API exists and has keys to list
		require.True(t, res.Status == 200 || res.Status == 404)
	})

	// Test case for specific API permissions (should work)
	t.Run("specific API permissions should work", func(t *testing.T) {
		// Create a root key with permissions for this specific API
		rootKey := h.CreateRootKey(workspace.ID,
			fmt.Sprintf("api.%s.read_key", apiID),
			fmt.Sprintf("api.%s.read_api", apiID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		// Should not be 403 - specific permissions should work
		require.NotEqual(t, 403, res.Status)
		// Should be 200 (success)
		require.Equal(t, 200, res.Status)
	})

	// Test case for verifying error response structure
	t.Run("verify error response structure", func(t *testing.T) {
		// Create a root key with insufficient permissions
		rootKey := h.CreateRootKey(workspace.ID, "workspace.read")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.NotEmpty(t, res.Body.Error.Detail)
		require.Equal(t, 403, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Error.Title)

		// Verify meta information is included
		require.NotNil(t, res.Body.Meta)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	// Test case for partial permissions (read_api but not read_key)
	t.Run("partial permissions insufficient", func(t *testing.T) {
		// Create a root key with only one of the required permissions
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ApiId: apiID,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	// Test case for decrypt permission with wildcard
	t.Run("decrypt with wildcard permission should work", func(t *testing.T) {
		// Create a root key with wildcard API permissions (includes decrypt)
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.read_api", "api.*.decrypt_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		decrypt := true
		req := handler.Request{
			ApiId:   apiID,
			Decrypt: &decrypt,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		// Should not be 403 - wildcard API permissions should include decrypt
		require.NotEqual(t, 403, res.Status)
		// Should be 200 (success)
		require.Equal(t, 200, res.Status)
	})
}
