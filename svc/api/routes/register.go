package routes

import (
	"time"

	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	openapi "github.com/unkeyed/unkey/svc/api/routes/openapi"
	"github.com/unkeyed/unkey/svc/api/routes/reference"
	v2Liveness "github.com/unkeyed/unkey/svc/api/routes/v2_liveness"

	chproxyMetrics "github.com/unkeyed/unkey/svc/api/routes/chproxy_metrics"
	chproxyRatelimits "github.com/unkeyed/unkey/svc/api/routes/chproxy_ratelimits"
	chproxyVerifications "github.com/unkeyed/unkey/svc/api/routes/chproxy_verifications"

	pprofRoute "github.com/unkeyed/unkey/svc/api/routes/pprof"

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
	v2DeployGenerateUploadUrl "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_generate_upload_url"
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

	zen "github.com/unkeyed/unkey/pkg/zen"
)

// Register wires up all API route handlers with their dependencies and middleware
// chains. This function runs once during server startup; routes cannot be added
// or removed after initialization.
//
// The function applies a default middleware stack to most routes: panic recovery,
// observability (tracing), metrics collection to ClickHouse, structured logging,
// error handling, a one-minute request timeout, and request validation. Internal
// endpoints (chproxy, pprof) use reduced middleware stacks appropriate to their
// needs.
//
// Conditional routes are registered based on [Services] configuration. Chproxy
// endpoints require a non-empty ChproxyToken, and pprof endpoints require
// PprofEnabled to be true.
func Register(srv *zen.Server, svc *Services, info zen.InstanceInfo) {
	withObservability := zen.WithObservability()
	withMetrics := zen.WithMetrics(svc.ClickHouse, info)
	withLogging := zen.WithLogging(svc.Logger)
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withErrorHandling := middleware.WithErrorHandling(svc.Logger)
	withValidation := zen.WithValidation(svc.Validator)
	withTimeout := zen.WithTimeout(time.Minute)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withObservability,
		withMetrics,
		withLogging,
		withErrorHandling,
		withTimeout,
		withValidation,
	}

	srv.RegisterRoute(defaultMiddlewares, &v2Liveness.Handler{})

	// ---------------------------------------------------------------------------
	// chproxy (internal endpoints)

	if svc.ChproxyToken != "" {
		chproxyMiddlewares := []zen.Middleware{
			withMetrics,
			withLogging,
			withObservability,
			withPanicRecovery,
			withErrorHandling,
		}

		// chproxy/verifications - internal endpoint for key verification events
		srv.RegisterRoute(chproxyMiddlewares, &chproxyVerifications.Handler{
			ClickHouse: svc.ClickHouse,
			Logger:     svc.Logger,
			Token:      svc.ChproxyToken,
		})

		// chproxy/metrics - internal endpoint for API request metrics
		srv.RegisterRoute(chproxyMiddlewares, &chproxyMetrics.Handler{
			ClickHouse: svc.ClickHouse,
			Logger:     svc.Logger,
			Token:      svc.ChproxyToken,
		})

		// chproxy/ratelimits - internal endpoint for ratelimit events
		srv.RegisterRoute(chproxyMiddlewares, &chproxyRatelimits.Handler{
			ClickHouse: svc.ClickHouse,
			Logger:     svc.Logger,
			Token:      svc.ChproxyToken,
		})
	}

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
			Logger:   svc.Logger,
			Username: svc.PprofUsername,
			Password: svc.PprofPassword,
		})
	}

	// ---------------------------------------------------------------------------
	// v2/ratelimit

	// v2/ratelimit.limit
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2RatelimitLimit.Handler{
			Logger:                  svc.Logger,
			DB:                      svc.Database,
			Keys:                    svc.Keys,
			ClickHouse:              svc.ClickHouse,
			Ratelimit:               svc.Ratelimit,
			RatelimitNamespaceCache: svc.Caches.RatelimitNamespace,
			TestMode:                srv.Flags().TestMode,
			Auditlogs:               svc.Auditlogs,
		},
	)

	// v2/ratelimit.multiLimit
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2RatelimitMultiLimit.Handler{
			Logger:                  svc.Logger,
			DB:                      svc.Database,
			Keys:                    svc.Keys,
			ClickHouse:              svc.ClickHouse,
			Ratelimit:               svc.Ratelimit,
			RatelimitNamespaceCache: svc.Caches.RatelimitNamespace,
			TestMode:                srv.Flags().TestMode,
			Auditlogs:               svc.Auditlogs,
		},
	)

	// v2/ratelimit.setOverride
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2RatelimitSetOverride.Handler{
			Logger:                  svc.Logger,
			DB:                      svc.Database,
			Keys:                    svc.Keys,
			Auditlogs:               svc.Auditlogs,
			RatelimitNamespaceCache: svc.Caches.RatelimitNamespace,
		},
	)

	// v2/ratelimit.getOverride
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2RatelimitGetOverride.Handler{
			Logger:                  svc.Logger,
			DB:                      svc.Database,
			Keys:                    svc.Keys,
			RatelimitNamespaceCache: svc.Caches.RatelimitNamespace,
		},
	)

	// v2/ratelimit.deleteOverride
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2RatelimitDeleteOverride.Handler{
			Logger:                  svc.Logger,
			DB:                      svc.Database,
			Keys:                    svc.Keys,
			Auditlogs:               svc.Auditlogs,
			RatelimitNamespaceCache: svc.Caches.RatelimitNamespace,
		},
	)

	// v2/ratelimit.listOverrides
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2RatelimitListOverrides.Handler{
			Logger: svc.Logger,
			DB:     svc.Database,
			Keys:   svc.Keys,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/identities

	// v2/identities.createIdentity
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2IdentitiesCreateIdentity.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/identities.deleteIdentity
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2IdentitiesDeleteIdentity.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/identities.getIdentity
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2IdentitiesGetIdentity.Handler{
			Logger: svc.Logger,
			DB:     svc.Database,
			Keys:   svc.Keys,
		},
	)

	// v2/identities.listIdentities
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2IdentitiesListIdentities.Handler{
			Logger: svc.Logger,
			DB:     svc.Database,
			Keys:   svc.Keys,
		},
	)

	// v2/identities.updateIdentity
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2IdentitiesUpdateIdentity.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/apis

	// v2/apis.createApi
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2ApisCreateApi.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)
	// v2/apis.getApi
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2ApisGetApi.Handler{
			Logger: svc.Logger,
			DB:     svc.Database,
			Keys:   svc.Keys,
			Caches: svc.Caches,
		},
	)

	// v2/apis.listKeys
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2ApisListKeys.Handler{
			Logger:   svc.Logger,
			DB:       svc.Database,
			Keys:     svc.Keys,
			Vault:    svc.Vault,
			ApiCache: svc.Caches.LiveApiByID,
		},
	)

	// v2/apis.deleteApi
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2ApisDeleteApi.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			Caches:    svc.Caches,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/deploy

	// v2/deploy.createDeployment
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2DeployCreateDeployment.Handler{
			Logger:     svc.Logger,
			DB:         svc.Database,
			Keys:       svc.Keys,
			CtrlClient: svc.CtrlDeploymentClient,
		},
	)

	// v2/deploy.getDeployment
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2DeployGetDeployment.Handler{
			Logger:     svc.Logger,
			DB:         svc.Database,
			Keys:       svc.Keys,
			CtrlClient: svc.CtrlDeploymentClient,
		},
	)

	// v2/deploy.generateUploadUrl
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2DeployGenerateUploadUrl.Handler{
			Logger:     svc.Logger,
			DB:         svc.Database,
			Keys:       svc.Keys,
			CtrlClient: svc.CtrlDeploymentClient,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/permissions

	// v2/permissions.createPermission
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2PermissionsCreatePermission.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/permissions.getPermission
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2PermissionsGetPermission.Handler{
			Logger: svc.Logger,
			DB:     svc.Database,
			Keys:   svc.Keys,
		},
	)

	// v2/permissions.getRole
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2PermissionsGetRole.Handler{
			Logger: svc.Logger,
			DB:     svc.Database,
			Keys:   svc.Keys,
		},
	)

	// v2/permissions.listPermissions
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2PermissionsListPermissions.Handler{
			Logger: svc.Logger,
			DB:     svc.Database,
			Keys:   svc.Keys,
		},
	)

	// v2/permissions.deletePermission
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2PermissionsDeletePermission.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/permissions.createRole
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2PermissionsCreateRole.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/permissions.listRoles
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2PermissionsListRoles.Handler{
			Logger: svc.Logger,
			DB:     svc.Database,
			Keys:   svc.Keys,
		},
	)

	// v2/permissions.deleteRole
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2PermissionsDeleteRole.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/keys

	// v2/keys.verifyKey
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysVerifyKey.Handler{
			Logger:     svc.Logger,
			ClickHouse: svc.ClickHouse,
			DB:         svc.Database,
			Keys:       svc.Keys,
			Auditlogs:  svc.Auditlogs,
		},
	)

	// v2/keys.migrateKeys
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysMigrateKeys.Handler{
			Logger:    svc.Logger,
			ApiCache:  svc.Caches.LiveApiByID,
			DB:        svc.Database,
			Auditlogs: svc.Auditlogs,
			Keys:      svc.Keys,
		},
	)

	// v2/keys.createKey
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysCreateKey.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			Vault:     svc.Vault,
		},
	)

	// v2/keys.rerollKey
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysRerollKey.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			Vault:     svc.Vault,
		},
	)

	// v2/keys.deleteKey
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysDeleteKey.Handler{
			KeyCache:  svc.Caches.VerificationKeyByHash,
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
		},
	)

	// v2/keys.updateKey
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysUpdateKey.Handler{
			Logger:       svc.Logger,
			DB:           svc.Database,
			Keys:         svc.Keys,
			Auditlogs:    svc.Auditlogs,
			KeyCache:     svc.Caches.VerificationKeyByHash,
			UsageLimiter: svc.UsageLimiter,
		},
	)

	// v2/keys.getKey
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysGetKey.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			Vault:     svc.Vault,
		},
	)

	// v2/keys.whoami
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysWhoami.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			Vault:     svc.Vault,
		},
	)

	// v2/keys.updateCredits
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysUpdateCredits.Handler{
			Logger:       svc.Logger,
			DB:           svc.Database,
			Keys:         svc.Keys,
			Auditlogs:    svc.Auditlogs,
			KeyCache:     svc.Caches.VerificationKeyByHash,
			UsageLimiter: svc.UsageLimiter,
		},
	)

	// v2/keys.setRoles
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysSetRoles.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.setPermissions
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysSetPermissions.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.addPermissions
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysAddPermissions.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.addRoles
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysAddRoles.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.removePermissions
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysRemovePermissions.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// v2/keys.removeRoles
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2KeysRemoveRoles.Handler{
			Logger:    svc.Logger,
			DB:        svc.Database,
			Keys:      svc.Keys,
			Auditlogs: svc.Auditlogs,
			KeyCache:  svc.Caches.VerificationKeyByHash,
		},
	)

	// ---------------------------------------------------------------------------
	// v2/analytics

	// v2/analytics.getVerifications
	srv.RegisterRoute(
		defaultMiddlewares,
		&v2AnalyticsGetVerifications.Handler{
			Logger:                     svc.Logger,
			DB:                         svc.Database,
			Keys:                       svc.Keys,
			ClickHouse:                 svc.ClickHouse,
			AnalyticsConnectionManager: svc.AnalyticsConnectionManager,
			Caches:                     svc.Caches,
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
		&reference.Handler{
			Logger: svc.Logger,
		},
	)

	srv.RegisterRoute(
		miscMiddlewares,
		&openapi.Handler{
			Logger: svc.Logger,
		},
	)
}
