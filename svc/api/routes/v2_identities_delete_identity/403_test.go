package handler_test

import (
	"fmt"
	"net/http"
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
// find or delete identities without access to the identity.
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

	for _, tt := range []struct {
		name        string
		permissions []string
	}{
		{name: "no permissions", permissions: nil},
		{
			name:        "wrong action",
			permissions: []string{rbac.U(urn.New().Workspace(identity.WorkspaceID).Project(identity.ProjectID).Identity(identity.ID), permissions.Write).Value},
		},
		{
			name:        "different identity",
			permissions: []string{rbac.U(urn.New().Workspace(identity.WorkspaceID).Project(identity.ProjectID).Identity(uid.New(uid.IdentityPrefix)), permissions.Delete).Value},
		},
		{
			name:        "different project",
			permissions: []string{rbac.U(urn.New().Workspace(identity.WorkspaceID).Project(uid.New(uid.ProjectPrefix)).Identity("*"), permissions.Delete).Value},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, tt.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Identity: externalID})
			require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
			missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Identity: uid.New(uid.TestPrefix)})
			require.Equal(t, http.StatusNotFound, missing.Status)
			require.Equal(t, missing.Body.Error, res.Body.Error)
			require.NotContains(t, res.RawBody, identity.ID)
			require.NotContains(t, res.RawBody, externalID)
		})
	}

	t.Run("read access names the missing permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, rbac.U(urn.New().Workspace(h.Resources().UserWorkspace.ID).Project(identity.ProjectID).Identity(identity.ID), permissions.Read).Value)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{Identity: externalID}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, sent: %+v, received: %s", req, res.RawBody)
		require.Contains(t, res.Body.Error.Detail, "#delete")
		require.Contains(t, res.Body.Error.Detail, "Log in again")
		require.NotContains(t, res.RawBody, identity.ID)
	})

	_, err := db.Query.FindIdentityByID(t.Context(), h.DB.RO(), db.FindIdentityByIDParams{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		IdentityID:  identity.ID,
		Deleted:     false,
	})
	require.NoError(t, err)
}
