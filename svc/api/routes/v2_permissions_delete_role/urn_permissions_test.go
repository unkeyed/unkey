package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_role"
)

// TestDeleteRoleAuthorizesCanonicalDeleteRole guarantees an exact
// project-scoped role grant can delete its role.
func TestDeleteRoleAuthorizesCanonicalDeleteRole(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspaceID, Name: "canonical-role"})
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/%s#delete", workspaceID, role.ProjectID, role.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Role: role.ID})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	_, err := db.Query.FindRoleByID(t.Context(), h.DB.RO(), role.ID)
	require.True(t, db.IsNotFound(err))
}
