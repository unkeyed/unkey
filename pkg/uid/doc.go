// Package uid generates prefixed random identifiers for Unkey resources.
//
// The package provides three generation functions: [DNS1035] and [New] for
// fast, non-secure identifiers, and [Secure] for cryptographically secure
// identifiers. Use [ToDNS1035] and [FromDNS1035] to convert between formats.
//
// # Security
//
// [DNS1035] and [New] use math/rand/v2 which is NOT cryptographically secure.
// Generated identifiers are predictable. Use [Secure] for verification tokens,
// API keys, or any security-sensitive purposes.
//
// # Usage
//
// Generate a DNS-1035 compliant label:
//
//	label := uid.DNS1035()      // "k3n5p8x2" (8 chars, starts with letter)
//	label := uid.DNS1035(12)    // "a9k2n5p8x3m7"
//
// Generate a prefixed identifier:
//
//	id := uid.New(uid.KeyPrefix)      // "key_k3n5p8x2"
//	id := uid.New(uid.APIPrefix, 12)  // "api_a9k2n5p8x3m7"
//	id := uid.New("")                 // "k3n5p8x2" (no prefix)
//
// Generate a secure identifier:
//
//	token := uid.Secure()      // 24 chars, cryptographically secure
//	token := uid.Secure(32)    // 32 chars
//
// Convert to DNS-1035 format:
//
//	label, err := uid.ToDNS1035("key_abc123")  // "key-abc123"
//	id := uid.FromDNS1035("key-abc123")        // "key_abc123"
//
// # Prefixes
//
// Standard prefixes are defined as [Prefix] constants to make IDs
// self-descriptive. See [KeyPrefix], [APIPrefix], [WorkspacePrefix], and
// others in prefix.go.
//
// # Format
//
// All generated identifiers follow the same pattern: the first character is
// always a lowercase letter (a-z), followed by alphanumeric characters (a-z,
// 0-9). When a prefix is provided, it is joined with an underscore separator.
// This ensures identifiers can be converted to valid DNS-1035 labels by
// replacing underscores with dashes via [ToDNS1035].
package uid
