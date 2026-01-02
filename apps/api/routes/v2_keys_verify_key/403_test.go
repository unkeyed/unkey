package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
)

func TestForbidden_NoVerifyPermissions(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	t.Run("root key with no verify permissions returns 403", func(t *testing.T) {
		// Create root key with a permission that is NOT verify_key
		rootKeyWithoutVerify := h.CreateRootKey(workspace.ID, "api.*.read_key")

		req := handler.Request{
			Key: key.Key,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyWithoutVerify)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status, "expected 403, received: %d", res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.NotEmpty(t, res.Body.Meta.RequestId, "RequestId should be returned in error response")

		// Verify the error message mentions both permission options
		require.Contains(t, res.Body.Error.Detail, "api.*.verify_key", "error should mention wildcard permission option")
		require.Contains(t, res.Body.Error.Detail, "api.<API_ID>.verify_key", "error should mention specific API permission option")
	})

	t.Run("root key with verify permission for different api returns 200 NOT_FOUND (not 403)", func(t *testing.T) {
		// Create a second API
		api2 := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

		// Create root key with verify permission for api2 only
		rootKeyForApi2 := h.CreateRootKey(workspace.ID, fmt.Sprintf("api.%s.verify_key", api2.ID))

		// Try to verify a key from api1
		req := handler.Request{
			Key: key.Key,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyForApi2)},
		}

		// Should return 200 with NOT_FOUND (not 403) to avoid leaking key existence
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %d", res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.NOTFOUND, res.Body.Data.Code, "should return NOT_FOUND to avoid leaking key existence")
		require.False(t, res.Body.Data.Valid)
	})

	t.Run("root key with wildcard verify permission returns 200 VALID", func(t *testing.T) {
		// Create root key with wildcard verify permission
		rootKeyWithVerify := h.CreateRootKey(workspace.ID, "api.*.verify_key")

		req := handler.Request{
			Key: key.Key,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyWithVerify)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %d", res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid)
	})

	t.Run("root key with specific api verify permission returns 200 VALID", func(t *testing.T) {
		// Create root key with specific API verify permission
		rootKeyWithSpecificVerify := h.CreateRootKey(workspace.ID, fmt.Sprintf("api.%s.verify_key", api.ID))

		req := handler.Request{
			Key: key.Key,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyWithSpecificVerify)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %d", res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid)
	})
}
