package uid

import (
	"crypto/sha256"
	"strings"
)

const derivedLength = 22

// Derived generates a deterministic prefixed identifier from the given parts.
// The same parts always produce the same identifier. Use it to make an insert
// idempotent: derive the id from the caller's idempotency key and let the
// unique id reject the replay.
func Derived(prefix Prefix, parts ...string) string {
	// The separator prevents part-boundary ambiguity: ("ab","c") and
	// ("a","bc") must not hash to the same input.
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))

	var id strings.Builder
	if prefix != "" {
		id.Grow(len(prefix) + 1 + derivedLength)
		id.WriteString(string(prefix))
		id.WriteByte('_')
	} else {
		id.Grow(derivedLength)
	}

	written := 0
	for written < derivedLength {
		for _, b := range hash {
			if int(b) >= secureAlphabetMax {
				continue
			}

			id.WriteByte(defaultAlphabet[int(b)%len(defaultAlphabet)])
			written++
			if written == derivedLength {
				break
			}
		}

		if written < derivedLength {
			hash = sha256.Sum256(hash[:])
		}
	}

	return id.String()
}
