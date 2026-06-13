package workos

import (
	"slices"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/urn"
)

// permissionMapping pairs one WorkOS permission slug with the canonical Unkey
// permissions it grants.
type permissionMapping struct {
	// used to update the permission in workos via /tools/upsert-workos-permissions
	name string
	// used to update the permission in workos via /tools/upsert-workos-permissions
	description string
	permissions []permissionGrant
}

type permissionGrant struct {
	resource string
	action   rbac.ActionType
}

// PermissionDefinition is the WorkOS-facing definition of one Unkey permission.
type PermissionDefinition struct {
	Slug        string
	Name        string
	Description string
}

// Some of these are commented out, because I don't have the mental foresight to ensure they all are perfect as they are.
// Thus it's probably better to just "unlock" them when we need them and be sure that what we unlock is exactly what we need or tweak them.
var permissionMappings = map[string]permissionMapping{
	"admin:*": {
		name:        "Admin",
		description: "Grants full administrative access.",
		permissions: []permissionGrant{
			{resource: "**", action: "*"},
		},
	},
	//"keyspaces:create": {
	//	name:        "Create keyspaces",
	//	description: "Allows creating keyspaces.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*", action: "create_keyspace"},
	//	},
	//},
	//"keyspaces:read": {
	//	name:        "Read keyspaces",
	//	description: "Allows reading keyspaces.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*", action: "read_keyspace"},
	//	},
	//},
	//"keyspaces:update": {
	//	name:        "Update keyspaces",
	//	description: "Allows updating keyspaces.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*", action: "update_keyspace"},
	//	},
	//},
	//"keyspaces:delete": {
	//	name:        "Delete keyspaces",
	//	description: "Allows deleting keyspaces.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*", action: "delete_keyspace"},
	//	},
	//},
	//"keyspaces:read_analytics": {
	//	name:        "Read keyspace analytics",
	//	description: "Allows reading keyspace analytics.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*", action: "read_analytics"},
	//	},
	//},
	"keys:create": {
		name:        "Create keys",
		description: "Allows creating keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: "create_key"},
		},
	},
	"keys:encrypt": {
		name:        "Encrypt keys",
		description: "Allows creating recoverable keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: "encrypt_key"},
		},
	},
	"keys:decrypt": {
		name:        "Decrypt keys",
		description: "Allows reading recoverable key material.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: "decrypt_key"},
		},
	},
	//"keys:update": {
	//	name:        "Update keys",
	//	description: "Allows updating keys.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*/keys/*", action: "update_key"},
	//	},
	//},
	//"keys:delete": {
	//	name:        "Delete keys",
	//	description: "Allows deleting keys.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*/keys/*", action: "delete_key"},
	//	},
	//},
	//"keys:read": {
	//	name:        "Read keys",
	//	description: "Allows reading keys.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*/keys/*", action: "read_key"},
	//	},
	//},
	//"keys:verify": {
	//	name:        "Verify keys",
	//	description: "Allows verifying keys.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*/keys/*", action: "verify_key"},
	//	},
	//},
	//"identities:create": {
	//	name:        "Create identities",
	//	description: "Allows creating identities.",
	//	permissions: []permissionGrant{
	//		{resource: "identities/*", action: "create_identity"},
	//	},
	//},
	//"identities:read": {
	//	name:        "Read identities",
	//	description: "Allows reading identities.",
	//	permissions: []permissionGrant{
	//		{resource: "identities/*", action: "read_identity"},
	//	},
	//},
	//"identities:update": {
	//	name:        "Update identities",
	//	description: "Allows updating identities.",
	//	permissions: []permissionGrant{
	//		{resource: "identities/*", action: "update_identity"},
	//	},
	//},
	//"identities:delete": {
	//	name:        "Delete identities",
	//	description: "Allows deleting identities.",
	//	permissions: []permissionGrant{
	//		{resource: "identities/*", action: "delete_identity"},
	//	},
	//},
	//"ratelimit_namespaces:create": {
	//	name:        "Create rate limit namespaces",
	//	description: "Allows creating rate limit namespaces.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*", action: "create_namespace"},
	//	},
	//},
	//"ratelimit_namespaces:read": {
	//	name:        "Read rate limit namespaces",
	//	description: "Allows reading rate limit namespaces.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*", action: "read_namespace"},
	//	},
	//},
	//"ratelimit_namespaces:update": {
	//	name:        "Update rate limit namespaces",
	//	description: "Allows updating rate limit namespaces.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*", action: "update_namespace"},
	//	},
	//},
	//"ratelimit_namespaces:delete": {
	//	name:        "Delete rate limit namespaces",
	//	description: "Allows deleting rate limit namespaces.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*", action: "delete_namespace"},
	//	},
	//},
	//"ratelimit_namespaces:limit": {
	//	name:        "Limit rate limit namespaces",
	//	description: "Allows consuming rate limits.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*", action: "limit"},
	//	},
	//},
	//"ratelimit_overrides:set": {
	//	name:        "Set rate limit overrides",
	//	description: "Allows setting rate limit overrides.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*/overrides/*", action: "set_override"},
	//	},
	//},
	//"ratelimit_overrides:read": {
	//	name:        "Read rate limit overrides",
	//	description: "Allows reading rate limit overrides.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*/overrides/*", action: "read_override"},
	//	},
	//},
	//"ratelimit_overrides:delete": {
	//	name:        "Delete rate limit overrides",
	//	description: "Allows deleting rate limit overrides.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*/overrides/*", action: "delete_override"},
	//	},
	//},
	//"ratelimit_overrides:list": {
	//	name:        "List rate limit overrides",
	//	description: "Allows listing rate limit overrides.",
	//	permissions: []permissionGrant{
	//		{resource: "ratelimits/namespaces/*/overrides/*", action: "list_overrides"},
	//	},
	//},
	//"permissions:create": {
	//	name:        "Create permissions",
	//	description: "Allows creating permissions.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/permissions/*", action: "create_permission"},
	//	},
	//},
	//"permissions:update": {
	//	name:        "Update permissions",
	//	description: "Allows updating permissions.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/permissions/*", action: "update_permission"},
	//	},
	//},
	//"permissions:delete": {
	//	name:        "Delete permissions",
	//	description: "Allows deleting permissions.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/permissions/*", action: "delete_permission"},
	//	},
	//},
	//"permissions:read": {
	//	name:        "Read permissions",
	//	description: "Allows reading permissions.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/permissions/*", action: "read_permission"},
	//	},
	//},
	//"roles:create": {
	//	name:        "Create roles",
	//	description: "Allows creating roles.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/roles/*", action: "create_role"},
	//	},
	//},
	//"roles:update": {
	//	name:        "Update roles",
	//	description: "Allows updating roles.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/roles/*", action: "update_role"},
	//	},
	//},
	//"roles:delete": {
	//	name:        "Delete roles",
	//	description: "Allows deleting roles.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/roles/*", action: "delete_role"},
	//	},
	//},
	//"roles:read": {
	//	name:        "Read roles",
	//	description: "Allows reading roles.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/roles/*", action: "read_role"},
	//	},
	//},
	//"keys:add_permission": {
	//	name:        "Add permissions to keys",
	//	description: "Allows adding permissions to keys.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*/keys/*", action: "add_permission_to_key"},
	//	},
	//},
	//"keys:remove_permission": {
	//	name:        "Remove permissions from keys",
	//	description: "Allows removing permissions from keys.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*/keys/*", action: "remove_permission_from_key"},
	//	},
	//},
	//"keys:add_role": {
	//	name:        "Add roles to keys",
	//	description: "Allows adding roles to keys.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*/keys/*", action: "add_role_to_key"},
	//	},
	//},
	//"keys:remove_role": {
	//	name:        "Remove roles from keys",
	//	description: "Allows removing roles from keys.",
	//	permissions: []permissionGrant{
	//		{resource: "keyspaces/*/keys/*", action: "remove_role_from_key"},
	//	},
	//},
	//"roles:add_permission": {
	//	name:        "Add permissions to roles",
	//	description: "Allows adding permissions to roles.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/roles/*", action: "add_permission_to_role"},
	//	},
	//},
	//"roles:remove_permission": {
	//	name:        "Remove permissions from roles",
	//	description: "Allows removing permissions from roles.",
	//	permissions: []permissionGrant{
	//		{resource: "rbac/roles/*", action: "remove_permission_from_role"},
	//	},
	//},
	//"projects:create": {
	//	name:        "Create projects",
	//	description: "Allows creating projects.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*", action: "create_project"},
	//	},
	//},
	//"projects:read": {
	//	name:        "Read projects",
	//	description: "Allows reading projects.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*", action: "read_project"},
	//	},
	//},
	//"projects:update": {
	//	name:        "Update projects",
	//	description: "Allows updating projects.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*", action: "update_project"},
	//	},
	//},
	//"projects:delete": {
	//	name:        "Delete projects",
	//	description: "Allows deleting projects.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*", action: "delete_project"},
	//	},
	//},
	//"apps:create": {
	//	name:        "Create apps",
	//	description: "Allows creating apps.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*", action: "create_app"},
	//	},
	//},
	//"apps:read": {
	//	name:        "Read apps",
	//	description: "Allows reading apps.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*", action: "read_app"},
	//	},
	//},
	//"apps:update": {
	//	name:        "Update apps",
	//	description: "Allows updating apps.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*", action: "update_app"},
	//	},
	//},
	//"apps:delete": {
	//	name:        "Delete apps",
	//	description: "Allows deleting apps.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*", action: "delete_app"},
	//	},
	//},
	// Not implemented yet
	// "environments:create": {
	// 	name:        "Create environments",
	// 	description: "Allows creating environments.",
	// 	resource:    "projects/*/apps/*/environments/*",
	// 	action:      "create_environment",
	// },
	//"environments:read": {
	//	name:        "Read environments",
	//	description: "Allows reading environments.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*", action: "read_environment"},
	//	},
	//},
	// Not implemented yet
	// "environments:update": {
	// 	name:        "Update environments",
	// 	description: "Allows updating environments.",
	// 	resource:    "projects/*/apps/*/environments/*",
	// 	action:      "update_environment",
	// },
	// "environments:delete": {
	// 	name:        "Delete environments",
	// 	description: "Allows deleting environments.",
	// 	resource:    "projects/*/apps/*/environments/*",
	// 	action:      "delete_environment",
	// },
	//"deployments:create": {
	//	name:        "Create deployments",
	//	description: "Allows creating deployments.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/**", action: "create_deployment"},
	//	},
	//},
	//"deployments:read": {
	//	name:        "Read deployments",
	//	description: "Allows reading deployments.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/**", action: "read_deployment"},
	//	},
	//},
	//"deployments:update": {
	//	name:        "Update deployments",
	//	description: "Allows updating deployments.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/deployments/*", action: "update_deployment"},
	//	},
	//},
	//"deployments:delete": {
	//	name:        "Delete deployments",
	//	description: "Allows deleting deployments.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/deployments/*", action: "delete_deployment"},
	//	},
	//},
	//"deployment_instances:read": {
	//	name:        "Read deployment instances",
	//	description: "Allows reading deployment instances.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/deployments/*/instances/*", action: "read_deployment_instance"},
	//	},
	//},
	//"domains:create": {
	//	name:        "Create domains",
	//	description: "Allows creating domains.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/domains/*", action: "create_domain"},
	//	},
	//},
	//"domains:read": {
	//	name:        "Read domains",
	//	description: "Allows reading domains.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/domains/*", action: "read_domain"},
	//	},
	//},
	//"domains:update": {
	//	name:        "Update domains",
	//	description: "Allows updating domains.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/domains/*", action: "update_domain"},
	//	},
	//},
	//"domains:delete": {
	//	name:        "Delete domains",
	//	description: "Allows deleting domains.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/domains/*", action: "delete_domain"},
	//	},
	//},
	//"variables:create": {
	//	name:        "Create variables",
	//	description: "Allows creating variables.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/variables/*", action: "create_variable"},
	//	},
	//},
	//"variables:read": {
	//	name:        "Read variables",
	//	description: "Allows reading variables.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/variables/*", action: "read_variable"},
	//	},
	//},
	//"variables:update": {
	//	name:        "Update variables",
	//	description: "Allows updating variables.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/variables/*", action: "update_variable"},
	//	},
	//},
	//"variables:delete": {
	//	name:        "Delete variables",
	//	description: "Allows deleting variables.",
	//	permissions: []permissionGrant{
	//		{resource: "projects/*/apps/*/environments/*/variables/*", action: "delete_variable"},
	//	},
	//},
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
//	keys:create        => unkey:v1:ws_1:keyspaces/*/keys/*#create_key
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
