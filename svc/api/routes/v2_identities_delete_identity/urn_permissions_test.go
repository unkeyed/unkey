package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_delete_identity"
)

// TestDeleteIdentity_AuthorizesCanonicalURNPermission guarantees an exact
// identity URN can delete a request addressed by external ID.
func TestDeleteIdentity_AuthorizesCanonicalURNPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	externalID := uid.New(uid.TestPrefix)
	identity := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: externalID})
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:identities/%s#delete_identity", workspaceID, identity.ID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Identity: externalID})
	require.Equal(t, http.StatusOK, res.Status, "got: %s", res.RawBody)
}
