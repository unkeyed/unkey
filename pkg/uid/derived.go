package uid

import (
	"crypto/sha256"
	"strings"

	"github.com/unkeyed/unkey/pkg/base58"
)

// derivedBytes is how much of the digest feeds the identifier. Sixteen bytes
// encode to 22 base58 characters, so a prefixed deployment id is 24 characters
// and fits the varchar(48) id columns. Sixteen bytes also keep a collision out
// of reach inside the single workspace a scope covers.
const derivedBytes = 16

// Derived builds a deterministic identifier from scope parts. A caller that
// repeats the same request computes the same id and lands on the same Restate
// virtual object. The parts are joined with a separator no id can contain, so
// order and boundaries matter: ("a", "bc") and ("ab", "c") differ.
//
// Always put the tenant in the scope. Without it, two workspaces that send the
// same idempotency key derive one id and collide across the tenant boundary.
//
// The result is a hash of values the caller already knows, so never use it
// where an id has to be unguessable.
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
