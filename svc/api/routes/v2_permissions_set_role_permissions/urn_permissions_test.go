package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_set_role_permissions"
)

// TestSetRolePermissionsAuthorizesCanonicalWriteRole guarantees write_role can
// replace a role's existing permission attachments.
func TestSetRolePermissionsAuthorizesCanonicalWriteRole(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspaceID, Name: "canonical-role"})
	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspaceID,
		Name:        "canonical.role.permission",
		Slug:        "canonical.role.permission",
	})
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/%s#write", workspaceID, role.ProjectID, role.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Role:        ptr.P(role.ID),
		Permissions: []string{permission.Slug},
	})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
}
