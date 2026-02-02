// Package uid generates prefixed random identifiers for Unkey resources.
//
// The package provides two main functions for generating random strings:
// [New] for prefixed identifiers and [DNS1035] for DNS-compliant labels.
//
// # Security
//
// This package uses math/rand/v2 which is NOT cryptographically secure.
// The generated identifiers are predictable and MUST NOT be used for
// API keys, session tokens, or any security-sensitive purposes.
// Use crypto/rand directly for those cases.
//
// # Usage
//
// Generate a prefixed identifier:
//
//	id := uid.New(uid.KeyPrefix)      // "key_k3n5p8x2"
//	id := uid.New(uid.APIPrefix, 12)  // "api_a9k2n5p8x3m7"
//
// Generate a DNS-1035 compliant label:
//
//	label := uid.DNS1035()    // "k3n5p8x2" (starts with letter)
//	label := uid.DNS1035(12)  // "a9k2n5p8x3m7"
//
// # Prefixes
//
// Standard prefixes are defined as [Prefix] constants (KeyPrefix, APIPrefix,
// WorkspacePrefix, etc.) to make IDs self-descriptive. See prefix.go for the
// complete list.
package uid
