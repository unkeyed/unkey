package handler

import "github.com/unkeyed/unkey/pkg/rbac"

// CreateKeyPermissions is the api-scoped legacy-tuple requirement for minting a
// key in a keyspace. Rerolling is a create, not an update, so this route shares
// it.
//
// It lives here, next to the handler that enforces it, and is exported because
// the portal session route needs the same requirement as its authorization
// ceiling: a portal may never let an end user do something this route would have
// refused. One definition owned by the enforcing route is what makes that
// checkable, rather than two hand-written query trees that can drift apart.
//
// No URN leaf is emitted here. This route Ors its own URN arm on top, because a
// separate migration moves all of authorization to URN-only and half-converting
// it from here would leave call sites in two different worlds.
func CreateKeyPermissions(apiID string) rbac.PermissionQuery {
	return rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   apiID,
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
	)
}

// EncryptKeyPermissions is the additional api-scoped legacy-tuple requirement
// for handling recoverable key material.
//
// It stays separate from CreateKeyPermissions because callers decide
// independently whether encryption is in play: this route looks at the
// individual key's encryption row, while a keyspace-wide caller such as portal
// session minting looks at key_auth.store_encrypted_keys.
func EncryptKeyPermissions(apiID string) rbac.PermissionQuery {
	return rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   apiID,
			Action:       rbac.EncryptKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.EncryptKey,
		}),
	)
}
