package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestDeleteIdentityForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	t.Run("insufficient permissions - no permissions", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID) // No permissions
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("insufficient permissions - wrong permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity") // Wrong permission
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("insufficient permissions - different resource permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "key.*.delete_key") // Different resource type
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("specific identity permission for wrong identity", func(t *testing.T) {
		// Create a test identity
		identityId := uid.New(uid.IdentityPrefix)
		err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
			ID:          identityId,
			ExternalID:  "ext_" + identityId,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		// Create a different identity ID
		differentIdentityId := uid.New(uid.IdentityPrefix)

		// Create root key with permission for the different identity
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("identity.%s.delete_identity", differentIdentityId))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{IdentityId: ptr.P(identityId)}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("read-only permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity") // Read permission instead of delete
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("partial permission match", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.create_identity") // Missing wildcard/specific ID
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
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

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("case sensitivity test", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "IDENTITY.*.DELETE_IDENTITY") // Wrong case
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
		require.Equal(t, "Insufficient Permissions", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})
}