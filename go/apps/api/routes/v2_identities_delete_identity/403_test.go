package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestDeleteIdentityForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	t.Run("insufficient permissions - no permissions", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID) // No permissions
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("insufficient permissions - wrong permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity") // Wrong permission
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("insufficient permissions - different resource permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "key.*.delete_key") // Different resource type
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("read-only permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity") // Read permission instead of delete
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("partial permission match", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.create_identity") // Missing wildcard/specific ID
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("multiple permissions but none matching", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID,
			"key.*.delete_key",
			"api.*.delete_api",
			"workspace.*.read_workspace") // Multiple permissions but none for identity deletion
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Identity: uid.New("test"),
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("case sensitivity test", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "IDENTITY.*.DELETE_IDENTITY") // Wrong case
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
