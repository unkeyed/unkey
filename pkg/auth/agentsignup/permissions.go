package agentsignup

import (
	"github.com/unkeyed/unkey/pkg/rbac"
	rbacpermissions "github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// Role is the only JWT role this provider expands. admin and every other
// role grant nothing, so a stolen signing key cannot mint a full-admin token
// through this auth entry.
const Role = "agent_signup"

// Issuer is the iss claim the dashboard uses for agent-signup root-key JWTs.
// It is distinct from the local dashboard proxy issuer so that entry cannot
// verify these tokens as workspace admins.
const Issuer = "https://app.unkey.com/agent-signup"

// allowlist is the key-management set the dashboard agent signup path may
// attach to a root key. decrypt and root-key permissions are absent on purpose.
var allowlist = []struct {
	resource string
	action   rbac.ActionType
}{
	{resource: "projects/*/keyspaces/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/keyspaces/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/keyspaces/*/logs", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Verify)},
}

// permissionsForRoles expands role slugs into API permissions.
// Only [Role] contributes permissions. Unknown roles, including admin, are ignored.
func permissionsForRoles(workspaceID string, roles []string) []string {
	for _, role := range roles {
		if role != Role {
			continue
		}
		permissions := make([]string, 0, len(allowlist))
		for _, grant := range allowlist {
			permissions = append(permissions, rbac.UnkeyPermission{
				Resource: urn.V1{
					WorkspaceID: workspaceID,
					Resource:    grant.resource,
				},
				Action: grant.action,
			}.String())
		}
		return permissions
	}
	return nil
}

// PermissionAllowed reports whether permission is an exact member of the
// agent-signup allowlist for workspaceID. Patterns such as ** and concrete
// resource ids are rejected. The check does not consult the caller's grants.
func PermissionAllowed(workspaceID, permission string) bool {
	if workspaceID == "" || permission == "" {
		return false
	}
	for _, allowed := range permissionsForRoles(workspaceID, []string{Role}) {
		if allowed == permission {
			return true
		}
	}
	return false
}
