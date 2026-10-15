package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_delete_identity"
)

// TestDeleteIdentityMasksInsufficientPermissions guarantees that callers cannot
// find or delete identities without delete access.
func TestDeleteIdentityMasksInsufficientPermissions(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)
	externalID := uid.New(uid.TestPrefix)
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		ExternalID:  externalID,
	})

	for _, testCase := range []struct {
		name       string
		identityID string
	}{
		{"partial identity ID", identity.ID[:len(identity.ID)-1]},
		{"identity ID case differs", strings.ToUpper(identity.ID)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(identity.WorkspaceID, rbac.U(urn.New().Workspace(identity.WorkspaceID).Project(identity.ProjectID).Identity(testCase.identityID), permissions.Delete).Value)
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer " + rootKey},
			}, handler.Request{Identity: externalID})
			require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
			require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
			require.Equal(t, "Not Found", res.Body.Error.Title)
		})
	}

	t.Run("insufficient permissions - no permissions", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID) // No permissions
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("insufficient permissions - wrong permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*/identities/*#write", h.Resources().UserWorkspace.ID))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("insufficient permissions - different resource permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*/keyspaces/*/keys/*#delete", h.Resources().UserWorkspace.ID))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("read-only permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*/identities/*#read", h.Resources().UserWorkspace.ID))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("multiple permissions but none matching", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID,
			fmt.Sprintf("unkey:v1:%s:projects/*/keyspaces/*/keys/*#delete", h.Resources().UserWorkspace.ID),
			fmt.Sprintf("unkey:v1:%s:projects/*/keyspaces/*#delete", h.Resources().UserWorkspace.ID),
			fmt.Sprintf("unkey:v1:%s:projects/*#read", h.Resources().UserWorkspace.ID))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Identity: externalID,
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	_, err := db.Query.FindIdentityByID(t.Context(), h.DB.RO(), db.FindIdentityByIDParams{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		IdentityID:  identity.ID,
		Deleted:     false,
	})
	require.NoError(t, err)
}
