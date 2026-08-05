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

// permissionMappings contains the canonical catalog that is frontloaded into
// WorkOS before route migration. Existing slugs and their legacy resource
// grants remain during the additive rollout so old and migrated routes can run
// concurrently.
var permissionMappings = map[string]permissionMapping{
	"admin:*": {
		name:        "Admin",
		description: "Grants full administrative access.",
		permissions: []permissionGrant{
			{resource: "**", action: "*"},
		},
	},
	"identities:create": {
		name:        "Create identities",
		description: "Allows creating identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.CreateIdentity{})},
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
	"identities:delete": {
		name:        "Delete identities",
		description: "Allows deleting identities.",
		permissions: []permissionGrant{
			{resource: "projects/*/identities/*", action: action(rbacpermissions.DeleteIdentity{})},
		},
	},
	"keys:create": {
		name:        "Create keys",
		description: "Allows creating keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*", action: action(rbacpermissions.CreateKey{})},
			{resource: "projects/*/keyspaces/*/keys/*", action: action(rbacpermissions.CreateKey{})},
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
	"keys:encrypt": {
		name:        "Encrypt keys",
		description: "Allows creating recoverable keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.EncryptKey{})},
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
	"projects:create": {
		name:        "Create Projects",
		description: "Allows creating projects.",
		permissions: []permissionGrant{{resource: "projects/*", action: action(rbacpermissions.CreateProject{})}},
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
	"projects:delete": {
		name:        "Delete Projects",
		description: "Allows deleting projects.",
		permissions: []permissionGrant{{resource: "projects/*", action: action(rbacpermissions.DeleteProject{})}},
	},
	"apps:create": {
		name:        "Create Apps",
		description: "Allows creating apps.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*", action: action(rbacpermissions.CreateApp{})}},
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
	"apps:delete": {
		name:        "Delete Apps",
		description: "Allows deleting apps.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*", action: action(rbacpermissions.DeleteApp{})}},
	},
	"environments:read": {
		name:        "Read Environments",
		description: "Allows reading environments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.ReadEnvironment{})}},
	},
	"environments:update": {
		name:        "Update Environments",
		description: "Allows updating environment settings.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.UpdateEnvironment{})}},
	},
	"environments:read_variables": {
		name:        "Read Environment Variables",
		description: "Allows reading environment variables.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.ReadVariables{})}},
	},
	"environments:set_variables": {
		name:        "Set Environment Variables",
		description: "Allows setting environment variables.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.SetVariables{})}},
	},
	"environments:remove_variables": {
		name:        "Remove Environment Variables",
		description: "Allows removing environment variables.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.RemoveVariables{})}},
	},
	"deployments:create": {
		name:        "Create Deployments",
		description: "Allows creating deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.CreateDeployment{})}},
	},
	"deployments:read": {
		name:        "Read Deployments",
		description: "Allows reading deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.ReadDeployment{})}},
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
	"deployments:promote": {
		name:        "Promote Deployments",
		description: "Allows promoting deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.PromoteDeployment{})}},
	},
	"deployments:rollback": {
		name:        "Rollback Deployments",
		description: "Allows rolling back deployments.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.RollbackDeployment{})}},
	},
	"keyspaces:create": {
		name:        "Create Keyspaces",
		description: "Allows creating keyspaces.",
		permissions: []permissionGrant{{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.CreateKeyspace{})}},
	},
	"keyspaces:read": {
		name:        "Read Keyspaces",
		description: "Allows reading keyspaces.",
		permissions: []permissionGrant{{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.ReadKeyspace{})}},
	},
	"keyspaces:delete": {
		name:        "Delete Keyspaces",
		description: "Allows deleting keyspaces.",
		permissions: []permissionGrant{{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.DeleteKeyspace{})}},
	},
	"keyspaces:read_logs": {
		name:        "Read Keyspace Logs",
		description: "Allows reading key verification logs.",
		permissions: []permissionGrant{{resource: "projects/*/keyspaces/*", action: action(rbacpermissions.ReadKeyspaceLogs{})}},
	},
	"ratelimits:create_namespace": {
		name:        "Create Rate Limit Namespaces",
		description: "Allows creating rate limit namespaces.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*", action: action(rbacpermissions.CreateNamespace{})}},
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
	"ratelimits:set_override": {
		name:        "Set Rate Limit Overrides",
		description: "Allows setting rate limit overrides.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: action(rbacpermissions.SetOverride{})}},
	},
	"ratelimits:delete_override": {
		name:        "Delete Rate Limit Overrides",
		description: "Allows deleting rate limit overrides.",
		permissions: []permissionGrant{{resource: "projects/*/ratelimits/namespaces/*/overrides/*", action: action(rbacpermissions.DeleteOverride{})}},
	},
	"roles:create": {
		name:        "Create Roles",
		description: "Allows creating roles.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.CreateRole{})}},
	},
	"roles:read": {
		name:        "Read Roles",
		description: "Allows reading roles.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.ReadRole{})}},
	},
	"roles:delete": {
		name:        "Delete Roles",
		description: "Allows deleting roles.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/roles/*", action: action(rbacpermissions.DeleteRole{})}},
	},
	"permissions:create": {
		name:        "Create Permission Definitions",
		description: "Allows creating permission definitions.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.CreatePermission{})}},
	},
	"permissions:read": {
		name:        "Read Permission Definitions",
		description: "Allows reading permission definitions.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.ReadPermission{})}},
	},
	"permissions:delete": {
		name:        "Delete Permission Definitions",
		description: "Allows deleting permission definitions.",
		permissions: []permissionGrant{{resource: "projects/*/rbac/permissions/*", action: action(rbacpermissions.DeletePermission{})}},
	},
	"gateway:read_policies": {
		name:        "Read Gateway Policies",
		description: "Allows reading gateway policies.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: action(rbacpermissions.ReadPolicy{})}},
	},
	"gateway:set_policies": {
		name:        "Set Gateway Policies",
		description: "Allows replacing an environment's gateway policies.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/gateway", action: action(rbacpermissions.SetPolicies{})}},
	},
	"gateway:update_policy": {
		name:        "Update Gateway Policies",
		description: "Allows updating individual gateway policies.",
		permissions: []permissionGrant{{resource: "projects/*/apps/*/environments/*/gateway/policies/*", action: action(rbacpermissions.UpdatePolicy{})}},
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
//	keys:create        => unkey:v1:ws_1:projects/*/keyspaces/*/keys/*#create_key
//	keys:read          => unkey:v1:ws_1:projects/*/keyspaces/*/keys/*#read_key
//	keys:update        => unkey:v1:ws_1:projects/*/keyspaces/*/keys/*#update_key
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
