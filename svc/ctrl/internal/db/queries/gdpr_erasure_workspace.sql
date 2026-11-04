-- name: EraseWorkspaceLegacyRootKeyPermissions :execrows
DELETE FROM keys_permissions
WHERE keys_permissions.key_id IN (SELECT `keys`.id FROM `keys` WHERE `keys`.for_workspace_id = sqlc.arg(workspace_id))
LIMIT 10000;

-- name: EraseWorkspaceLegacyRootKeyRoles :execrows
DELETE FROM keys_roles
WHERE keys_roles.key_id IN (SELECT `keys`.id FROM `keys` WHERE `keys`.for_workspace_id = sqlc.arg(workspace_id))
LIMIT 10000;

-- name: EraseWorkspaceLegacyRootKeyEncryptions :execrows
DELETE FROM encrypted_keys
WHERE encrypted_keys.key_id IN (SELECT `keys`.id FROM `keys` WHERE `keys`.for_workspace_id = sqlc.arg(workspace_id))
LIMIT 10000;

-- name: EraseWorkspaceLegacyRootKeyRatelimits :execrows
DELETE FROM ratelimits
WHERE ratelimits.key_id IN (SELECT `keys`.id FROM `keys` WHERE `keys`.for_workspace_id = sqlc.arg(workspace_id))
LIMIT 10000;

-- name: EraseWorkspaceLegacyRootKeys :execrows
DELETE FROM `keys`
WHERE for_workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceConnectionAppTargets :execrows
DELETE FROM connection_app_targets
WHERE connection_id IN (SELECT app_connections.id FROM app_connections WHERE app_connections.workspace_id = sqlc.arg(workspace_id))
LIMIT 10000;

-- name: EraseWorkspaceDeploymentConnectionAppTargets :execrows
DELETE FROM deployment_connection_app_targets
WHERE deployment_id IN (SELECT deployments.id FROM deployments WHERE deployments.workspace_id = sqlc.arg(workspace_id))
LIMIT 10000;

-- name: EraseWorkspaceFrontlineRoutes :execrows
DELETE FROM frontline_routes
WHERE project_id IN (SELECT projects.id FROM projects WHERE projects.workspace_id = sqlc.arg(workspace_id))
LIMIT 10000;

-- name: EraseWorkspaceAcmeChallenges :execrows
DELETE FROM acme_challenges
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceAcmeUsers :execrows
DELETE FROM acme_users
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceApis :execrows
DELETE FROM apis
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceAppBuildSettings :execrows
DELETE FROM app_build_settings
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceAppConnections :execrows
DELETE FROM app_connections
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceAppEnvironmentVariables :execrows
DELETE FROM app_environment_variables
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceAppRegionalSettings :execrows
DELETE FROM app_regional_settings
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceAppRuntimeSettings :execrows
DELETE FROM app_runtime_settings
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceAppSourceOci :execrows
DELETE FROM app_source_oci
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceApps :execrows
DELETE FROM apps
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceBillingSubscriptions :execrows
DELETE FROM billing_subscriptions
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceCertificates :execrows
DELETE FROM certificates
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceCiliumNetworkPolicies :execrows
DELETE FROM cilium_network_policies
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceCliSessions :execrows
DELETE FROM cli_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceClickhouseOutbox :execrows
DELETE FROM clickhouse_outbox
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceClickhouseWorkspaceSettings :execrows
DELETE FROM clickhouse_workspace_settings
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceComputeSignupCreditClaims :execrows
DELETE FROM compute_signup_credit_claims
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceCustomDomains :execrows
DELETE FROM custom_domains
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceDeploymentConnections :execrows
DELETE FROM deployment_connections
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceDeploymentSteps :execrows
DELETE FROM deployment_steps
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceDeploymentTopology :execrows
DELETE FROM deployment_topology
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceDeployments :execrows
DELETE FROM deployments
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceDeviceAuthorizations :execrows
DELETE FROM device_authorizations
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceEncryptedKeys :execrows
DELETE FROM encrypted_keys
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceEnvironments :execrows
DELETE FROM environments
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceFlagOverrides :execrows
DELETE FROM flag_overrides
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceGithubAppInstallations :execrows
DELETE FROM github_app_installations
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceGithubRepoConnections :execrows
DELETE FROM github_repo_connections
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceHorizontalAutoscalingPolicies :execrows
DELETE FROM horizontal_autoscaling_policies
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceIdentities :execrows
DELETE FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceInstances :execrows
DELETE FROM instances
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceKeyAuth :execrows
DELETE FROM key_auth
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceKeyMigrations :execrows
DELETE FROM key_migrations
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceKeys :execrows
DELETE FROM `keys`
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceKeysPermissions :execrows
DELETE FROM keys_permissions
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceKeysRoles :execrows
DELETE FROM keys_roles
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceLimits :execrows
DELETE FROM limits
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceLogdrains :execrows
DELETE FROM logdrains
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceOpenapiSpecs :execrows
DELETE FROM openapi_specs
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspacePermissions :execrows
DELETE FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspacePortalDomains :execrows
DELETE FROM portal_domains
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspacePortalSessions :execrows
DELETE FROM portal_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspacePortals :execrows
DELETE FROM portals
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceProjects :execrows
DELETE FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceRatelimitGlobalCounters :execrows
DELETE FROM ratelimit_global_counters
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceRatelimitNamespaces :execrows
DELETE FROM ratelimit_namespaces
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceRatelimitOverrides :execrows
DELETE FROM ratelimit_overrides
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceRatelimits :execrows
DELETE FROM ratelimits
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceRoles :execrows
DELETE FROM roles
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceRolesPermissions :execrows
DELETE FROM roles_permissions
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceSharedSecrets :execrows
DELETE FROM shared_secrets
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceUnkeyPrincipalPermissions :execrows
DELETE FROM unkey_principal_permissions
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceUnkeyRootKeys :execrows
DELETE FROM unkey_root_keys
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspaceWorkspaceBilling :execrows
DELETE FROM workspace_billing
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 10000;

-- name: EraseWorkspace :execrows
DELETE FROM workspaces
WHERE id = sqlc.arg(workspace_id)
LIMIT 10000;
