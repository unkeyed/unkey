package migration

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/unkeyed/unkey/pkg/urn"
)

// Scope supplies canonical resource names resolved by the caller in its workspace.
// Concrete mappings must not contain wildcards. Wildcard grants need no lookup.
type Scope struct {
	WorkspaceID string
	// APIs is keyed by legacy API ID, not keyspace ID. Values are keyspace URNs.
	APIs map[string]urn.V1
	// Namespaces is keyed by namespace ID, not namespace name. Values are namespace URNs.
	Namespaces map[string]urn.V1
	// Apps is keyed by app ID. Values are app URNs.
	Apps map[string]urn.V1
	// Environments is keyed by environment ID, not environment slug. Values are environment URNs.
	Environments map[string]urn.V1
	// Identities is keyed by internal identity ID, not external ID. Values are identity URNs.
	Identities map[string]urn.V1
}

// ErrUnsupported indicates an unknown permission or a resource without a mapping.
var ErrUnsupported = errors.New("unsupported legacy permission")

// ErrInvalidScope indicates missing, foreign, malformed, or widened resource context.
var ErrInvalidScope = errors.New("invalid migration scope")

var (
	apiRead                       = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.read_api$`)
	apiDelete                     = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.delete_api$`)
	apiReadKey                    = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.read_key$`)
	apiReadAnalytics              = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.read_analytics$`)
	apiDecryptKey                 = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.decrypt_key$`)
	apiDeleteKey                  = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.delete_key$`)
	apiVerifyKey                  = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.verify_key$`)
	namespaceLimit                = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.limit$`)
	namespaceReadOverride         = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.read_override$`)
	namespaceSetOverride          = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.set_override$`)
	namespaceDeleteOverride       = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.delete_override$`)
	namespaceRead                 = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.read_namespace$`)
	namespaceDelete               = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.delete_namespace$`)
	namespaceReadAnalytics        = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.read_analytics$`)
	projectRead                   = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.read_project$`)
	projectDelete                 = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.delete_project$`)
	projectReadDeployment         = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.read_deployment$`)
	projectReadRuntimeLogs        = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.read_runtime_logs$`)
	projectReadGatewayRequests    = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.read_gateway_requests$`)
	appRead                       = regexp.MustCompile(`^app\.(\*|[^.*/:# \t\r\n]+)\.read_app$`)
	appDelete                     = regexp.MustCompile(`^app\.(\*|[^.*/:# \t\r\n]+)\.delete_app$`)
	environmentRead               = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.read_environment$`)
	environmentReadDeployment     = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.read_deployment$`)
	environmentReadVariables      = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.read_environment_variables$`)
	environmentRemoveVariables    = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.remove_environment_variables$`)
	environmentReadPolicies       = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.read_policies$`)
	environmentReadDomain         = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.read_domain$`)
	environmentDeleteDomain       = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.delete_domain$`)
	identityRead                  = regexp.MustCompile(`^identity\.(\*|[^.*/:# \t\r\n]+)\.read_identity$`)
	identityDelete                = regexp.MustCompile(`^identity\.(\*|[^.*/:# \t\r\n]+)\.delete_identity$`)
	roleRead                      = regexp.MustCompile(`^rbac\.\*\.read_role$`)
	roleDelete                    = regexp.MustCompile(`^rbac\.\*\.delete_role$`)
	permissionRead                = regexp.MustCompile(`^rbac\.\*\.read_permission$`)
	permissionDelete              = regexp.MustCompile(`^rbac\.\*\.delete_permission$`)
	apiCreate                     = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.create_api$`)
	apiUpdate                     = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.update_api$`)
	apiCreateKey                  = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.create_key$`)
	apiUpdateKey                  = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.update_key$`)
	apiEncryptKey                 = regexp.MustCompile(`^api\.(\*|[^.*/:# \t\r\n]+)\.encrypt_key$`)
	namespaceCreate               = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.create_namespace$`)
	namespaceUpdate               = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.update_namespace$`)
	namespaceListOverrides        = regexp.MustCompile(`^ratelimit\.(\*|[^.*/:# \t\r\n]+)\.list_overrides$`)
	projectCreate                 = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.create_project$`)
	projectUpdate                 = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.update_project$`)
	projectCreateApp              = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.create_app$`)
	projectCreateDeployment       = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.create_deployment$`)
	projectGenerateUploadURL      = regexp.MustCompile(`^project\.(\*|[^.*/:# \t\r\n]+)\.generate_upload_url$`)
	appUpdate                     = regexp.MustCompile(`^app\.(\*|[^.*/:# \t\r\n]+)\.update_app$`)
	appConnectRepository          = regexp.MustCompile(`^app\.(\*|[^.*/:# \t\r\n]+)\.connect_repository$`)
	environmentUpdate             = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.update_environment$`)
	environmentPromoteDeployment  = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.promote_deployment$`)
	environmentRollbackDeployment = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.rollback_deployment$`)
	environmentCreateDeployment   = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.create_deployment$`)
	environmentStartDeployment    = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.start_deployment$`)
	environmentStopDeployment     = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.stop_deployment$`)
	environmentSetVariables       = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.set_environment_variables$`)
	environmentSetPolicies        = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.set_policies$`)
	environmentUpdatePolicy       = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.update_policy$`)
	environmentCreateDomain       = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.create_domain$`)
	environmentVerifyDomain       = regexp.MustCompile(`^environment\.(\*|[^.*/:# \t\r\n]+)\.verify_domain$`)
	identityCreate                = regexp.MustCompile(`^identity\.(\*|[^.*/:# \t\r\n]+)\.create_identity$`)
	identityUpdate                = regexp.MustCompile(`^identity\.(\*|[^.*/:# \t\r\n]+)\.update_identity$`)
	permissionCreate              = regexp.MustCompile(`^rbac\.\*\.create_permission$`)
	permissionUpdate              = regexp.MustCompile(`^rbac\.\*\.update_permission$`)
	roleCreate                    = regexp.MustCompile(`^rbac\.\*\.create_role$`)
	roleUpdate                    = regexp.MustCompile(`^rbac\.\*\.update_role$`)
	roleAddPermission             = regexp.MustCompile(`^rbac\.\*\.add_permission_to_role$`)
	roleRemovePermission          = regexp.MustCompile(`^rbac\.\*\.remove_permission_from_role$`)
	keyAddPermission              = regexp.MustCompile(`^rbac\.\*\.add_permission_to_key$`)
	keyRemovePermission           = regexp.MustCompile(`^rbac\.\*\.remove_permission_from_key$`)
	keyAddRole                    = regexp.MustCompile(`^rbac\.\*\.add_role_to_key$`)
	keyRemoveRole                 = regexp.MustCompile(`^rbac\.\*\.remove_role_from_key$`)
	workspaceInstallGithub        = regexp.MustCompile(`^workspace\.\*\.install_github$`)
	workspaceCreateRootKey        = regexp.MustCompile(`^workspace\.\*\.create_root_key$`)
)

// Translate converts one legacy permission to a URN permission without storage
// or authorization. Create and update actions map to write, which can grant
// additional operations. Callers must authorize the result independently.
// It returns an empty string on failure.
func Translate(legacy string, scope Scope) (string, error) {
	if strings.Contains(scope.WorkspaceID, "*") {
		return "", ErrInvalidScope
	}
	if _, err := urn.ParseV1((urn.V1{WorkspaceID: scope.WorkspaceID, Resource: "**"}).String()); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidScope, err)
	}
	parts := strings.Split(legacy, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", fmt.Errorf("%w: expected resource.id.action, got %q", ErrUnsupported, legacy)
	}
	switch {
	case apiRead.MatchString(legacy):
		return translateAPI(apiRead.FindStringSubmatch(legacy)[1], "#read", scope)
	case apiDelete.MatchString(legacy):
		return translateAPI(apiDelete.FindStringSubmatch(legacy)[1], "#delete", scope)
	case apiReadKey.MatchString(legacy):
		return translateAPI(apiReadKey.FindStringSubmatch(legacy)[1], "/keys/*#read", scope)
	case apiReadAnalytics.MatchString(legacy):
		return translateAPI(apiReadAnalytics.FindStringSubmatch(legacy)[1], "/logs#read", scope)
	case apiDecryptKey.MatchString(legacy):
		return translateAPI(apiDecryptKey.FindStringSubmatch(legacy)[1], "/keys/*#decrypt", scope)
	case apiDeleteKey.MatchString(legacy):
		return translateAPI(apiDeleteKey.FindStringSubmatch(legacy)[1], "/keys/*#delete", scope)
	case apiVerifyKey.MatchString(legacy):
		return translateAPI(apiVerifyKey.FindStringSubmatch(legacy)[1], "/keys/*#verify", scope)
	case namespaceLimit.MatchString(legacy):
		return translateNamespace(namespaceLimit.FindStringSubmatch(legacy)[1], "#limit", scope)
	case namespaceReadOverride.MatchString(legacy):
		return translateNamespace(namespaceReadOverride.FindStringSubmatch(legacy)[1], "/overrides/*#read", scope)
	case namespaceSetOverride.MatchString(legacy):
		return translateNamespace(namespaceSetOverride.FindStringSubmatch(legacy)[1], "/overrides/*#write", scope)
	case namespaceDeleteOverride.MatchString(legacy):
		return translateNamespace(namespaceDeleteOverride.FindStringSubmatch(legacy)[1], "/overrides/*#delete", scope)
	case namespaceRead.MatchString(legacy):
		return translateNamespace(namespaceRead.FindStringSubmatch(legacy)[1], "#read", scope)
	case namespaceDelete.MatchString(legacy):
		return translateNamespace(namespaceDelete.FindStringSubmatch(legacy)[1], "#delete", scope)
	case namespaceReadAnalytics.MatchString(legacy):
		return translateNamespace(namespaceReadAnalytics.FindStringSubmatch(legacy)[1], "/logs#read", scope)
	case projectRead.MatchString(legacy):
		return translateProject(projectRead.FindStringSubmatch(legacy)[1], "#read", scope)
	case projectDelete.MatchString(legacy):
		return translateProject(projectDelete.FindStringSubmatch(legacy)[1], "#delete", scope)
	case projectReadDeployment.MatchString(legacy):
		return translateProject(projectReadDeployment.FindStringSubmatch(legacy)[1], "/apps/*/environments/*/deployments/*#read", scope)
	case projectReadRuntimeLogs.MatchString(legacy):
		return translateProject(projectReadRuntimeLogs.FindStringSubmatch(legacy)[1], "/apps/*/environments/*/deployments/*/logs#read", scope)
	case projectReadGatewayRequests.MatchString(legacy):
		return translateProject(projectReadGatewayRequests.FindStringSubmatch(legacy)[1], "/apps/*/environments/*/gateway/logs#read", scope)
	case appRead.MatchString(legacy):
		return translateResource(appRead.FindStringSubmatch(legacy)[1], "#read", "projects/*/apps/*", scope.Apps, scope)
	case appDelete.MatchString(legacy):
		return translateResource(appDelete.FindStringSubmatch(legacy)[1], "#delete", "projects/*/apps/*", scope.Apps, scope)
	case environmentRead.MatchString(legacy):
		return translateResource(environmentRead.FindStringSubmatch(legacy)[1], "#read", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentReadDeployment.MatchString(legacy):
		return translateResource(environmentReadDeployment.FindStringSubmatch(legacy)[1], "/deployments/*#read", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentReadVariables.MatchString(legacy):
		return translateResource(environmentReadVariables.FindStringSubmatch(legacy)[1], "/variables/*#read", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentRemoveVariables.MatchString(legacy):
		return translateResource(environmentRemoveVariables.FindStringSubmatch(legacy)[1], "/variables/*#delete", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentReadPolicies.MatchString(legacy):
		return translateResource(environmentReadPolicies.FindStringSubmatch(legacy)[1], "/gateway/policies/*#read", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentReadDomain.MatchString(legacy):
		return translateResource(environmentReadDomain.FindStringSubmatch(legacy)[1], "/domains/*#read", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentDeleteDomain.MatchString(legacy):
		return translateResource(environmentDeleteDomain.FindStringSubmatch(legacy)[1], "/domains/*#delete", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case identityRead.MatchString(legacy):
		return translateResource(identityRead.FindStringSubmatch(legacy)[1], "#read", "projects/*/identities/*", scope.Identities, scope)
	case identityDelete.MatchString(legacy):
		return translateResource(identityDelete.FindStringSubmatch(legacy)[1], "#delete", "projects/*/identities/*", scope.Identities, scope)
	case roleRead.MatchString(legacy):
		return translateProject("*", "/rbac/roles/*#read", scope)
	case roleDelete.MatchString(legacy):
		return translateProject("*", "/rbac/roles/*#delete", scope)
	case permissionRead.MatchString(legacy):
		return translateProject("*", "/rbac/permissions/*#read", scope)
	case permissionDelete.MatchString(legacy):
		return translateProject("*", "/rbac/permissions/*#delete", scope)
	case apiCreate.MatchString(legacy):
		return translateAPI(apiCreate.FindStringSubmatch(legacy)[1], "#write", scope)
	case apiUpdate.MatchString(legacy):
		return translateAPI(apiUpdate.FindStringSubmatch(legacy)[1], "#write", scope)
	case apiCreateKey.MatchString(legacy):
		return translateAPI(apiCreateKey.FindStringSubmatch(legacy)[1], "/keys/*#write", scope)
	case apiUpdateKey.MatchString(legacy):
		return translateAPI(apiUpdateKey.FindStringSubmatch(legacy)[1], "/keys/*#write", scope)
	case apiEncryptKey.MatchString(legacy):
		return translateAPI(apiEncryptKey.FindStringSubmatch(legacy)[1], "/keys/*#write", scope)
	case namespaceCreate.MatchString(legacy):
		return translateNamespace(namespaceCreate.FindStringSubmatch(legacy)[1], "#write", scope)
	case namespaceUpdate.MatchString(legacy):
		return translateNamespace(namespaceUpdate.FindStringSubmatch(legacy)[1], "#write", scope)
	case namespaceListOverrides.MatchString(legacy):
		return translateNamespace(namespaceListOverrides.FindStringSubmatch(legacy)[1], "/overrides/*#read", scope)
	case projectCreate.MatchString(legacy):
		return translateProject(projectCreate.FindStringSubmatch(legacy)[1], "#write", scope)
	case projectUpdate.MatchString(legacy):
		return translateProject(projectUpdate.FindStringSubmatch(legacy)[1], "#write", scope)
	case projectCreateApp.MatchString(legacy):
		return translateProject(projectCreateApp.FindStringSubmatch(legacy)[1], "/apps/*#write", scope)
	case projectCreateDeployment.MatchString(legacy):
		return translateProject(projectCreateDeployment.FindStringSubmatch(legacy)[1], "/apps/*/environments/*/deployments/*#write", scope)
	case projectGenerateUploadURL.MatchString(legacy):
		return translateProject(projectGenerateUploadURL.FindStringSubmatch(legacy)[1], "/apps/*/environments/*/deployments/*#write", scope)
	case appUpdate.MatchString(legacy):
		return translateResource(appUpdate.FindStringSubmatch(legacy)[1], "#write", "projects/*/apps/*", scope.Apps, scope)
	case appConnectRepository.MatchString(legacy):
		return translateResource(appConnectRepository.FindStringSubmatch(legacy)[1], "#write", "projects/*/apps/*", scope.Apps, scope)
	case environmentUpdate.MatchString(legacy):
		return translateResource(environmentUpdate.FindStringSubmatch(legacy)[1], "#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentPromoteDeployment.MatchString(legacy):
		return translateResource(environmentPromoteDeployment.FindStringSubmatch(legacy)[1], "#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentRollbackDeployment.MatchString(legacy):
		return translateResource(environmentRollbackDeployment.FindStringSubmatch(legacy)[1], "#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentCreateDeployment.MatchString(legacy):
		return translateResource(environmentCreateDeployment.FindStringSubmatch(legacy)[1], "/deployments/*#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentStartDeployment.MatchString(legacy):
		return translateResource(environmentStartDeployment.FindStringSubmatch(legacy)[1], "/deployments/*#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentStopDeployment.MatchString(legacy):
		return translateResource(environmentStopDeployment.FindStringSubmatch(legacy)[1], "/deployments/*#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentSetVariables.MatchString(legacy):
		return translateResource(environmentSetVariables.FindStringSubmatch(legacy)[1], "/variables/*#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentSetPolicies.MatchString(legacy):
		return translateResource(environmentSetPolicies.FindStringSubmatch(legacy)[1], "/gateway/policies/*#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentUpdatePolicy.MatchString(legacy):
		return translateResource(environmentUpdatePolicy.FindStringSubmatch(legacy)[1], "/gateway/policies/*#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentCreateDomain.MatchString(legacy):
		return translateResource(environmentCreateDomain.FindStringSubmatch(legacy)[1], "/domains/*#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case environmentVerifyDomain.MatchString(legacy):
		return translateResource(environmentVerifyDomain.FindStringSubmatch(legacy)[1], "/domains/*#write", "projects/*/apps/*/environments/*", scope.Environments, scope)
	case identityCreate.MatchString(legacy):
		return translateResource(identityCreate.FindStringSubmatch(legacy)[1], "#write", "projects/*/identities/*", scope.Identities, scope)
	case identityUpdate.MatchString(legacy):
		return translateResource(identityUpdate.FindStringSubmatch(legacy)[1], "#write", "projects/*/identities/*", scope.Identities, scope)
	case permissionCreate.MatchString(legacy), permissionUpdate.MatchString(legacy):
		return translateProject("*", "/rbac/permissions/*#write", scope)
	case roleCreate.MatchString(legacy), roleUpdate.MatchString(legacy), roleAddPermission.MatchString(legacy), roleRemovePermission.MatchString(legacy):
		return translateProject("*", "/rbac/roles/*#write", scope)
	case keyAddPermission.MatchString(legacy), keyRemovePermission.MatchString(legacy), keyAddRole.MatchString(legacy), keyRemoveRole.MatchString(legacy):
		return translateProject("*", "/keyspaces/*/keys/*#write", scope)
	case workspaceInstallGithub.MatchString(legacy):
		return "unkey:v1:" + scope.WorkspaceID + ":github/apps/*#write", nil
	case workspaceCreateRootKey.MatchString(legacy):
		return "unkey:v1:" + scope.WorkspaceID + ":rootKeys/*#write", nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnsupported, legacy)
	}
}

func translateProject(projectID, suffix string, scope Scope) (string, error) {
	resource, err := urn.ParseV1("unkey:v1:" + scope.WorkspaceID + ":projects/" + projectID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidScope, err)
	}
	return resource.String() + suffix, nil
}

func translateResource(id, suffix, pattern string, resources map[string]urn.V1, scope Scope) (string, error) {
	resource := urn.V1{WorkspaceID: scope.WorkspaceID, Resource: pattern}
	if id != "*" {
		resolved := resources[id]
		if _, err := urn.ParseV1(resolved.String()); err != nil {
			return "", fmt.Errorf("%w: %v", ErrInvalidScope, err)
		}
		if strings.Contains(resolved.Resource, "*") || !resource.Covers(resolved) || !strings.HasSuffix(resolved.Resource, "/"+id) {
			return "", ErrInvalidScope
		}
		resource = resolved
	}
	return resource.String() + suffix, nil
}

func translateAPI(apiID, suffix string, scope Scope) (string, error) {
	resource := urn.V1{WorkspaceID: scope.WorkspaceID, Resource: "projects/*/keyspaces/*"}
	if apiID != "*" {
		resource = scope.APIs[apiID]
		if err := validateResource(resource, scope.WorkspaceID, "keyspaces", 4); err != nil {
			return "", err
		}
	}
	return resource.String() + suffix, nil
}

func translateNamespace(namespaceID, suffix string, scope Scope) (string, error) {
	resource := urn.V1{WorkspaceID: scope.WorkspaceID, Resource: "projects/*/ratelimits/namespaces/*"}
	if namespaceID != "*" {
		resource = scope.Namespaces[namespaceID]
		if err := validateResource(resource, scope.WorkspaceID, "ratelimits", 5); err != nil {
			return "", err
		}
		if !strings.HasSuffix(resource.Resource, "/namespaces/"+namespaceID) {
			return "", ErrInvalidScope
		}
	}
	return resource.String() + suffix, nil
}

func validateResource(resource urn.V1, workspaceID, kind string, segments int) error {
	if _, err := urn.ParseV1(resource.String()); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidScope, err)
	}
	parts := strings.Split(resource.Resource, "/")
	if resource.WorkspaceID != workspaceID || strings.Contains(resource.Resource, "*") ||
		len(parts) != segments || parts[0] != "projects" || parts[2] != kind {
		return ErrInvalidScope
	}
	return nil
}
