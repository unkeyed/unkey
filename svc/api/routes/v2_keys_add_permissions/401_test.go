package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_add_permissions"
)

func TestAuthenticationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		Disabled:    false,
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Remaining:   nil,
		IdentityID:  nil,
		Meta:        nil,
	})

	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspace.ID,
		Name:        "documents.read.auth",
		Slug:        "documents.read.auth",
	})

	req := handler.Request{
		KeyId:       key.KeyID,
		Permissions: []string{permission.ID},
	}

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Authorization header")
	})

	t.Run("invalid bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "key")
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"invalid_format"},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "valid root key")
	})

	t.Run("empty bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer "},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "valid root key")
	})

	t.Run("disabled root key", func(t *testing.T) {
		// Use invalid root key to simulate disabled key
		rootKey := "invalid_disabled_key"

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})
}
