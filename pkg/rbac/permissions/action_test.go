package permissions_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// TestActions_BuildCanonicalPermissions pins every supported resource and
// action pair and its serialized form.
func TestActions_BuildCanonicalPermissions(t *testing.T) {
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
	identity := project.Identity("id_123")
	keyspace := project.Keyspace("ks_123")
	key := keyspace.Key("key_123")
	namespace := project.RatelimitNamespace("ns_123")
	override := namespace.Override("ov_123")
	role := project.RBAC().Role("role_123")
	permission := project.RBAC().Permission("perm_123")

	requirePermission(t, githubApp, permissions.ReadGitHubApp{}, "unkey:v1:ws_123:github/apps/gh_123#read_github_app")
	requirePermission(t, githubApp, permissions.WriteGitHubApp{}, "unkey:v1:ws_123:github/apps/gh_123#write_github_app")
	requirePermission(t, githubApp, permissions.DeleteGitHubApp{}, "unkey:v1:ws_123:github/apps/gh_123#delete_github_app")
	requirePermission(t, project, permissions.ReadProject{}, "unkey:v1:ws_123:projects/proj_123#read_project")
	requirePermission(t, project, permissions.WriteProject{}, "unkey:v1:ws_123:projects/proj_123#write_project")
	requirePermission(t, project, permissions.DeleteProject{}, "unkey:v1:ws_123:projects/proj_123#delete_project")
	requirePermission(t, app, permissions.ReadApp{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123#read_app")
	requirePermission(t, app, permissions.WriteApp{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123#write_app")
	requirePermission(t, app, permissions.DeleteApp{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123#delete_app")
	requirePermission(t, environment, permissions.ReadEnvironment{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123#read_environment")
	requirePermission(t, environment, permissions.WriteEnvironment{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123#write_environment")
	requirePermission(t, environment, permissions.DeleteEnvironment{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123#delete_environment")
	requirePermission(t, deployment, permissions.ReadDeployment{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123#read_deployment")
	requirePermission(t, deployment, permissions.WriteDeployment{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123#write_deployment")
	requirePermission(t, deployment, permissions.DeleteDeployment{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123#delete_deployment")
	requirePermission(t, deployment.Logs(), permissions.ReadDeploymentLogs{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123/logs#read_deployment_logs")
	requirePermission(t, domain, permissions.ReadDomain{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/dom_123#read_domain")
	requirePermission(t, domain, permissions.WriteDomain{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/dom_123#write_domain")
	requirePermission(t, domain, permissions.DeleteDomain{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/dom_123#delete_domain")
	requirePermission(t, variable, permissions.ReadEnvironmentVariable{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123#read_environment_variable")
	requirePermission(t, variable, permissions.WriteEnvironmentVariable{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123#write_environment_variable")
	requirePermission(t, variable, permissions.DeleteEnvironmentVariable{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123#delete_environment_variable")
	requirePermission(t, gateway.Logs(), permissions.ReadGatewayLogs{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/logs#read_gateway_logs")
	requirePermission(t, gateway.Policy("pol_123"), permissions.ReadGatewayPolicy{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/policies/pol_123#read_gateway_policy")
	requirePermission(t, gateway.Policy("pol_123"), permissions.WriteGatewayPolicy{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/policies/pol_123#write_gateway_policy")
	requirePermission(t, gateway.Policy("pol_123"), permissions.DeleteGatewayPolicy{}, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/policies/pol_123#delete_gateway_policy")
	requirePermission(t, identity, permissions.ReadIdentity{}, "unkey:v1:ws_123:projects/proj_123/identities/id_123#read_identity")
	requirePermission(t, identity, permissions.WriteIdentity{}, "unkey:v1:ws_123:projects/proj_123/identities/id_123#write_identity")
	requirePermission(t, identity, permissions.DeleteIdentity{}, "unkey:v1:ws_123:projects/proj_123/identities/id_123#delete_identity")
	requirePermission(t, keyspace, permissions.ReadKeyspace{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123#read_keyspace")
	requirePermission(t, keyspace, permissions.WriteKeyspace{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123#write_keyspace")
	requirePermission(t, keyspace, permissions.DeleteKeyspace{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123#delete_keyspace")
	requirePermission(t, keyspace.Logs(), permissions.ReadKeyspaceLogs{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/logs#read_keyspace_logs")
	requirePermission(t, key, permissions.ReadKey{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#read_key")
	requirePermission(t, key, permissions.WriteKey{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#write_key")
	requirePermission(t, key, permissions.DeleteKey{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#delete_key")
	requirePermission(t, key, permissions.DecryptKey{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#decrypt_key")
	requirePermission(t, key, permissions.VerifyKey{}, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123#verify_key")
	requirePermission(t, namespace, permissions.ReadRatelimitNamespace{}, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read_ratelimit_namespace")
	requirePermission(t, namespace, permissions.WriteRatelimitNamespace{}, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#write_ratelimit_namespace")
	requirePermission(t, namespace, permissions.DeleteRatelimitNamespace{}, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#delete_ratelimit_namespace")
	requirePermission(t, namespace, permissions.LimitRatelimitNamespace{}, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#limit_ratelimit_namespace")
	requirePermission(t, namespace.Logs(), permissions.ReadRatelimitLogs{}, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/logs#read_ratelimit_logs")
	requirePermission(t, override, permissions.ReadRatelimitOverride{}, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#read_ratelimit_override")
	requirePermission(t, override, permissions.WriteRatelimitOverride{}, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#write_ratelimit_override")
	requirePermission(t, override, permissions.DeleteRatelimitOverride{}, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#delete_ratelimit_override")
	requirePermission(t, role, permissions.ReadRole{}, "unkey:v1:ws_123:projects/proj_123/rbac/roles/role_123#read_role")
	requirePermission(t, role, permissions.WriteRole{}, "unkey:v1:ws_123:projects/proj_123/rbac/roles/role_123#write_role")
	requirePermission(t, role, permissions.DeleteRole{}, "unkey:v1:ws_123:projects/proj_123/rbac/roles/role_123#delete_role")
	requirePermission(t, permission, permissions.ReadPermission{}, "unkey:v1:ws_123:projects/proj_123/rbac/permissions/perm_123#read_permission")
	requirePermission(t, permission, permissions.WritePermission{}, "unkey:v1:ws_123:projects/proj_123/rbac/permissions/perm_123#write_permission")
	requirePermission(t, permission, permissions.DeletePermission{}, "unkey:v1:ws_123:projects/proj_123/rbac/permissions/perm_123#delete_permission")
}

// requirePermission verifies the public query builder output for one valid
// resource and action pair.
func requirePermission[R fmt.Stringer, A permissions.Action[R]](t *testing.T, resource R, action A, want string) {
	t.Helper()
	require.Equal(t, want, rbac.U(resource, action).Value)
}
