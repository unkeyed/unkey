package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/api/routes"
	"gopkg.in/yaml.v3"
)

func TestPublicAPIOperationsHaveRegisteredRoutesAndDeclaredSecurity(t *testing.T) {
	type operation struct {
		Security *[]map[string][]string `yaml:"security"`
	}
	var spec struct {
		Security []map[string][]string `yaml:"security"`
		Paths    map[string]struct {
			Get     *operation `yaml:"get"`
			Post    *operation `yaml:"post"`
			Put     *operation `yaml:"put"`
			Patch   *operation `yaml:"patch"`
			Delete  *operation `yaml:"delete"`
			Options *operation `yaml:"options"`
			Head    *operation `yaml:"head"`
			Trace   *operation `yaml:"trace"`
		} `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(openapi.Spec, &spec))

	expected := map[string][]string{
		"bearer": strings.Fields(`
			/v2/analytics.getGatewayRequests /v2/analytics.getRatelimits
			/v2/analytics.getRuntimeLogs /v2/analytics.getVerifications
			/v2/apis.createApi /v2/apis.deleteApi /v2/apis.getApi /v2/apis.listKeys
			/v2/deployments.createDeployment /v2/deployments.getDeployment
			/v2/deployments.listBuildLogs /v2/deployments.listDeployments
			/v2/deployments.stopDeployment /v2/deployments.startDeployment
			/v2/deployments.promoteDeployment /v2/deployments.rollbackDeployment
			/v3/deployments.createDeployment /v2/deploy.createDeployment /v2/deploy.getDeployment
			/v2/identities.createIdentity /v2/identities.deleteIdentity /v2/identities.getIdentity
			/v2/identities.listIdentities /v2/identities.updateIdentity
			/v2/keys.addPermissions /v2/keys.setPermissions /v2/keys.removePermissions
			/v2/keys.addRoles /v2/keys.setRoles /v2/keys.removeRoles
			/v2/keys.createKey /v2/keys.rerollKey /v2/keys.updateKey /v2/keys.updateCredits
			/v2/keys.deleteKey /v2/keys.getKey /v2/keys.whoami /v2/keys.verifyKey /v2/keys.migrateKeys
			/v2/rootKeys.createKey /v2/rootKeys.listKeys /v2/rootKeys.updateKey
			/v2/rootKeys.deleteKey /v2/rootKeys.rerollKey
			/v2/ratelimit.limit /v2/ratelimit.multiLimit /v2/ratelimit.setOverride
			/v2/ratelimit.getOverride /v2/ratelimit.listOverrides /v2/ratelimit.deleteOverride
			/v2/ratelimit.createNamespace /v2/ratelimit.listNamespaces
			/v2/ratelimit.updateNamespace /v2/ratelimit.deleteNamespace
			/v2/portal.createPortal /v2/portal.getPortal /v2/portal.updatePortal /v2/portal.deletePortal
			/v2/portal.createSession /v2/portal.revokeSession /v2/portal.listSessions
			/v2/projects.createProject /v2/projects.listProjects /v2/projects.getProject
			/v2/projects.updateProject /v2/projects.deleteProject /v2/github.installApp
			/v2/workspace.getLimits /v2/workspace.getUsage
			/v2/workspace.getFlags /v2/workspace.updateFlag
			/v2/apps.createApp /v2/apps.getApp /v2/apps.listApps /v2/apps.updateApp /v2/apps.deleteApp
			/v2/environments.getEnvironment /v2/environments.listEnvironments
			/v2/environments.updateSettings /v2/environments.setEnvironmentVariables
			/v2/environments.removeEnvironmentVariables /v2/environments.listEnvironmentVariables
			/v2/domains.createDomain /v2/domains.getDomain /v2/domains.listDomains
			/v2/domains.deleteDomain /v2/domains.verifyDomain
			/v2/gateway.setPolicies /v2/gateway.listPolicies /v2/gateway.updatePolicy
			/v2/permissions.setRolePermissions /v2/permissions.createRole /v2/permissions.getRole
			/v2/permissions.listRoles /v2/permissions.deleteRole /v2/permissions.createPermission
			/v2/permissions.getPermission /v2/permissions.listPermissions /v2/permissions.deletePermission
			/v2/permissions.updatePermission /v2/permissions.updateRole
		`),
		"portalSession": {
			"/v2/portal.listKeys",
			"/v2/portal.rerollKey",
			"/v2/portal.getVerifications",
		},
		"public": {"/v2/liveness", "/v2/portal.exchangeCode"},
	}

	validator, err := validation.New()
	require.NoError(t, err)
	srv, err := zen.New(zen.Config{})
	require.NoError(t, err)
	routes.Register(srv, &routes.Services{Validator: validator}, zen.InstanceInfo{})

	covered := map[string]bool{}
	for scheme, paths := range expected {
		for _, path := range paths {
			t.Run(path, func(t *testing.T) {
				require.False(t, covered[path], "operation must have exactly one security classification")
				covered[path] = true

				pathSpec, exists := spec.Paths[path]
				require.True(t, exists, "expected operation is absent from the schema")
				method, op := http.MethodPost, pathSpec.Post
				if path == "/v2/liveness" {
					method, op = http.MethodGet, pathSpec.Get
				}
				require.NotNil(t, op)

				security := spec.Security
				if op.Security != nil {
					security = *op.Security
				}
				if scheme == "public" {
					require.Empty(t, security)
				} else {
					require.Equal(t, []map[string][]string{{scheme: {}}}, security)
				}

				req := httptest.NewRequest(method, path, nil)
				_, pattern := srv.Mux().Handler(req)
				require.Equal(
					t,
					method+" "+path,
					pattern,
					"operation must be mounted by production registration",
				)
			})
		}
	}

	for path, pathSpec := range spec.Paths {
		require.True(t, covered[path], "classify new operation %s explicitly", path)
		require.Nil(t, pathSpec.Put, "classify PUT %s explicitly", path)
		require.Nil(t, pathSpec.Patch, "classify PATCH %s explicitly", path)
		require.Nil(t, pathSpec.Delete, "classify DELETE %s explicitly", path)
		require.Nil(t, pathSpec.Options, "classify OPTIONS %s explicitly", path)
		require.Nil(t, pathSpec.Head, "classify HEAD %s explicitly", path)
		require.Nil(t, pathSpec.Trace, "classify TRACE %s explicitly", path)
		if path == "/v2/liveness" {
			require.Nil(t, pathSpec.Post)
		} else {
			require.Nil(t, pathSpec.Get)
		}
	}

	for _, path := range []string{"/reference", "/openapi.yaml"} {
		_, pattern := srv.Mux().Handler(httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.MethodGet+" "+path, pattern)
	}
}
