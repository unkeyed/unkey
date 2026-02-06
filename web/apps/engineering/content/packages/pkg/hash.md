---
title: hash
description: "provides cryptographic hashing utilities for various security-related"
---

Package hash provides cryptographic hashing utilities for various security-related operations such as API key validation, data integrity checks, and signatures.

Currently, the package implements SHA-256 hashing with base64 encoding for consistent string representation. This approach provides a good balance between security and performance for most use cases within the application.

These hash functions are designed for data integrity and comparison operations, not for password storage (which would require specialized password hashing algorithms like bcrypt or Argon2).

Example usage:

	// Generate a hash of an API key
	hashedKey := hash.Sha256("unkey_1234567890abcdef")

	// Compare a provided key against a stored hash
	inputHash := hash.Sha256(userProvidedKey)
	if inputHash == storedHash {
	    // Key is valid
	}

## Functions

### func Sha256

```go
func Sha256(s string) string
```

Sha256 computes the SHA-256 hash of a string and returns the result as a base64-encoded string. This format is more compact than hex encoding and suitable for storage in databases or transmission over networks.

This function is used primarily for API key validation, where the original key is never stored but its hash is used for verification.

Example:

	originalKey := "unkey_1234567890abcdef"
	hashedKey := hash.Sha256(originalKey)
	// Store hashedKey in the database

When validating:

	inputKey := getUserInput()
	inputHash := hash.Sha256(inputKey)
	if inputHash == storedHash {
	    // Key is valid
	}

