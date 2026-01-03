package hash

import (
	"crypto/sha256"
	"encoding/base64"
)

// Sha256 computes the SHA-256 hash of a string and returns the result
// as a base64-encoded string. This format is more compact than hex encoding
// and suitable for storage in databases or transmission over networks.
//
// This function is used primarily for API key validation, where the original
// key is never stored but its hash is used for verification.
//
// Example:
//
//	originalKey := "unkey_1234567890abcdef"
//	hashedKey := hash.Sha256(originalKey)
//	// Store hashedKey in the database
//
// When validating:
//
//	inputKey := getUserInput()
//	inputHash := hash.Sha256(inputKey)
//	if inputHash == storedHash {
//	    // Key is valid
//	}
func Sha256(s string) string {
	hash := sha256.New()
	hash.Write([]byte(s))

	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
