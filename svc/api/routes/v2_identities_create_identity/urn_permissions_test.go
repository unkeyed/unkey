package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_create_identity"
)

// TestCreateIdentity_AuthorizesCanonicalURNPermission guarantees project-scoped
// URNs can create identities without a legacy tuple grant.
func TestCreateIdentity_AuthorizesCanonicalURNPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:projects/*#create_identity", workspaceID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ExternalId: uid.New(uid.TestPrefix)})
	require.Equal(t, http.StatusOK, res.Status, "got: %s", res.RawBody)
}
