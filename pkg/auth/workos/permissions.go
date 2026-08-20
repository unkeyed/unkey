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

// permissionMappings is a partial list that intentionally contains only slugs
// the API is ready to authorize. Add new slugs with handler coverage for the
// translated Unkey permissions before syncing them to WorkOS.
var permissionMappings = map[string]permissionMapping{
	"admin:*": {
		name:        "Admin",
		description: "Grants full administrative access.",
		permissions: []permissionGrant{
			{resource: "**", action: "*"},
		},
	},
	"deployments:stop": {
		name:        "Stop deployments",
		description: "Allows stopping a running deployment.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.StopDeployment{})},
		},
	},
	"deployments:start": {
		name:        "Start deployments",
		description: "Allows starting a stopped deployment.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.StartDeployment{})},
		},
	},
	"deployments:rollback": {
		name:        "Roll back deployments",
		description: "Allows rolling live traffic back to a previous deployment.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.RollbackDeployment{})},
		},
	},
	"deployments:create": {
		name:        "Create deployments",
		description: "Allows creating and redeploying deployments.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.CreateDeployment{})},
		},
	},
	"deployments:promote": {
		name:        "Promote deployments",
		description: "Allows promoting a deployment to live.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/deployments/*", action: action(rbacpermissions.PromoteDeployment{})},
		},
	},
	"projects:create": {
		name:        "Create projects",
		description: "Allows creating projects.",
		permissions: []permissionGrant{
			{resource: "projects/*", action: action(rbacpermissions.CreateProject{})},
		},
	},
	"projects:update": {
		name:        "Update projects",
		description: "Allows updating projects.",
		permissions: []permissionGrant{
			{resource: "projects/*", action: action(rbacpermissions.UpdateProject{})},
		},
	},
	"projects:delete": {
		name:        "Delete projects",
		description: "Allows deleting projects.",
		permissions: []permissionGrant{
			{resource: "projects/*", action: action(rbacpermissions.DeleteProject{})},
		},
	},
	"apps:create": {
		name:        "Create apps",
		description: "Allows creating apps in a project.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*", action: action(rbacpermissions.CreateApp{})},
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
	"environments:update": {
		name:        "Update environments",
		description: "Allows updating environment build and runtime settings.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.UpdateEnvironment{})},
		},
	},
	"gateway_policies:read": {
		name:        "Read gateway policies",
		description: "Allows reading an environment's gateway policies.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.ReadPolicies{})},
		},
	},
	"gateway_policies:write": {
		name:        "Write gateway policies",
		description: "Allows creating, updating, and deleting an environment's gateway policies.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.WritePolicies{})},
		},
	},
	"environment_variables:read": {
		name:        "Read environment variables",
		description: "Allows reading environment variables, including recoverable values.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.ReadEnvironmentVariables{})},
		},
	},
	"environment_variables:set": {
		name:        "Set environment variables",
		description: "Allows creating and overwriting environment variables.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.SetEnvironmentVariables{})},
		},
	},
	"environment_variables:remove": {
		name:        "Remove environment variables",
		description: "Allows removing environment variables.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*", action: action(rbacpermissions.RemoveEnvironmentVariables{})},
		},
	},
	"domains:create": {
		name:        "Create domains",
		description: "Allows attaching a custom domain to an environment.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/domains/*", action: action(rbacpermissions.CreateDomain{})},
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
	"domains:verify": {
		name:        "Verify domains",
		description: "Allows restarting verification for a custom domain.",
		permissions: []permissionGrant{
			{resource: "projects/*/apps/*/environments/*/domains/*", action: action(rbacpermissions.VerifyDomain{})},
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
		},
	},
	"keys:read": {
		name:        "Read keys",
		description: "Allows reading keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.ReadKey{})},
			{resource: "keyspaces/*", action: action(rbacpermissions.ReadKeyspace{})},
		},
	},
	"keys:update": {
		name:        "Update keys",
		description: "Allows updating keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.UpdateKey{})},
		},
	},
	"keys:verify": {
		name:        "Verify keys",
		description: "Allows verifying keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.VerifyKey{})},
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
		},
	},
	"keys:delete": {
		name:        "Delete keys",
		description: "Allows deleting keys.",
		permissions: []permissionGrant{
			{resource: "keyspaces/*/keys/*", action: action(rbacpermissions.DeleteKey{})},
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
//	keys:create        => unkey:v1:ws_1:keyspaces/*#create_key
//	keys:read          => unkey:v1:ws_1:keyspaces/*/keys/*#read_key
//	keys:update        => unkey:v1:ws_1:keyspaces/*/keys/*#update_key
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
