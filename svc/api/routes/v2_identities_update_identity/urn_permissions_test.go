package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_update_identity"
)

// TestUpdateIdentity_AuthorizesCanonicalURNPermission guarantees a project-scoped
// URN can update an identity without a legacy tuple grant.
func TestUpdateIdentity_AuthorizesCanonicalURNPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	externalID := uid.New(uid.TestPrefix)
	h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: externalID})
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:projects/*/identities/*#update_identity", workspaceID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	meta := map[string]any{"updated": true}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Identity: externalID, Meta: &meta})
	require.Equal(t, http.StatusOK, res.Status, "got: %s", res.RawBody)
}
