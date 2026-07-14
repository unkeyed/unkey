package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_get_identity"
)

// TestGetIdentity_AuthorizesCanonicalURNPermission guarantees an exact identity
// URN can read a request addressed by external ID without a legacy tuple grant.
func TestGetIdentity_AuthorizesCanonicalURNPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	externalID := uid.New(uid.TestPrefix)
	identity := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: externalID})
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:identities/%s#read_identity", workspaceID, identity.ID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Identity: externalID})
	require.Equal(t, http.StatusOK, res.Status, "got: %s", res.RawBody)
}
