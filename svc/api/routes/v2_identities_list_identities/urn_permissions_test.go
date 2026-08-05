package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
)

// TestListIdentities_AuthorizesCanonicalURNPermission guarantees project-scoped
// URNs can list identities without a legacy tuple grant.
func TestListIdentities_AuthorizesCanonicalURNPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: uid.New(uid.TestPrefix)})
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:projects/*/identities/*#read_identity", workspaceID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "got: %s", res.RawBody)
}

func TestListIdentities_FiltersCanonicalURNPermissionsByProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	allowedProject := h.CreateProject(seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "Allowed", Slug: "allowed-identities",
	})
	filteredProject := h.CreateProject(seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "Filtered", Slug: "filtered-identities",
	})
	allowedIdentityID := uid.New(uid.IdentityPrefix)
	filteredIdentityID := uid.New(uid.IdentityPrefix)
	for _, identity := range []struct {
		id        string
		projectID string
		external  string
	}{
		{id: allowedIdentityID, projectID: allowedProject.ID, external: "allowed-project-identity"},
		{id: filteredIdentityID, projectID: filteredProject.ID, external: "filtered-project-identity"},
	} {
		err := db.Query.InsertIdentity(context.Background(), h.DB.RW(), db.InsertIdentityParams{
			ID: identity.id, ExternalID: identity.external, WorkspaceID: workspaceID,
			ProjectID: identity.projectID, Environment: "", CreatedAt: time.Now().UnixMilli(), Meta: []byte("{}"),
		})
		require.NoError(t, err)
	}

	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/identities/*#read_identity", workspaceID, allowedProject.ID,
	))
	headers := http.Header{
		"Content-Type": {"application/json"}, "Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "got: %s", res.RawBody)
	foundAllowedIdentity := false
	for _, identity := range res.Body.Data {
		if identity.Id == allowedIdentityID {
			foundAllowedIdentity = true
		}
		require.NotEqual(t, filteredIdentityID, identity.Id)
	}
	require.True(t, foundAllowedIdentity)
}
