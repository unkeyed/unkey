package uid

import (
	"math/rand/v2"
	"strings"
)

// nanoAlphabet defines the character set used for generating nano IDs.
// Uses lowercase letters and digits for URL-safe, case-insensitive identifiers.
const nanoAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// Nano generates a simple random alphanumeric string.
//
// Unlike the main UID generation functions in this package, Nano creates
// shorter, non-timestamped identifiers suitable for cases where you need
// simple, random strings without chronological ordering guarantees or prefixes.
//
// The generated string consists only of random alphanumeric characters
// (lowercase letters and digits). The default length is 8 characters,
// but this can be overridden by providing a custom length parameter.
//
// SECURITY WARNING: This function uses math/rand/v2 and is NOT cryptographically
// secure. The random values are predictable and MUST NOT be used for:
//   - API keys or access tokens
//   - Session identifiers
//   - Password reset tokens
//   - Any externally-exposed identifiers
//   - Any security-sensitive use cases
//
// Nano is intended ONLY for internal identifiers, test fixtures, and other
// non-sensitive uses where predictability is acceptable.
//
// Note: math/rand/v2 automatically seeds the global random source with a
// cryptographically secure random value, so manual seeding is not required.
//
// Example usage:
//
//	// Generate with default 8 characters
//	id := uid.Nano()  // e.g., "k3n5p8x2"
//
//	// Generate with custom 12 characters
//	id := uid.Nano(12)  // e.g., "a9k2n5p8x3m7"
//
//	// Use with a prefix manually
//	id := "usr_" + uid.Nano()  // e.g., "usr_k3n5p8x2"
//
// For production use cases requiring cryptographic security, use [New] which
// provides cryptographically secure random generation via crypto/rand.
func Nano(length ...int) string {
	// Default to 8 characters if no length specified
	n := 8
	if len(length) > 0 {
		n = length[0]
	}

	// Pre-allocate builder for efficiency
	// We use strings.Builder to avoid repeated string concatenations
	// which would create O(n) intermediate strings
	var b strings.Builder
	b.Grow(n)

	for i := 0; i < n; i++ {
		b.WriteByte(nanoAlphabet[rand.IntN(len(nanoAlphabet))])
	}

	return b.String()
}

func NanoLower(length ...int) string {
	return strings.ToLower(Nano(length...))
}
