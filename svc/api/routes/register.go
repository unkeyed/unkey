package routes

import (
	"time"

	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	openapi "github.com/unkeyed/unkey/svc/api/routes/openapi"
	"github.com/unkeyed/unkey/svc/api/routes/reference"
	v2Liveness "github.com/unkeyed/unkey/svc/api/routes/v2_liveness"

	pprofRoute "github.com/unkeyed/unkey/pkg/pprof"

	v2RatelimitDeleteOverride "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_delete_override"
	v2RatelimitGetOverride "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_get_override"
	v2RatelimitLimit "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_limit"
	v2RatelimitListOverrides "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_list_overrides"
	v2RatelimitMultiLimit "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_multi_limit"
	v2RatelimitSetOverride "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_set_override"

	v2ApisCreateApi "github.com/unkeyed/unkey/svc/api/routes/v2_apis_create_api"
	v2ApisDeleteApi "github.com/unkeyed/unkey/svc/api/routes/v2_apis_delete_api"
	v2ApisGetApi "github.com/unkeyed/unkey/svc/api/routes/v2_apis_get_api"
	v2ApisListKeys "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"

	v2DeployCreateDeployment "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
	v2DeployGetDeployment "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_get_deployment"

	v2IdentitiesCreateIdentity "github.com/unkeyed/unkey/svc/api/routes/v2_identities_create_identity"
	v2IdentitiesDeleteIdentity "github.com/unkeyed/unkey/svc/api/routes/v2_identities_delete_identity"
	v2IdentitiesGetIdentity "github.com/unkeyed/unkey/svc/api/routes/v2_identities_get_identity"
	v2IdentitiesListIdentities "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
	v2IdentitiesUpdateIdentity "github.com/unkeyed/unkey/svc/api/routes/v2_identities_update_identity"

	v2PermissionsCreatePermission "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_permission"
	v2PermissionsCreateRole "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
	v2PermissionsDeletePermission "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_permission"
	v2PermissionsDeleteRole "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_role"
	v2PermissionsGetPermission "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_permission"
	v2PermissionsGetRole "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_role"
	v2PermissionsListPermissions "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_list_permissions"
	v2PermissionsListRoles "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_list_roles"

	v2KeysAddPermissions "github.com/unkeyed/unkey/svc/api/routes/v2_keys_add_permissions"
	v2KeysAddRoles "github.com/unkeyed/unkey/svc/api/routes/v2_keys_add_roles"
	v2KeysCreateKey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_create_key"
	v2KeysDeleteKey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_delete_key"
	v2KeysGetKey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_get_key"
	v2KeysMigrateKeys "github.com/unkeyed/unkey/svc/api/routes/v2_keys_migrate_keys"
	v2KeysRemovePermissions "github.com/unkeyed/unkey/svc/api/routes/v2_keys_remove_permissions"
	v2KeysRemoveRoles "github.com/unkeyed/unkey/svc/api/routes/v2_keys_remove_roles"
	v2KeysRerollKey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
	v2KeysSetPermissions "github.com/unkeyed/unkey/svc/api/routes/v2_keys_set_permissions"
	v2KeysSetRoles "github.com/unkeyed/unkey/svc/api/routes/v2_keys_set_roles"
	v2KeysUpdateCredits "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_credits"
	v2KeysUpdateKey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_key"
	v2KeysVerifyKey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
	v2KeysWhoami "github.com/unkeyed/unkey/svc/api/routes/v2_keys_whoami"

	v2AnalyticsGetVerifications "github.com/unkeyed/unkey/svc/api/routes/v2_analytics_get_verifications"

	v2PortalCreateSession "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
	v2PortalExchangeSession "github.com/unkeyed/unkey/svc/api/routes/v2_portal_exchange_session"

	v2AppsCreateApp "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_app"
	v2AppsDeleteApp "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
	v2AppsGetApp "github.com/unkeyed/unkey/svc/api/routes/v2_apps_get_app"
	v2AppsListApps "github.com/unkeyed/unkey/svc/api/routes/v2_apps_list_apps"
	v2AppsUpdateApp "github.com/unkeyed/unkey/svc/api/routes/v2_apps_update_app"
	v2ProjectsCreateProject "github.com/unkeyed/unkey/svc/api/routes/v2_projects_create_project"
	v2ProjectsDeleteProject "github.com/unkeyed/unkey/svc/api/routes/v2_projects_delete_project"
	v2ProjectsGetProject "github.com/unkeyed/unkey/svc/api/routes/v2_projects_get_project"
	v2ProjectsListProjects "github.com/unkeyed/unkey/svc/api/routes/v2_projects_list_projects"
	v2ProjectsUpdateProject "github.com/unkeyed/unkey/svc/api/routes/v2_projects_update_project"

	zen "github.com/unkeyed/unkey/pkg/zen"
)

// Register wires up all API route handlers with their dependencies and middleware
// chains. This function runs once during server startup; routes cannot be added
// or removed after initialization.
//
// The function applies a default middleware stack to most routes: panic recovery,
// observability (tracing), metrics collection to ClickHouse, structured logging,
// error handling, a one-minute request timeout, and request validation. Internal
// endpoints (pprof) use reduced middleware stacks appropriate to their
// needs.
//
// Conditional routes are registered based on [Services] configuration.
func Register(srv *zen.Server, svc *Services, info zen.InstanceInfo) {
	withObservability := zen.WithObservability()
	withMetrics := zen.WithMetrics(svc.ApiRequests, info)
	withLogging := zen.WithLogging(zen.SkipPaths("/_unkey/internal/", "/health/"))
	withPanicRecovery := zen.WithPanicRecovery()
	withErrorHandling := middleware.WithErrorHandling()
	withValidation := zen.WithValidation(svc.Validator)
	withTimeout := zen.WithTimeout(time.Minute)
	withAuthentication := middleware.WithAuthentication(middleware.AuthenticationConfig{
		Auth:       svc.Auth,
		Database:   svc.Database,
		QuotaCache: svc.Caches.WorkspaceQuota,
		Ratelimit:  svc.Ratelimit,
	})

	publicMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withObservability,
		withMetrics,
		withLogging,
		withErrorHandling,
		withTimeout,
		withValidation,
	}

	protectedMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withObservability,
		withMetrics,
		withLogging,
		withErrorHandling,
		withTimeout,
		withValidation,
		withAuthentication,
	}

	srv.RegisterRoute(publicMiddlewares, &v2Liveness.Handler{})

	// ---------------------------------------------------------------------------
	// pprof (internal profiling endpoints)

	if svc.PprofEnabled {
		pprofMiddlewares := []zen.Middleware{
			withLogging,
			withObservability,
			withPanicRecovery,
			withErrorHandling,
		}

		srv.RegisterRoute(pprofMiddlewares, &pprofRoute.Handler{
			Username: svc.PprofUsername,
			Password: svc.PprofPassword,
			Prefix:   "/debug",
		})
	}

	// ---------------------------------------------------------------------------
	// v2/ratelimit

	// v2/ratelimit.limit
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2RatelimitLimit.Handler{
			DB:              svc.Database,
			RatelimitEvents: svc.RatelimitEvents,
			Ratelimit:       svc.Ratelimit,
			NamespaceCache:  svc.Caches.RatelimitNamespace,
			Auditlogs:       svc.Auditlogs,
			TestMode:        srv.Flags().TestMode,
		},
	)

	// v2/ratelimit.multiLimit
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2RatelimitMultiLimit.Handler{
			DB:              svc.Database,
			RatelimitEvents: svc.RatelimitEvents,
			Ratelimit:       svc.Ratelimit,
			NamespaceCache:  svc.Caches.RatelimitNamespace,
			Auditlogs:       svc.Auditlogs,
			TestMode:        srv.Flags().TestMode,
		},
	)

	// v2/ratelimit.setOverride
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2RatelimitSetOverride.Handler{
			DB:             svc.Database,
			Auditlogs:      svc.Auditlogs,
			NamespaceCache: svc.Caches.RatelimitNamespace,
		},
	)

	// v2/ratelimit.getOverride
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2RatelimitGetOverride.Handler{
			DB:             svc.Database,
			NamespaceCache: svc.Caches.RatelimitNamespace,
		},
	)

	// v2/ratelimit.deleteOverride
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2RatelimitDeleteOverride.Handler{
			DB:             svc.Database,
			Auditlogs:      svc.Auditlogs,
			NamespaceCache: svc.Caches.RatelimitNamespace,
		},
	)

	// v2/ratelimit.listOverrides
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2RatelimitListOverrides.Handler{

			DB: svc.Database,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/identities

	// v2/identities.createIdentity
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2IdentitiesCreateIdentity.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/identities.deleteIdentity
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2IdentitiesDeleteIdentity.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/identities.getIdentity
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2IdentitiesGetIdentity.Handler{

			DB: svc.Database,
		},
	)

	// v2/identities.listIdentities
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2IdentitiesListIdentities.Handler{

			DB: svc.Database,
		},
	)

	// v2/identities.updateIdentity
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2IdentitiesUpdateIdentity.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/apis

	// v2/apis.createApi
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ApisCreateApi.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)
	// v2/apis.getApi
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ApisGetApi.Handler{
			DB:     svc.Database,
			Caches: svc.Caches,
		},
	)

	// v2/apis.listKeys
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ApisListKeys.Handler{
			DB:       svc.Database,
			Vault:    svc.Vault,
			ApiCache: svc.Caches.LiveApiByID,
		},
	)

	// v2/apis.deleteApi
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ApisDeleteApi.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			Caches:    svc.Caches,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/deploy

	// v2/deploy.createDeployment
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2DeployCreateDeployment.Handler{
			DB:         svc.Database,
			CtrlClient: svc.CtrlDeploymentClient,
		},
	)

	// v2/deploy.getDeployment
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2DeployGetDeployment.Handler{

			DB: svc.Database,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/permissions

	// v2/permissions.createPermission
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PermissionsCreatePermission.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/permissions.getPermission
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PermissionsGetPermission.Handler{

			DB: svc.Database,
		},
	)

	// v2/permissions.getRole
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PermissionsGetRole.Handler{

			DB: svc.Database,
		},
	)

	// v2/permissions.listPermissions
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PermissionsListPermissions.Handler{

			DB: svc.Database,
		},
	)

	// v2/permissions.deletePermission
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PermissionsDeletePermission.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/permissions.createRole
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PermissionsCreateRole.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/permissions.listRoles
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PermissionsListRoles.Handler{

			DB: svc.Database,
		},
	)

	// v2/permissions.deleteRole
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PermissionsDeleteRole.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/keys

	// v2/keys.verifyKey
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysVerifyKey.Handler{
			DB:               svc.Database,
			Keys:             svc.Keys,
			Auditlogs:        svc.Auditlogs,
			KeyVerifications: svc.KeyVerifications,
		},
	)

	// v2/keys.migrateKeys
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysMigrateKeys.Handler{

			ApiCache:  svc.Caches.LiveApiByID,
			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/keys.createKey
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysCreateKey.Handler{

			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			Vault:     svc.Vault,
		},
	)

	// v2/keys.rerollKey
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysRerollKey.Handler{

			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			Vault:     svc.Vault,
		},
	)

	// v2/keys.deleteKey
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysDeleteKey.Handler{
			KeyCache: svc.Caches.VerificationKeyByHash,

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/keys.updateKey
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysUpdateKey.Handler{
			DB:           svc.Database,
			Auditlogs:    svc.Auditlogs,
			KeyCache:     svc.Caches.VerificationKeyByHash,
			UsageLimiter: svc.UsageLimiter,
		},
	)

	// v2/keys.getKey
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysGetKey.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			Vault:     svc.Vault,
		},
	)

	// v2/keys.whoami
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysWhoami.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			Vault:     svc.Vault,
		},
	)

	// v2/keys.updateCredits
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysUpdateCredits.Handler{
			DB:           svc.Database,
			Auditlogs:    svc.Auditlogs,
			KeyCache:     svc.Caches.VerificationKeyByHash,
			UsageLimiter: svc.UsageLimiter,
		},
	)

	// v2/keys.setRoles
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysSetRoles.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.setPermissions
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysSetPermissions.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.addPermissions
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysAddPermissions.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.addRoles
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysAddRoles.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.removePermissions
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysRemovePermissions.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.removeRoles
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2KeysRemoveRoles.Handler{

			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/analytics

	// v2/analytics.getVerifications
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2AnalyticsGetVerifications.Handler{
			DB:                         svc.Database,
			AnalyticsConnectionManager: svc.AnalyticsConnectionManager,
			Caches:                     svc.Caches,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/portal

	// v2/portal.createSession
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2PortalCreateSession.Handler{
			DB:            svc.Database,
			Auditlogs:     svc.Auditlogs,
			PortalBaseURL: svc.PortalBaseURL,
		},
	)

	// v2/portal.exchangeSession
	srv.RegisterRoute(
		publicMiddlewares,
		&v2PortalExchangeSession.Handler{
			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/projects.createProject
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ProjectsCreateProject.Handler{
			CtrlClient: svc.CtrlProjectClient,
		},
	)

	// v2/projects.listProjects
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ProjectsListProjects.Handler{
			DB: svc.Database,
		},
	)

	// v2/projects.getProject
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ProjectsGetProject.Handler{
			DB: svc.Database,
		},
	)

	// v2/projects.updateProject
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ProjectsUpdateProject.Handler{
			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/projects.deleteProject
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2ProjectsDeleteProject.Handler{
			DB:         svc.Database,
			CtrlClient: svc.CtrlProjectClient,
		},
	)

	// v2/apps.createApp
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2AppsCreateApp.Handler{
			DB:         svc.Database,
			CtrlClient: svc.CtrlAppClient,
		},
	)

	// v2/apps.getApp
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2AppsGetApp.Handler{
			DB: svc.Database,
		},
	)

	// v2/apps.listApps
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2AppsListApps.Handler{
			DB: svc.Database,
		},
	)

	// v2/apps.updateApp
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2AppsUpdateApp.Handler{
			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/apps.deleteApp
	srv.RegisterRoute(
		protectedMiddlewares,
		&v2AppsDeleteApp.Handler{
			DB:         svc.Database,
			CtrlClient: svc.CtrlAppClient,
		},
	)

	// ---------------------------------------------------------------------------
	// misc

	miscMiddlewares := []zen.Middleware{
		withObservability,
		withMetrics,
		withLogging,
		withPanicRecovery,
		withErrorHandling,
	}

	srv.RegisterRoute(
		miscMiddlewares,
		&reference.Handler{},
	)

	srv.RegisterRoute(
		miscMiddlewares,
		&openapi.Handler{},
	)
}
