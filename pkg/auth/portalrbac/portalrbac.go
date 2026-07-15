// Package portalrbac is the single translation seam between the portal's
// simplified, public-facing permission model and the RBAC mechanism that
// actually enforces access.
//
// A workspace owner creates a portal session with a small, stable vocabulary of
// capabilities ("keys:read", "keys:reroll", ...) scoped to a set of keyspaces.
// That simplified grant is what gets persisted on the session and what the
// public API exposes. Enforcement, however, currently reuses the existing
// keyspace-scoped URN permissions checked by the shared key handlers.
//
// Keeping the mapping here — invoked once, when a session principal is resolved
// — means:
//
//   - The persisted/public form never leaks the enforcement representation, so
//     the mapping can change without migrating stored sessions.
//   - Handlers stay ignorant of "portal": they keep authorizing against ordinary
//     RBAC permission strings.
//   - To later run a bespoke portal RBAC instead of the shared URN system, only
//     this package changes (swap [Expand] for an alternative [Authorizer], and
//     have the portal route wrappers consult it directly).
package portalrbac

import (
	"fmt"

	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// Capability is one entry in the portal's simplified permission vocabulary. It
// is intentionally coarse and product-oriented; the fine-grained RBAC actions it
// maps to are an implementation detail owned by this package.
type Capability string

const (
	// CapKeysRead lets the end user list and read their own keys.
	CapKeysRead Capability = "keys:read"

	// CapKeysCreate lets the end user create new keys.
	CapKeysCreate Capability = "keys:create"

	// CapKeysReroll lets the end user rotate the secret of an existing key.
	CapKeysReroll Capability = "keys:reroll"

	// CapAnalyticsRead lets the end user read verification analytics.
	CapAnalyticsRead Capability = "analytics:read"
)

// Parse validates a raw capability string and returns the typed Capability.
// Callers should use this at the API boundary (portal.createSession) to reject
// unknown verbs before they are ever persisted.
func Parse(s string) (Capability, error) {
	switch Capability(s) {
	case CapKeysRead, CapKeysCreate, CapKeysReroll, CapAnalyticsRead:
		return Capability(s), nil
	default:
		return "", fmt.Errorf("unknown portal capability %q", s)
	}
}

// ParseAll validates a slice of raw capability strings, returning the first
// error encountered.
func ParseAll(raw []string) ([]Capability, error) {
	caps := make([]Capability, 0, len(raw))
	for _, s := range raw {
		c, err := Parse(s)
		if err != nil {
			return nil, err
		}
		caps = append(caps, c)
	}
	return caps, nil
}

// Grant is the simplified authorization a portal session carries: a set of
// capabilities scoped to a set of keyspaces within one workspace.
type Grant struct {
	WorkspaceID  string
	KeyspaceIDs  []string
	Capabilities []Capability
}

// Expand translates a Grant into the concrete RBAC permission strings the shared
// key/analytics handlers authorize against. The strings are built with the same
// urn + permission primitives the handlers use in their rbac.U/rbac.T queries,
// so they match by construction (see portalrbac_test.go, which checks Expand's
// output against the real handler queries).
//
// Key capabilities are keyspace-scoped: each is expanded once per keyspace in
// the grant. Analytics is workspace-wide and expands to a single wildcard tuple.
func (g Grant) Expand() []string {
	var granted []string

	for _, c := range g.Capabilities {
		switch c {
		case CapKeysRead:
			// listKeys requires read_key on the keyspace's keys AND read_keyspace
			// on the keyspace itself.
			for _, ks := range g.KeyspaceIDs {
				granted = append(granted,
					unkeyPerm(urn.New().Workspace(g.WorkspaceID).Keyspace(ks).Key("*"), permissions.ReadKey{}),
					unkeyPerm(urn.New().Workspace(g.WorkspaceID).Keyspace(ks), permissions.ReadKeyspace{}),
				)
			}

		case CapKeysCreate:
			for _, ks := range g.KeyspaceIDs {
				granted = append(granted,
					unkeyPerm(urn.New().Workspace(g.WorkspaceID).Keyspace(ks), permissions.CreateKey{}),
				)
			}

		case CapKeysReroll:
			// Reroll is authorized as create_key today (it mints a fresh key), plus
			// encrypt_key so recoverable keys can be rotated. When reroll gets its
			// own RBAC action, only this arm changes.
			for _, ks := range g.KeyspaceIDs {
				granted = append(granted,
					unkeyPerm(urn.New().Workspace(g.WorkspaceID).Keyspace(ks), permissions.CreateKey{}),
					unkeyPerm(urn.New().Workspace(g.WorkspaceID).Keyspace(ks).Key("*"), permissions.EncryptKey{}),
				)
			}

		case CapAnalyticsRead:
			// Analytics is not keyspace-scoped; the portal getVerifications handler
			// checks the wildcard api tuple.
			granted = append(granted, rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.ReadAnalytics,
			}.String())
		}
	}

	return granted
}

// unkeyPerm renders a keyspace/key-scoped URN permission string in the exact
// format rbac.U produces, so granted strings match the handler queries.
func unkeyPerm(resource fmt.Stringer, action fmt.Stringer) string {
	return fmt.Sprintf("%s#%s", resource.String(), action.String())
}
