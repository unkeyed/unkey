// Package hash provides cryptographic hashing utilities for various security-related
// operations such as API key validation, data integrity checks, and signatures.
//
// Currently, the package implements SHA-256 hashing with base64 encoding for
// consistent string representation. This approach provides a good balance between
// security and performance for most use cases within the application.
//
// These hash functions are designed for data integrity and comparison operations,
// not for password storage (which would require specialized password hashing
// algorithms like bcrypt or Argon2).
//
// Example usage:
//
//	// Generate a hash of an API key
//	hashedKey := hash.Sha256("unkey_1234567890abcdef")
//
//	// Compare a provided key against a stored hash
//	inputHash := hash.Sha256(userProvidedKey)
//	if inputHash == storedHash {
//	    // Key is valid
//	}
package hash
