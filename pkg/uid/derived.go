package uid

import (
	"crypto/sha256"
	"strings"

	"github.com/unkeyed/unkey/pkg/base58"
)

// derivedBytes is how much of the digest feeds the identifier. Sixteen bytes
// encode to 22 base58 characters, so a derived deployment id is 24 characters
// including its prefix and fits the varchar(48) id columns. It also leaves the
// collision probability far below the point where a tenant could reach it: the
// scope already contains the workspace, so a collision would have to happen
// inside one customer's own keys.
const derivedBytes = 16

// Derived builds a deterministic identifier from scope parts, so a caller that
// repeats the same request computes the same id and lands on the same Restate
// virtual object. Order matters and the parts are joined with a separator that
// cannot appear inside an id, so ("a", "bc") and ("ab", "c") differ.
//
// Always include the tenant in the scope. Without it two workspaces sending the
// same idempotency key would derive one id and collide across the tenant
// boundary.
//
// This is a hash, not a secret: it is derived from values the caller already
// knows, so it must never be used where an id has to be unguessable.
func Derived(prefix Prefix, scope ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(scope, "\x00")))
	encoded := base58.Encode(digest[:derivedBytes])

	if prefix == "" {
		return encoded
	}

	var id strings.Builder
	id.Grow(len(prefix) + 1 + len(encoded))
	id.WriteString(string(prefix))
	id.WriteByte('_')
	id.WriteString(encoded)
	return id.String()
}
