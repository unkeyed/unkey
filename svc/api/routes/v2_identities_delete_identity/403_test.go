package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_delete_identity"
)

func TestDeleteIdentityForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		ExternalID:  uid.New(uid.TestPrefix),
	})

	t.Run("insufficient permissions - no permissions", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspaceID) // No permissions
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: identity.ExternalID}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.NotContains(t, res.Body.Error.Detail, identity.ID)
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("insufficient permissions - wrong permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspaceID, "identity.*.create_identity") // Wrong permission
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: identity.ExternalID}
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
		rootKey := h.CreateRootKey(workspaceID, "key.*.delete_key") // Different resource type
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: identity.ExternalID}
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
		rootKey := h.CreateRootKey(workspaceID, "identity.*.read_identity") // Read permission instead of delete
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: identity.ExternalID}
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
		rootKey := h.CreateRootKey(workspaceID, "identity.create_identity") // Missing wildcard/specific ID
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: identity.ExternalID}
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
		rootKey := h.CreateRootKey(workspaceID,
			"key.*.delete_key",
			"api.*.delete_api",
			"workspace.*.read_workspace") // Multiple permissions but none for identity deletion
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Identity: identity.ExternalID,
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
		rootKey := h.CreateRootKey(workspaceID, "IDENTITY.*.DELETE_IDENTITY") // Wrong case
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: identity.ExternalID}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("missing identity does not reveal existence without permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspaceID)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{Identity: uid.New(uid.TestPrefix)},
		)
		require.Equal(t, http.StatusForbidden, res.Status)
	})
}
