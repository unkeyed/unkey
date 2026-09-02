package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
)

// TestCreateRoleAuthorizesCanonicalURNPermissions guarantees write_role covers
// role creation and attachment while new permissions still require write_permission.
func TestCreateRoleAuthorizesCanonicalURNPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	projectID := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID}).ProjectID
	existingPermission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspaceID,
		Name:        "existing.canonical.permission",
		Slug:        "existing.canonical.permission",
	})

	tests := []struct {
		name        string
		permissions []string
		grants      []string
	}{
		{
			name:        "create role",
			permissions: []string{},
			grants: []string{
				fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/*#write", workspaceID, projectID),
			},
		},
		{
			name:        "attach existing permission",
			permissions: []string{existingPermission.Slug},
			grants: []string{
				fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/*#write", workspaceID, projectID),
			},
		},
		{
			name:        "create and attach permission",
			permissions: []string{"missing.canonical.permission"},
			grants: []string{
				fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/*#write", workspaceID, projectID),
				fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/permissions/*#write", workspaceID, projectID),
			},
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, test.grants...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}
			permissions := test.permissions

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Name:        fmt.Sprintf("canonical-role-%d", i),
				Permissions: &permissions,
			})

			require.Equal(t, http.StatusOK, res.Status, res.RawBody)
		})
	}
}
