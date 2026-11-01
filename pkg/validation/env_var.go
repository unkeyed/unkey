package validation

import (
	"encoding/base64"
	"regexp"
)

// EnvVarsBytesMax caps the combined key and value bytes of one environment's
// variables. Krane writes them into a single Kubernetes Secret, which the
// apiserver rejects above 1 MiB, and the container receives them through exec,
// where the kernel limits the whole environment to about 2 MiB.
const EnvVarsBytesMax = 512 << 10

// vaultCiphertextOverheadMin is the AES-GCM nonce and tag that every Vault
// ciphertext carries next to the plaintext bytes.
const vaultCiphertextOverheadMin = 12 + 16

// EnvVarBytesEncrypted returns an upper bound for the key and plaintext bytes of
// a variable whose stored value is Vault ciphertext of ciphertextBytes length.
// Vault ciphertext is base64 of a protobuf holding the AES-GCM output, so the
// bound needs no decryption.
func EnvVarBytesEncrypted(keyBytes, ciphertextBytes int) int {
	return keyBytes + max(base64.StdEncoding.DecodedLen(ciphertextBytes)-vaultCiphertextOverheadMin, 0)
}

// envVarKeyRegex matches POSIX shell variable names. Environment variables
// exist to become process env: builds expose them to install/build commands,
// and the runtime injects them into the container. Names with dots or hyphens
// are unreachable from shells and most languages' env APIs, so they are
// rejected at every boundary rather than failing in confusing ways later.
//
// This is deliberately stricter than what Kubernetes accepts for Secret data
// keys; the old, laxer rule let users create variables their builds and apps
// could never read.
var envVarKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ErrMsgInvalidEnvVarKey is the human-readable explanation used in validation errors.
const ErrMsgInvalidEnvVarKey = "only letters, digits, and underscores are allowed, and the name must not start with a digit"

// IsValidEnvVarKey reports whether key is a valid environment variable name.
func IsValidEnvVarKey(key string) bool {
	return envVarKeyRegex.MatchString(key)
}
