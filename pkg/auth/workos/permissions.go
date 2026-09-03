package workos

import (
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	rbacpermissions "github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// resourcePermission identifies one action on a resource.
type resourcePermission struct {
	resource string
	action   rbac.ActionType
}

// developerPermissions includes product writes and API key decryption, but no
// workspace root-key operations.
var developerPermissions = []resourcePermission{
	{resource: "github/apps/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "github/apps/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "github/apps/*", action: rbac.ActionType(rbacpermissions.Delete)},

	{resource: "projects/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/portals/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/portals/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/portals/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/portals/*/sessions/*", action: rbac.ActionType(rbacpermissions.Write)},

	{resource: "projects/*/apps/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/apps/*", action: rbac.ActionType(rbacpermissions.Delete)},

	{resource: "projects/*/apps/*/environments/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/apps/*/environments/*", action: rbac.ActionType(rbacpermissions.Delete)},

	{resource: "projects/*/apps/*/environments/*/deployments/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/deployments/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/apps/*/environments/*/deployments/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/apps/*/environments/*/deployments/*/logs", action: rbac.ActionType(rbacpermissions.Read)},

	{resource: "projects/*/apps/*/environments/*/domains/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/domains/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/apps/*/environments/*/domains/*", action: rbac.ActionType(rbacpermissions.Delete)},

	{resource: "projects/*/apps/*/environments/*/variables/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/variables/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/apps/*/environments/*/variables/*", action: rbac.ActionType(rbacpermissions.Delete)},

	{resource: "projects/*/apps/*/environments/*/gateway/logs", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: rbac.ActionType(rbacpermissions.Delete)},

	{resource: "projects/*/identities/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/identities/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/identities/*", action: rbac.ActionType(rbacpermissions.Delete)},

	{resource: "projects/*/keyspaces/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/keyspaces/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/keyspaces/*/logs", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Decrypt)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Verify)},

	{resource: "projects/*/ratelimits/namespaces/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/ratelimits/namespaces/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/ratelimits/namespaces/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/ratelimits/namespaces/*", action: rbac.ActionType(rbacpermissions.Limit)},
	{resource: "projects/*/ratelimits/namespaces/*/logs", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: rbac.ActionType(rbacpermissions.Delete)},

	{resource: "projects/*/rbac/roles/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/rbac/roles/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/rbac/roles/*", action: rbac.ActionType(rbacpermissions.Delete)},
	{resource: "projects/*/rbac/permissions/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/rbac/permissions/*", action: rbac.ActionType(rbacpermissions.Write)},
	{resource: "projects/*/rbac/permissions/*", action: rbac.ActionType(rbacpermissions.Delete)},
}

// viewerPermissions includes product reads but excludes API key decryption.
var viewerPermissions = []resourcePermission{
	{resource: "github/apps/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/portals/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/deployments/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/deployments/*/logs", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/domains/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/variables/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/gateway/logs", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/identities/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*/logs", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/keyspaces/*/keys/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/ratelimits/namespaces/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/ratelimits/namespaces/*/logs", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/rbac/roles/*", action: rbac.ActionType(rbacpermissions.Read)},
	{resource: "projects/*/rbac/permissions/*", action: rbac.ActionType(rbacpermissions.Read)},
}

// rolePolicies is the API-owned authorization policy for organization roles.
var rolePolicies = map[string][]resourcePermission{
	"admin": {
		{resource: "**", action: rbac.ActionType(rbacpermissions.Wildcard)},
	},
	"developer": developerPermissions,
	// Keep basic_member valid while existing WorkOS memberships migrate to developer.
	"basic_member": developerPermissions,
	"viewer":       viewerPermissions,
}

// permissionsForRoles expands role slugs into API-owned permissions.
// Unknown roles are ignored and logged so they cannot add permissions by accident.
func permissionsForRoles(workspaceID string, roles []string) []string {
	var permissions []string
	seen := make(map[string]struct{})

	for _, role := range roles {
		rolePermissions, ok := rolePolicies[role]
		if !ok {
			logger.Warn("unable to resolve role, skipping",
				"role", role,
			)
			continue
		}

		for _, rolePermission := range rolePermissions {
			permission := rbac.UnkeyPermission{
				Resource: urn.V1{
					WorkspaceID: workspaceID,
					Resource:    rolePermission.resource,
				},
				Action: rolePermission.action,
			}.String()
			if _, ok := seen[permission]; ok {
				continue
			}
			seen[permission] = struct{}{}
			permissions = append(permissions, permission)
		}
	}

	return permissions
}
