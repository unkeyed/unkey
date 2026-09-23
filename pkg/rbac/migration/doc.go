// Package migration translates legacy root-key permissions to canonical URN
// permissions without database access or writes. Callers resolve API IDs to
// keyspaces, and namespace, app, environment, and identity IDs to their
// project-scoped names through [Scope]. Project IDs are preserved directly.
//
// [Translate] maps legacy actions to the canonical catalog, including create
// and update actions to write. This intentionally merges capabilities: a
// create_key grant becomes key write, which also permits updates. Translation
// is not authorization or proof of permission containment. Callers must supply
// trusted ownership metadata and authorize the resulting grants separately.
// Translation is one-way and never emits legacy permissions.
//
// The legacy catch-all * returns [ErrUnsupported]: canonical global admin
// would add portal session minting authority. Legacy portal tuples also have
// no translation here. RBAC and workspace operations use the legacy wildcard
// ID; concrete IDs in those categories have no defined mapping and are rejected.
//
// The production fixture contains permission strings, not resource ownership.
// Its tests use synthetic ownership and explicit expected outcomes, including
// rejection. Passing them does not mean every permission can be migrated.
package migration
