package workos

import (
	"fmt"
	"slices"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	rbacpermissions "github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// permissionMapping pairs one WorkOS permission slug with the canonical Unkey
// permissions it grants.
type permissionMapping struct {
	// name is synced to WorkOS by tools/upsert-workos-permissions.
	name string
	// description is synced to WorkOS by tools/upsert-workos-permissions.
	description string
	permissions []permissionGrant
}

// permissionGrant pairs a canonical resource pattern with one action.
type permissionGrant struct {
	resource string
	action   rbac.ActionType
}

// action converts a typed action to the representation stored on a grant.
func action(value fmt.Stringer) rbac.ActionType {
	return rbac.ActionType(value.String())
}

// PermissionDefinition is the WorkOS-facing definition of one Unkey permission.
type PermissionDefinition struct {
	Slug        string
	Name        string
	Description string
}

// permissionMappings contains one WorkOS slug for every resource and action in
// the public permission catalog.
var permissionMappings = map[string]permissionMapping{
	"admin:*": {
		name:        "Admin",
		description: "Grants full administrative access.",
		permissions: []permissionGrant{
			{resource: "**", action: rbac.ActionType(rbacpermissions.Wildcard)},
		},
	},
	"github_apps:read": {
		name:        "Read GitHub apps",
		description: "Allows reading GitHub app installations.",
		permissions: []permissionGrant{
			{resource: "github/apps/*", action: action(rbacpermissions.ReadGitHubApp{})},
		},
	},
	"github_apps:write": {
		name:        "Write GitHub apps",
		description: "Allows creating and updating GitHub app installations.",
		permissions: []permissionGrant{
			{resource: "github/apps/*", action: action(rbacpermissions.WriteGitHubApp{})},
		},
	},
	"github_apps:delete": {
		name:        "Delete GitHub apps",
		description: "Allows deleting GitHub app installations.",
		permissions: []permissionGrant{
			{resource: "github/apps/*", action: action(rbacpermissions.DeleteGitHubApp{})},
		},
	},
	"deployments:read": {
		name:        "Read deployments",
		description: "Allows reading deployments.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.ReadDeployment{})},
		},
	},
	"deployments:delete": {
		name:        "Delete deployments",
		description: "Allows deleting deployments.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.DeleteDeployment{})},
		},
	},
	"deployments:write": {
		name:        "Write deployments",
		description: "Allows creating, updating, starting, and stopping deployments.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.WriteDeployment{})},
		},
	},
	"deployment_logs:read": {
		name:        "Read deployment logs",
		description: "Allows reading deployment logs.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*/logs", action: action(rbacpermissions.ReadDeploymentLogs{})},
		},
	},
	"projects:read": {
		name:        "Read projects",
		description: "Allows reading projects.",
		permissions: []permissionGrant{
			{resource: "projects/*", action: action(rbacpermissions.ReadProject{})},
		},
	},
	"projects:write": {
		name:        "Write projects",
		description: "Allows creating and updating projects.",
		permissions: []permissionGrant{
			{resource: "projects/*", action: action(rbacpermissions.WriteProject{})},
		},
	},
	"projects:delete": {
		name:        "Delete projects",
		description: "Allows deleting projects.",
		permissions: []permissionGrant{
			{resource: "projects/*", action: action(rbacpermissions.DeleteProject{})},
		},
	},
	"apps:read": {
		name:        "Read apps",
		description: "Allows reading apps in a project.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*", action: action(rbacpermissions.ReadApp{})},
		},
	},
	"apps:write": {
		name:        "Write apps",
		description: "Allows creating and updating apps in a project.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*", action: action(rbacpermissions.WriteApp{})},
		},
	},
	"apps:delete": {
		name:        "Delete apps",
		description: "Allows deleting apps.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*", action: action(rbacpermissions.DeleteApp{})},
		},
	},
	"environments:read": {
		name:        "Read environments",
		description: "Allows reading environment build and runtime settings.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.ReadEnvironment{})},
		},
	},
	"environments:write": {
		name:        "Write environments",
		description: "Allows updating environment settings and promoting or rolling back deployments.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.WriteEnvironment{})},
		},
	},
	"environments:delete": {
		name:        "Delete environments",
		description: "Allows deleting environments.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.DeleteEnvironment{})},
		},
	},
	"gateway_logs:read": {
		name:        "Read gateway logs",
		description: "Allows reading gateway logs.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/gateway/logs", action: action(rbacpermissions.ReadGatewayLogs{})},
		},
	},
	"gateway_policies:read": {
		name:        "Read gateway policies",
		description: "Allows reading an environment's gateway policies.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: action(rbacpermissions.ReadGatewayPolicy{})},
		},
	},
	"gateway_policies:write": {
		name:        "Write gateway policies",
		description: "Allows creating and updating an environment's gateway policies.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: action(rbacpermissions.WriteGatewayPolicy{})},
		},
	},
	"gateway_policies:delete": {
		name:        "Delete gateway policies",
		description: "Allows deleting an environment's gateway policies.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: action(rbacpermissions.DeleteGatewayPolicy{})},
		},
	},
	"environment_variables:read": {
		name:        "Read environment variables",
		description: "Allows reading environment variables, including recoverable values.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/variables/*", action: action(rbacpermissions.ReadEnvironmentVariable{})},
		},
	},
	"environment_variables:write": {
		name:        "Write environment variables",
		description: "Allows creating and overwriting environment variables.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/variables/*", action: action(rbacpermissions.WriteEnvironmentVariable{})},
		},
	},
	"environment_variables:delete": {
		name:        "Delete environment variables",
		description: "Allows removing environment variables.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/variables/*", action: action(rbacpermissions.DeleteEnvironmentVariable{})},
		},
	},
	"domains:write": {
		name:        "Write domains",
		description: "Allows attaching, updating, and restarting verification for custom domains.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/domains/*", action: action(rbacpermissions.WriteDomain{})},
		},
	},
	"domains:read": {
		name:        "Read domains",
		description: "Allows reading an environment's custom domains.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/domains/*", action: action(rbacpermissions.ReadDomain{})},
		},
	},
	"domains:delete": {
		name:        "Delete domains",
		description: "Allows removing a custom domain from an environment.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/domains/*", action: action(rbacpermissions.DeleteDomain{})},
		},
	},
	"identities:write": {
		name:        "Write identities",
		description: "Allows creating and updating identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.WriteIdentity{})},
		},
	},
	"identities:read": {
		name:        "Read identities",
		description: "Allows reading identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.ReadIdentity{})},
		},
	},
	"identities:delete": {
		name:        "Delete identities",
		description: "Allows deleting identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.DeleteIdentity{})},
		},
	},
	"keyspaces:read": {
		name:        "Read keyspaces",
		description: "Allows reading keyspaces.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.ReadKeyspace{})},
		},
	},
	"keyspaces:write": {
		name:        "Write keyspaces",
		description: "Allows creating and updating keyspaces.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.WriteKeyspace{})},
		},
	},
	"keyspaces:delete": {
		name:        "Delete keyspaces",
		description: "Allows deleting keyspaces.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.DeleteKeyspace{})},
		},
	},
	"keyspace_logs:read": {
		name:        "Read keyspace logs",
		description: "Allows reading keyspace logs.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*/logs", action: action(rbacpermissions.ReadKeyspaceLogs{})},
		},
	},
	"keys:write": {
		name:        "Write keys",
		description: "Allows creating, updating, and encrypting recoverable keys.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.WriteKey{})},
		},
	},
	"keys:read": {
		name:        "Read keys",
		description: "Allows reading keys.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.ReadKey{})},
		},
	},
	"keys:verify": {
		name:        "Verify keys",
		description: "Allows verifying keys.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.VerifyKey{})},
		},
	},
	"keys:decrypt": {
		name:        "Decrypt keys",
		description: "Allows reading recoverable key material.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.DecryptKey{})},
		},
	},
	"keys:delete": {
		name:        "Delete keys",
		description: "Allows deleting keys.",
		permissions: []permissionGrant{
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.DeleteKey{})},
		},
	},
	"ratelimit_namespaces:read": {
		name:        "Read rate limit namespaces",
		description: "Allows reading rate limit namespaces.",
		permissions: []permissionGrant{
			{resource: "projects/*/ratelimits/namespaces/*", action: action(rbacpermissions.ReadRatelimitNamespace{})},
		},
	},
	"ratelimit_namespaces:write": {
		name:        "Write rate limit namespaces",
		description: "Allows creating and updating rate limit namespaces.",
		permissions: []permissionGrant{
			{resource: "projects/*/ratelimits/namespaces/*", action: action(rbacpermissions.WriteRatelimitNamespace{})},
		},
	},
	"ratelimit_namespaces:delete": {
		name:        "Delete rate limit namespaces",
		description: "Allows deleting rate limit namespaces.",
		permissions: []permissionGrant{
			{resource: "projects/*/ratelimits/namespaces/*", action: action(rbacpermissions.DeleteRatelimitNamespace{})},
		},
	},
	"ratelimit_namespaces:limit": {
		name:        "Use rate limit namespaces",
		description: "Allows checking and using rate limit namespaces.",
		permissions: []permissionGrant{
			{resource: "projects/*/ratelimits/namespaces/*", action: action(rbacpermissions.LimitRatelimitNamespace{})},
		},
	},
	"ratelimit_logs:read": {
		name:        "Read rate limit logs",
		description: "Allows reading rate limit logs.",
		permissions: []permissionGrant{
			{resource: "projects/*/ratelimits/namespaces/*/logs", action: action(rbacpermissions.ReadRatelimitLogs{})},
		},
	},
	"ratelimit_overrides:read": {
		name:        "Read rate limit overrides",
		description: "Allows reading rate limit overrides.",
		permissions: []permissionGrant{
			{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: action(rbacpermissions.ReadRatelimitOverride{})},
		},
	},
	"ratelimit_overrides:write": {
		name:        "Write rate limit overrides",
		description: "Allows creating and updating rate limit overrides.",
		permissions: []permissionGrant{
			{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: action(rbacpermissions.WriteRatelimitOverride{})},
		},
	},
	"ratelimit_overrides:delete": {
		name:        "Delete rate limit overrides",
		description: "Allows deleting rate limit overrides.",
		permissions: []permissionGrant{
			{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: action(rbacpermissions.DeleteRatelimitOverride{})},
		},
	},
	"roles:read": {
		name:        "Read roles",
		description: "Allows reading roles.",
		permissions: []permissionGrant{
			{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.ReadRole{})},
		},
	},
	"roles:write": {
		name:        "Write roles",
		description: "Allows creating and updating roles and their permission assignments.",
		permissions: []permissionGrant{
			{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.WriteRole{})},
		},
	},
	"roles:delete": {
		name:        "Delete roles",
		description: "Allows deleting roles.",
		permissions: []permissionGrant{
			{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.DeleteRole{})},
		},
	},
	"permissions:read": {
		name:        "Read permissions",
		description: "Allows reading permission definitions.",
		permissions: []permissionGrant{
			{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.ReadPermission{})},
		},
	},
	"permissions:write": {
		name:        "Write permissions",
		description: "Allows creating and updating permission definitions.",
		permissions: []permissionGrant{
			{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.WritePermission{})},
		},
	},
	"permissions:delete": {
		name:        "Delete permissions",
		description: "Allows deleting permission definitions.",
		permissions: []permissionGrant{
			{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.DeletePermission{})},
		},
	},
}

// PermissionDefinitions returns every WorkOS permission definition understood
// by Unkey.
func PermissionDefinitions() []PermissionDefinition {
	slugs := sortedPermissionSlugs()
	definitions := make([]PermissionDefinition, 0, len(slugs))
	for _, slug := range slugs {
		mapping := permissionMappings[slug]
		definitions = append(definitions, PermissionDefinition{
			Slug:        slug,
			Name:        mapping.name,
			Description: mapping.description,
		})
	}
	return definitions
}

func sortedPermissionSlugs() []string {
	slugs := make([]string, 0, len(permissionMappings))
	for slug := range permissionMappings {
		slugs = append(slugs, slug)
	}
	slices.Sort(slugs)
	return slugs
}

// translatePermissions translates WorkOS permission strings into canonical
// Unkey resource permissions. Unknown permissions are ignored but logged.
//
// For workspaceID "ws_1":
//
//	keys:write         => unkey:v1:ws_1:projects/*/keyspaces/*/keys/*#write_key
//	keys:read          => unkey:v1:ws_1:projects/*/keyspaces/*/keys/*#read_key
//	identities:read    => unkey:v1:ws_1:projects/*/identities/*#read_identity
//	admin:*            => unkey:v1:ws_1:**#*
//	unknown:permission => dropped with a warning log
func translatePermissions(workspaceID string, permissions []string) []string {
	var out []string

	for _, permission := range permissions {
		mapping, ok := permissionMappings[permission]
		if !ok {
			logger.Warn("unable to translate permission from workos to unkey, skipping ...",
				"permission", permission,
			)
			continue
		}

		for _, grant := range mapping.permissions {
			out = append(out, rbac.UnkeyPermission{
				Resource: urn.V1{
					WorkspaceID: workspaceID,
					Resource:    grant.resource,
				},
				Action: grant.action,
			}.String())
		}
	}

	return out
}
