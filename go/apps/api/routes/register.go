package routes

import (
	"github.com/unkeyed/unkey/go/apps/api/routes/reference"
	v2Liveness "github.com/unkeyed/unkey/go/apps/api/routes/v2_liveness"

	v2RatelimitDeleteOverride "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_delete_override"
	v2RatelimitGetOverride "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_get_override"
	v2RatelimitLimit "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	v2RatelimitListOverrides "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_list_overrides"
	v2RatelimitSetOverride "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_set_override"

	v2ApisCreateApi "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_create_api"
	v2ApisDeleteApi "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_delete_api"
	v2ApisGetApi "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_get_api"
	v2ApisListKeys "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_list_keys"

	v2IdentitiesCreateIdentity "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	v2IdentitiesDeleteIdentity "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	v2IdentitiesGetIdentity "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity"
	v2IdentitiesListIdentities "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_list_identities"
	v2IdentitiesUpdateIdentity "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity"

	v2PermissionsCreatePermission "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_permission"
	v2PermissionsCreateRole "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_role"
	v2PermissionsDeletePermission "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_permission"
	v2PermissionsDeleteRole "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_role"
	v2PermissionsGetPermission "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_permission"
	v2PermissionsGetRole "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_role"
	v2PermissionsListPermissions "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_permissions"
	v2PermissionsListRoles "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_roles"

	zen "github.com/unkeyed/unkey/go/pkg/zen"
)

// here we register all of the routes.
// this function runs during startup.
func Register(srv *zen.Server, svc *Services) {
	withTracing := zen.WithTracing()
	withMetrics := zen.WithMetrics(svc.ClickHouse)

	withLogging := zen.WithLogging(svc.Logger)
	withErrorHandling := zen.WithErrorHandling(svc.Logger)
	withValidation := zen.WithValidation(svc.Validator)

	defaultMiddlewares := []zen.Middleware{
		withTracing,
		withMetrics,
		withLogging,
		withErrorHandling,
		withValidation,
	}

	srv.RegisterRoute(defaultMiddlewares, v2Liveness.New())

	// ---------------------------------------------------------------------------
	// v2/ratelimit

	// v2/ratelimit.limit
	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitLimit.New(v2RatelimitLimit.Services{
			Logger:                        svc.Logger,
			DB:                            svc.Database,
			Keys:                          svc.Keys,
			ClickHouse:                    svc.ClickHouse,
			Ratelimit:                     svc.Ratelimit,
			Permissions:                   svc.Permissions,
			RatelimitNamespaceByNameCache: svc.Caches.RatelimitNamespaceByName,
			RatelimitOverrideMatchesCache: svc.Caches.RatelimitOverridesMatch,
			TestMode:                      srv.Flags().TestMode,
		}),
	)

	// v2/ratelimit.setOverride
	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitSetOverride.New(v2RatelimitSetOverride.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// v2/ratelimit.getOverride
	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitGetOverride.New(v2RatelimitGetOverride.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// v2/ratelimit.deleteOverride
	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitDeleteOverride.New(v2RatelimitDeleteOverride.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// v2/ratelimit.listOverrides
	srv.RegisterRoute(
		defaultMiddlewares,
		v2RatelimitListOverrides.New(v2RatelimitListOverrides.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// ---------------------------------------------------------------------------
	// v2/identities

	// v2/identities.createIdentity
	srv.RegisterRoute(
		defaultMiddlewares,
		v2IdentitiesCreateIdentity.New(v2IdentitiesCreateIdentity.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// v2/identities.deleteIdentity
	srv.RegisterRoute(
		defaultMiddlewares,
		v2IdentitiesDeleteIdentity.New(v2IdentitiesDeleteIdentity.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// v2/identities.getIdentity
	srv.RegisterRoute(
		defaultMiddlewares,
		v2IdentitiesGetIdentity.New(v2IdentitiesGetIdentity.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// v2/identities.listIdentities
	srv.RegisterRoute(
		defaultMiddlewares,
		v2IdentitiesListIdentities.New(v2IdentitiesListIdentities.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// v2/identities.updateIdentity
	srv.RegisterRoute(
		defaultMiddlewares,
		v2IdentitiesUpdateIdentity.New(v2IdentitiesUpdateIdentity.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// ---------------------------------------------------------------------------
	// v2/apis

	// v2/apis.createApi
	srv.RegisterRoute(
		defaultMiddlewares,
		v2ApisCreateApi.New(v2ApisCreateApi.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)
	// v2/apis.getApi
	srv.RegisterRoute(
		defaultMiddlewares,
		v2ApisGetApi.New(v2ApisGetApi.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// v2/apis.listKeys
	srv.RegisterRoute(
		defaultMiddlewares,
		v2ApisListKeys.New(v2ApisListKeys.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Vault:       svc.Vault,
		}),
	)

	// v2/apis.deleteApi
	srv.RegisterRoute(
		defaultMiddlewares,
		v2ApisDeleteApi.New(v2ApisDeleteApi.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
			Caches:      svc.Caches,
		}),
	)

	// ---------------------------------------------------------------------------
	// v2/permissions

	// v2/permissions.createPermission
	srv.RegisterRoute(
		defaultMiddlewares,
		v2PermissionsCreatePermission.New(v2PermissionsCreatePermission.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// v2/permissions.getPermission
	srv.RegisterRoute(
		defaultMiddlewares,
		v2PermissionsGetPermission.New(v2PermissionsGetPermission.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// v2/permissions.getRole
	srv.RegisterRoute(
		defaultMiddlewares,
		v2PermissionsGetRole.New(v2PermissionsGetRole.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// v2/permissions.listPermissions
	srv.RegisterRoute(
		defaultMiddlewares,
		v2PermissionsListPermissions.New(v2PermissionsListPermissions.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// v2/permissions.deletePermission
	srv.RegisterRoute(
		defaultMiddlewares,
		v2PermissionsDeletePermission.New(v2PermissionsDeletePermission.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// v2/permissions.createRole
	srv.RegisterRoute(
		defaultMiddlewares,
		v2PermissionsCreateRole.New(v2PermissionsCreateRole.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// v2/permissions.listRoles
	srv.RegisterRoute(
		defaultMiddlewares,
		v2PermissionsListRoles.New(v2PermissionsListRoles.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
		}),
	)

	// v2/permissions.deleteRole
	srv.RegisterRoute(
		defaultMiddlewares,
		v2PermissionsDeleteRole.New(v2PermissionsDeleteRole.Services{
			Logger:      svc.Logger,
			DB:          svc.Database,
			Keys:        svc.Keys,
			Permissions: svc.Permissions,
			Auditlogs:   svc.Auditlogs,
		}),
	)

	// ---------------------------------------------------------------------------
	// misc

	srv.RegisterRoute([]zen.Middleware{
		withTracing,
		withMetrics,
		withLogging,
		withErrorHandling,
	}, reference.New())

}
