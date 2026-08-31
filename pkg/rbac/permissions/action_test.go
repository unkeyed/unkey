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

// requirePermission verifies the public query builder output for one valid
// resource and action pair.
func requirePermission(t *testing.T, resource fmt.Stringer, action permissions.Action, want string) {
	t.Helper()
	require.Equal(t, want, rbac.U(resource, action).Value)
}
