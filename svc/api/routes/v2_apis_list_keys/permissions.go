package handler

import "github.com/unkeyed/unkey/pkg/rbac"

// ReadKeysPermissions is the api-scoped legacy-tuple requirement for
// enumerating the keys of a keyspace.
//
// It lives here, next to the handler that enforces it, and is exported because
// the portal session route needs the same requirement as its authorization
// ceiling: a portal may never let an end user do something this route would have
// refused. One definition owned by the enforcing route is what makes that
// checkable, rather than two hand-written query trees that can drift apart.
//
// The conjunction is deliberate. Listing keys exposes both key material metadata
// and the shape of the api, so a caller needs read_key and read_api. Requiring
// only read_key would be strictly weaker than this route.
//
// No URN leaf is emitted here. This route Ors its own URN arm on top, because a
// separate migration moves all of authorization to URN-only and half-converting
// it from here would leave call sites in two different worlds.
func ReadKeysPermissions(apiID string) rbac.PermissionQuery {
	return rbac.And(
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.ReadKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   apiID,
				Action:       rbac.ReadKey,
			}),
		),
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.ReadAPI,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   apiID,
				Action:       rbac.ReadAPI,
			}),
		),
	)
}
