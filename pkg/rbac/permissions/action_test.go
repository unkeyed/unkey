package permissions_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// TestActions_BuildPlatformPermissions pins every platform resource and action
// pair and its serialized form.
func TestActions_BuildPlatformPermissions(t *testing.T) {
	t.Parallel()

	workspace := urn.New().Workspace("ws_123")
	githubApp := workspace.GitHubApp("gh_123")
	project := workspace.Project("proj_123")
	app := project.App("app_123")
	environment := app.Environment("env_123")
	deployment := environment.Deployment("dep_123")
	domain := environment.Domain("dom_123")
	variable := environment.Variable("var_123")
	gateway := environment.Gateway()
	portal := project.Portal("portal_123")
	portalSession := portal.Session("sess_123")

	requirePermission(t, githubApp, permissions.Read, "unkey:v1:ws_123:github/apps/gh_123#read")
	requirePermission(t, githubApp, permissions.Write, "unkey:v1:ws_123:github/apps/gh_123#write")
	requirePermission(t, githubApp, permissions.Delete, "unkey:v1:ws_123:github/apps/gh_123#delete")
	requirePermission(t, project, permissions.Read, "unkey:v1:ws_123:projects/proj_123#read")
	requirePermission(t, project, permissions.Write, "unkey:v1:ws_123:projects/proj_123#write")
	requirePermission(t, project, permissions.Delete, "unkey:v1:ws_123:projects/proj_123#delete")
	requirePermission(t, app, permissions.Read, "unkey:v1:ws_123:projects/proj_123/apps/app_123#read")
	requirePermission(t, app, permissions.Write, "unkey:v1:ws_123:projects/proj_123/apps/app_123#write")
	requirePermission(t, app, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/apps/app_123#delete")
	requirePermission(t, environment, permissions.Read, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123#read")
	requirePermission(t, environment, permissions.Write, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123#write")
	requirePermission(t, environment, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123#delete")
	requirePermission(t, deployment, permissions.Read, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123#read")
	requirePermission(t, deployment, permissions.Write, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123#write")
	requirePermission(t, deployment, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123#delete")
	requirePermission(t, deployment.Logs(), permissions.Read, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123/logs#read")
	requirePermission(t, domain, permissions.Read, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/dom_123#read")
	requirePermission(t, domain, permissions.Write, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/dom_123#write")
	requirePermission(t, domain, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/dom_123#delete")
	requirePermission(t, variable, permissions.Read, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123#read")
	requirePermission(t, variable, permissions.Write, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123#write")
	requirePermission(t, variable, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123#delete")
	requirePermission(t, gateway.Logs(), permissions.Read, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/logs#read")
	requirePermission(t, gateway.Policy("pol_123"), permissions.Read, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/policies/pol_123#read")
	requirePermission(t, gateway.Policy("pol_123"), permissions.Write, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/policies/pol_123#write")
	requirePermission(t, gateway.Policy("pol_123"), permissions.Delete, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/policies/pol_123#delete")
	requirePermission(t, portal, permissions.Read, "unkey:v1:ws_123:projects/proj_123/portals/portal_123#read")
	requirePermission(t, portal, permissions.Write, "unkey:v1:ws_123:projects/proj_123/portals/portal_123#write")
	requirePermission(t, portal, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/portals/portal_123#delete")
	requirePermission(t, portalSession, permissions.Write, "unkey:v1:ws_123:projects/proj_123/portals/portal_123/sessions/sess_123#write")
}

// TestActions_BuildDataPermissions pins every data resource and action pair and
// its serialized form.
func TestActions_BuildDataPermissions(t *testing.T) {
	t.Parallel()

	project := urn.New().Workspace("ws_123").Project("proj_123")
	identity := project.Identity("id_123")
	keyspace := project.Keyspace("ks_123")
	key := keyspace.Key("key_123")
	namespace := project.RatelimitNamespace("ns_123")
	override := namespace.Override("ov_123")
	role := project.RBAC().Role("role_123")
	permission := project.RBAC().Permission("perm_123")

	requirePermission(t, identity, permissions.Read, "unkey:v1:ws_123:projects/proj_123/identities/id_123#read")
	requirePermission(t, identity, permissions.Write, "unkey:v1:ws_123:projects/proj_123/identities/id_123#write")
	requirePermission(t, identity, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/identities/id_123#delete")
	requirePermission(t, keyspace, permissions.Read, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123#read")
	requirePermission(t, keyspace, permissions.Write, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123#write")
	requirePermission(t, keyspace, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123#delete")
	requirePermission(t, keyspace.Logs(), permissions.Read, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/logs#read")
	requirePermission(t, key, permissions.Read, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#read")
	requirePermission(t, key, permissions.Write, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#write")
	requirePermission(t, key, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#delete")
	requirePermission(t, key, permissions.Decrypt, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#decrypt")
	requirePermission(t, key, permissions.Verify, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#verify")
	requirePermission(t, namespace, permissions.Read, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read")
	requirePermission(t, namespace, permissions.Write, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#write")
	requirePermission(t, namespace, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#delete")
	requirePermission(t, namespace, permissions.Limit, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#limit")
	requirePermission(t, namespace.Logs(), permissions.Read, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/logs#read")
	requirePermission(t, override, permissions.Read, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#read")
	requirePermission(t, override, permissions.Write, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#write")
	requirePermission(t, override, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#delete")
	requirePermission(t, role, permissions.Read, "unkey:v1:ws_123:projects/proj_123/rbac/roles/role_123#read")
	requirePermission(t, role, permissions.Write, "unkey:v1:ws_123:projects/proj_123/rbac/roles/role_123#write")
	requirePermission(t, role, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/rbac/roles/role_123#delete")
	requirePermission(t, permission, permissions.Read, "unkey:v1:ws_123:projects/proj_123/rbac/permissions/perm_123#read")
	requirePermission(t, permission, permissions.Write, "unkey:v1:ws_123:projects/proj_123/rbac/permissions/perm_123#write")
	requirePermission(t, permission, permissions.Delete, "unkey:v1:ws_123:projects/proj_123/rbac/permissions/perm_123#delete")
}

// TestIsValid_RejectsUnsupportedActions guarantees each special resource type
// rejects actions from another type. For example, logs reject write, portal
// sessions reject read, and only the global resource accepts the * action.
func TestIsValid_RejectsUnsupportedActions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		resource string
		action   permissions.Action
	}{
		{name: "root key read", resource: "rootKeys/key_123", action: permissions.Read},
		{name: "keyspace log write", resource: "projects/proj_123/keyspaces/ks_123/logs", action: permissions.Write},
		{name: "key limit", resource: "projects/proj_123/keyspaces/ks_123/keys/key_123", action: permissions.Limit},
		{name: "namespace decrypt", resource: "projects/proj_123/ratelimits/namespaces/ns_123", action: permissions.Decrypt},
		{name: "override limit", resource: "projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123", action: permissions.Limit},
		{name: "portal session read", resource: "projects/proj_123/portals/portal_123/sessions/sess_123", action: permissions.Read},
		{name: "portal session delete", resource: "projects/proj_123/portals/portal_123/sessions/sess_123", action: permissions.Delete},
		{name: "resource action wildcard", resource: "projects/proj_123", action: permissions.Action(permissions.Wildcard)},
		{name: "unknown global action", resource: "**", action: permissions.Action("rotate")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resource, err := urn.ParseV1("unkey:v1:ws_123:" + testCase.resource)
			require.NoError(t, err)
			require.False(t, permissions.IsValid(resource, testCase.action))
		})
	}
}

// TestIsValid_ValidatesGlobalAndDescendantPatterns guarantees a pattern accepts
// only actions used by its selected resources. For example, a project subtree
// accepts limit, a keyspace subtree accepts decrypt but not limit, and **#*
// remains the global administrator permission.
func TestIsValid_ValidatesGlobalAndDescendantPatterns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		resource string
		action   permissions.Action
		valid    bool
	}{
		{name: "global read", resource: "**", action: permissions.Read, valid: true},
		{name: "global write", resource: "**", action: permissions.Write, valid: true},
		{name: "global delete", resource: "**", action: permissions.Delete, valid: true},
		{name: "global decrypt", resource: "**", action: permissions.Decrypt, valid: true},
		{name: "global verify", resource: "**", action: permissions.Verify, valid: true},
		{name: "global limit", resource: "**", action: permissions.Limit, valid: true},
		{name: "global action wildcard", resource: "**", action: permissions.Action(permissions.Wildcard), valid: true},
		{name: "project subtree limit", resource: "projects/proj_123/**", action: permissions.Limit, valid: true},
		{name: "project subtree decrypt", resource: "projects/proj_123/**", action: permissions.Decrypt, valid: true},
		{name: "keyspace subtree decrypt", resource: "projects/proj_123/keyspaces/ks_123/**", action: permissions.Decrypt, valid: true},
		{name: "keyspace subtree verify", resource: "projects/proj_123/keyspaces/ks_123/**", action: permissions.Verify, valid: true},
		{name: "keyspace subtree limit", resource: "projects/proj_123/keyspaces/ks_123/**", action: permissions.Limit, valid: false},
		{name: "namespace subtree limit", resource: "projects/proj_123/ratelimits/namespaces/ns_123/**", action: permissions.Limit, valid: true},
		{name: "portal subtree write", resource: "projects/proj_123/portals/portal_123/**", action: permissions.Write, valid: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resource, err := urn.ParseV1("unkey:v1:ws_123:" + testCase.resource)
			require.NoError(t, err)
			require.Equal(t, testCase.valid, permissions.IsValid(resource, testCase.action))
		})
	}
}

// TestIsValid_RejectsMalformedResources guarantees manually constructed urn.V1
// values cannot bypass URN validation or cause a panic. For example, an empty
// resource and projects//apps/app_123 both return false.
func TestIsValid_RejectsMalformedResources(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		resource urn.V1
	}{
		{name: "zero value", resource: urn.V1{}},
		{name: "invalid workspace", resource: urn.V1{WorkspaceID: "ws/123", Resource: "projects/proj_123"}},
		{name: "empty resource", resource: urn.V1{WorkspaceID: "ws_123"}},
		{name: "unknown resource", resource: urn.V1{WorkspaceID: "ws_123", Resource: "widgets/widget_123"}},
		{name: "empty path segment", resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects//apps/app_123"}},
		{name: "middle descendant wildcard", resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/**/apps/*"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			require.False(t, permissions.IsValid(testCase.resource, permissions.Read))
		})
	}
}

// TestIsValid_ClassifiesStructuralPathPositions guarantees ID text cannot
// change the resource type. For example, a keyspace ID named logs is still a
// keyspace, while a key ID named overrides still supports decrypt.
func TestIsValid_ClassifiesStructuralPathPositions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		resource string
		action   permissions.Action
		valid    bool
	}{
		{name: "project and app IDs use reserved words", resource: "projects/keys/apps/logs", action: permissions.Decrypt, valid: false},
		{name: "keyspace ID named logs is not a log", resource: "projects/proj_123/keyspaces/logs", action: permissions.Write, valid: true},
		{name: "keyspace ID named logs has no key action", resource: "projects/proj_123/keyspaces/logs", action: permissions.Decrypt, valid: false},
		{name: "key ID named overrides remains a key", resource: "projects/proj_123/keyspaces/ks_123/keys/overrides", action: permissions.Decrypt, valid: true},
		{name: "override ID named keys remains an override", resource: "projects/proj_123/ratelimits/namespaces/ns_123/overrides/keys", action: permissions.Limit, valid: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resource, err := urn.ParseV1("unkey:v1:ws_123:" + testCase.resource)
			require.NoError(t, err)
			require.Equal(t, testCase.valid, permissions.IsValid(resource, testCase.action))
		})
	}
}

// requirePermission verifies the public query builder output for one valid
// resource and action pair.
func requirePermission(t *testing.T, resource fmt.Stringer, action permissions.Action, want string) {
	t.Helper()
	require.Equal(t, want, rbac.U(resource, action).Value)
	parsed, err := urn.ParseV1(resource.String())
	require.NoError(t, err)
	require.True(t, permissions.IsValid(parsed, action))
}
