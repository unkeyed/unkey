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
	identity := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: uid.New(uid.TestPrefix)})
	otherProjectID := uid.New(uid.ProjectPrefix)
	h.CreateProject(seed.CreateProjectRequest{
		ID:          otherProjectID,
		WorkspaceID: workspaceID,
		Name:        "Other",
		Slug:        uid.New("other"),
	})
	otherExternalID := uid.New(uid.TestPrefix)
	err := db.Query.InsertIdentity(context.Background(), h.DB.RW(), db.InsertIdentityParams{
		ID:          uid.New(uid.IdentityPrefix),
		ExternalID:  otherExternalID,
		WorkspaceID: workspaceID,
		ProjectID:   otherProjectID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	})
	require.NoError(t, err)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:projects/%s/identities/*#read_identity", workspaceID, identity.ProjectID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "got: %s", res.RawBody)
	found := false
	for _, listedIdentity := range res.Body.Data {
		found = found || listedIdentity.ExternalId == identity.ExternalID
		require.NotEqual(t, otherExternalID, listedIdentity.ExternalId)
	}
	require.True(t, found)
}
