package workos

import (
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

type permissionGrant struct {
	resource string
	action   rbac.ActionType
}

func action(value interface{ String() string }) rbac.ActionType {
	return rbac.ActionType(value.String())
}

// PermissionDefinition is the WorkOS-facing definition of one Unkey permission.
type PermissionDefinition struct {
	Slug        string
	Name        string
	Description string
}

// permissionMappings contains WorkOS permissions and their Unkey grants. New
// permissions are added before routes use them, and old grants stay while
// routes migrate.
var permissionMappings = map[string]permissionMapping{
	"admin:*": {
		name:        "Admin",
		description: "Grants full administrative access.",
		permissions: []permissionGrant{
			{resource: "**", action: "*"},
		},
	},
	"apps:create": {
		name:        "Create Apps",
		description: "Allows creating apps.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*", action: action(rbacpermissions.CreateApp{})}},
	},
	"apps:delete": {
		name:        "Delete Apps",
		description: "Allows deleting apps.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*", action: action(rbacpermissions.DeleteApp{})}},
	},
	"apps:read": {
		name:        "Read Apps",
		description: "Allows reading apps.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*", action: action(rbacpermissions.ReadApp{})}},
	},
	"apps:update": {
		name:        "Update Apps",
		description: "Allows updating apps.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*", action: action(rbacpermissions.UpdateApp{})}},
	},
	"deployments:create": {
		name:        "Create Deployments",
		description: "Allows creating deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.CreateDeployment{})}},
	},
	"deployments:promote": {
		name:        "Promote Deployments",
		description: "Allows promoting deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.PromoteDeployment{})}},
	},
	"deployments:read": {
		name:        "Read Deployments",
		description: "Allows reading deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.ReadDeployment{})}},
	},
	"deployments:rollback": {
		name:        "Rollback Deployments",
		description: "Allows rolling back deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.RollbackDeployment{})}},
	},
	"deployments:start": {
		name:        "Start Deployments",
		description: "Allows starting deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.StartDeployment{})}},
	},
	"deployments:stop": {
		name:        "Stop Deployments",
		description: "Allows stopping deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.StopDeployment{})}},
	},
	"environments:create_variables": {
		name:        "Create Environment Variables",
		description: "Allows creating or replacing environment variables.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.CreateVariables{})}},
	},
	"environments:delete_variables": {
		name:        "Delete Environment Variables",
		description: "Allows deleting environment variables.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.DeleteVariables{})}},
	},
	"environments:read": {
		name:        "Read Environments",
		description: "Allows reading environments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.ReadEnvironment{})}},
	},
	"environments:read_variables": {
		name:        "Read Environment Variables",
		description: "Allows reading environment variables.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.ReadVariables{})}},
	},
	"environments:update": {
		name:        "Update Environments",
		description: "Allows updating environment settings.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.UpdateEnvironment{})}},
	},
	"gateway:create_policies": {
		name:        "Create Gateway Policies",
		description: "Allows creating or replacing gateway policies.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/gateway", action: action(rbacpermissions.CreatePolicies{})}},
	},
	"gateway:read_policies": {
		name:        "Read Gateway Policies",
		description: "Allows reading gateway policies.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: action(rbacpermissions.ReadPolicy{})}},
	},
	"gateway:update_policy": {
		name:        "Update Gateway Policies",
		description: "Allows updating individual gateway policies.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: action(rbacpermissions.UpdatePolicy{})}},
	},
	"identities:create": {
		name:        "Create identities",
		description: "Allows creating identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.CreateIdentity{})},
		},
	},
	"identities:delete": {
		name:        "Delete identities",
		description: "Allows deleting identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.DeleteIdentity{})},
		},
	},
	"identities:read": {
		name:        "Read identities",
		description: "Allows reading identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.ReadIdentity{})},
		},
	},
	"identities:update": {
		name:        "Update identities",
		description: "Allows updating identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.UpdateIdentity{})},
		},
	},
	"keys:create": {
		name:        "Create keys",
		description: "Allows creating keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*", action: action(rbacpermissions.CreateKey{})},
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.CreateKey{})},
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.CreateKey{})},
		},
	},
	"keys:decrypt": {
		name:        "Decrypt keys",
		description: "Allows reading recoverable key material.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.DecryptKey{})},
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.DecryptKey{})},
		},
	},
	"keys:delete": {
		name:        "Delete keys",
		description: "Allows deleting keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.DeleteKey{})},
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.DeleteKey{})},
		},
	},
	"keys:encrypt": {
		name:        "Encrypt keys",
		description: "Allows creating recoverable keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.EncryptKey{})},
		},
	},
	"keys:read": {
		name:        "Read keys",
		description: "Allows reading keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.ReadKey{})},
			{resource: "keyspaces/*", action: action(rbacpermissions.ReadKeyspace{})},
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.ReadKey{})},
			{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.ReadKeyspace{})},
		},
	},
	"keys:update": {
		name:        "Update keys",
		description: "Allows updating keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.UpdateKey{})},
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.UpdateKey{})},
		},
	},
	"keys:verify": {
		name:        "Verify keys",
		description: "Allows verifying keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.VerifyKey{})},
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.VerifyKey{})},
		},
	},
	"keyspaces:create": {
		name:        "Create Keyspaces",
		description: "Allows creating keyspaces.",
		permissions: []permissionGrant{{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.CreateKeyspace{})}},
	},
	"keyspaces:delete": {
		name:        "Delete Keyspaces",
		description: "Allows deleting keyspaces.",
		permissions: []permissionGrant{{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.DeleteKeyspace{})}},
	},
	"keyspaces:read": {
		name:        "Read Keyspaces",
		description: "Allows reading keyspaces.",
		permissions: []permissionGrant{{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.ReadKeyspace{})}},
	},
	"keyspaces:read_logs": {
		name:        "Read Keyspace Logs",
		description: "Allows reading key verification logs.",
		permissions: []permissionGrant{{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.ReadKeyspaceLogs{})}},
	},
	"permissions:create": {
		name:        "Create Permission Definitions",
		description: "Allows creating permission definitions.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.CreatePermission{})}},
	},
	"permissions:delete": {
		name:        "Delete Permission Definitions",
		description: "Allows deleting permission definitions.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.DeletePermission{})}},
	},
	"permissions:read": {
		name:        "Read Permission Definitions",
		description: "Allows reading permission definitions.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.ReadPermission{})}},
	},
	"projects:create": {
		name:        "Create Projects",
		description: "Allows creating projects.",
		permissions: []permissionGrant{{resource: "projects/*", action: action(rbacpermissions.CreateProject{})}},
	},
	"projects:delete": {
		name:        "Delete Projects",
		description: "Allows deleting projects.",
		permissions: []permissionGrant{{resource: "projects/*", action: action(rbacpermissions.DeleteProject{})}},
	},
	"projects:read": {
		name:        "Read Projects",
		description: "Allows reading projects.",
		permissions: []permissionGrant{{resource: "projects/*", action: action(rbacpermissions.ReadProject{})}},
	},
	"projects:update": {
		name:        "Update Projects",
		description: "Allows updating projects.",
		permissions: []permissionGrant{{resource: "projects/*", action: action(rbacpermissions.UpdateProject{})}},
	},
	"ratelimits:create_namespace": {
		name:        "Create Rate Limit Namespaces",
		description: "Allows creating rate limit namespaces.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*", action: action(rbacpermissions.CreateNamespace{})}},
	},
	"ratelimits:create_override": {
		name:        "Create Rate Limit Overrides",
		description: "Allows creating or replacing rate limit overrides.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: action(rbacpermissions.CreateOverride{})}},
	},
	"ratelimits:delete_override": {
		name:        "Delete Rate Limit Overrides",
		description: "Allows deleting rate limit overrides.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: action(rbacpermissions.DeleteOverride{})}},
	},
	"ratelimits:limit": {
		name:        "Use Rate Limits",
		description: "Allows consuming rate limits.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*", action: action(rbacpermissions.Limit{})}},
	},
	"ratelimits:read_logs": {
		name:        "Read Rate Limit Logs",
		description: "Allows reading rate limit logs.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*", action: action(rbacpermissions.ReadRatelimitLogs{})}},
	},
	"ratelimits:read_overrides": {
		name:        "Read Rate Limit Overrides",
		description: "Allows reading rate limit overrides.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: action(rbacpermissions.ReadOverride{})}},
	},
	"roles:create": {
		name:        "Create Roles",
		description: "Allows creating roles.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.CreateRole{})}},
	},
	"roles:delete": {
		name:        "Delete Roles",
		description: "Allows deleting roles.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.DeleteRole{})}},
	},
	"roles:read": {
		name:        "Read Roles",
		description: "Allows reading roles.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.ReadRole{})}},
	},
	"workspaces:install_github": {
		name:        "Install GitHub App",
		description: "Allows installing the Unkey GitHub App for a workspace.",
		permissions: []permissionGrant{{resource: "workspace", action: action(rbacpermissions.InstallGithub{})}},
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
// A WorkOS permission may add several grants while old and new resource scopes
// coexist during migration. For workspaceID "ws_1", "keys:create" adds:
//
//	unkey:v1:ws_1:keyspaces/*/keys/*#create_key
//	unkey:v1:ws_1:projects/*/keyspaces/*/keys/*#create_key
//
// "admin:*" adds "unkey:v1:ws_1:**#*". Unknown permissions are dropped with a
// warning. The complete translation table is permissionMappings.
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

		for _, permission := range mapping.permissions {
			out = append(out, rbac.UnkeyPermission{
				Resource: urn.V1{
					WorkspaceID: workspaceID,
					Resource:    permission.resource,
				},
				Action: permission.action,
			}.String())
		}
	}

	return out
}
