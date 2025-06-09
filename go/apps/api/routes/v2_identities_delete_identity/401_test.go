package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestDeleteIdentityUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// No Authorization header
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "Authorization header")
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authentication/malformed", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "Bearer")
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("invalid auth token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authentication/key_not_found", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "key")
		require.Equal(t, http.StatusUnauthorized, res.Body.Error.Status)
		require.Equal(t, "Unauthorized", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("empty bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer "},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authentication/malformed", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "Bearer")
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("bearer token with invalid format", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer not-a-valid-key-format"},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authentication/key_not_found", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "key")
		require.Equal(t, http.StatusUnauthorized, res.Body.Error.Status)
		require.Equal(t, "Unauthorized", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("token with wrong format", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer random_string_not_a_key"},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authentication/key_not_found", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "key")
		require.Equal(t, http.StatusUnauthorized, res.Body.Error.Status)
		require.Equal(t, "Unauthorized", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("key from wrong workspace", func(t *testing.T) {
		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		// Create a root key for different workspace
		differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID, "identity.*.delete_identity")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "identity does not exist")
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("completely invalid token format", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer 123"},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authentication/key_not_found", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "key")
		require.Equal(t, http.StatusUnauthorized, res.Body.Error.Status)
		require.Equal(t, "Unauthorized", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})
}
