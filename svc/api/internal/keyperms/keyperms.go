// Package keyperms holds the authorization requirements for acting on the keys
// of a single keyspace, expressed as reusable rbac queries.
//
// The operator routes under svc/api/routes used to spell these requirements out
// inline. The portal routes need the *same* requirements as their authorization
// ceiling: a portal may never let a caller do something the equivalent operator
// route would have refused. Keeping one copy here is what makes that guarantee
// checkable, instead of two hand-written query trees that can drift apart.
//
// Every query here is built from api-scoped legacy tuples only. No builder emits
// a URN leaf, deliberately: a separate migration moves all of authorization to
// URN-only, and half-converting it from here would leave call sites in two
// different worlds. Routes that additionally accept a URN grant today keep that
// arm at the call site and Or it with the builder's output.
package keyperms

import "github.com/unkeyed/unkey/pkg/rbac"

// Scope identifies the keyspace a requirement applies to.
//
// WorkspaceID and KeyspaceID are carried even though the legacy tuples below
// only need APIID. They are the inputs the URN leaves will need once the
// URN-only migration lands, and threading them through now means that migration
// touches this package rather than every call site.
type Scope struct {
	// WorkspaceID owns both the keyspace and the api.
	WorkspaceID string

	// KeyspaceID is the key_auth id the request acts on.
	KeyspaceID string

	// APIID is the api that owns KeyspaceID. apis.key_auth_id is unique, so a
	// keyspace resolves to at most one api.
	APIID string
}

// ReadKeys is the requirement for enumerating the keys of a keyspace.
//
// It is a conjunction on purpose: listing keys exposes both key material
// metadata and the shape of the api, so a caller needs read_key *and* read_api.
// Requiring only read_key would be strictly weaker than the operator route.
func ReadKeys(scope Scope) rbac.PermissionQuery {
	return rbac.And(
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.ReadKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   scope.APIID,
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
				ResourceID:   scope.APIID,
				Action:       rbac.ReadAPI,
			}),
		),
	)
}

// CreateKey is the requirement for minting a key in a keyspace. Rerolling is a
// create, not an update, so it shares this requirement.
func CreateKey(scope Scope) rbac.PermissionQuery {
	return rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   scope.APIID,
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
	)
}

// EncryptKey is the additional requirement for handling recoverable key
// material. It stays a separate query rather than being folded into CreateKey
// because callers decide independently whether encryption is in play: the
// operator route looks at the individual key, while a keyspace-wide caller
// looks at key_auth.store_encrypted_keys.
func EncryptKey(scope Scope) rbac.PermissionQuery {
	return rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   scope.APIID,
			Action:       rbac.EncryptKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.EncryptKey,
		}),
	)
}

// ReadAnalytics is the requirement for reading verification analytics for a
// keyspace. It mirrors what the verifications route matches today: the api
// wildcard or the specific api.
func ReadAnalytics(scope Scope) rbac.PermissionQuery {
	return rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.ReadAnalytics,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   scope.APIID,
			Action:       rbac.ReadAnalytics,
		}),
	)
}
