package workos_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
)

func TestSharedNotificationManagementIsAdminOnlyByDefault(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	for _, role := range []string{"admin", "developer", "basic_member", "viewer", "unknown"} {
		for _, action := range []permissions.Action{permissions.Read, permissions.Write} {
			err := rbac.Check(rbac.U(urn.New().Workspace(workspaceID).Notifications(), action), workos.PermissionsForRoles(workspaceID, []string{role}))
			if role == "admin" {
				require.NoError(t, err)
			} else {
				require.Error(t, err, "role %s must not manage shared notifications", role)
			}
		}
	}
}
