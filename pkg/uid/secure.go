package uid

import (
	"crypto/rand"
	"strings"
)

// Secure generates a cryptographically secure random identifier.
//
// Uses crypto/rand for secure random generation. Use this for verification
// tokens, API keys, or any security-sensitive purposes.
//
// The identifier consists of random alphanumeric characters. Default length
// is 24 characters; pass a custom length to override.
func Secure(length ...int) string {
	n := 24
	if len(length) > 0 {
		n = length[0]
	}

	if n == 0 {
		return ""
	}

	var id strings.Builder
	id.Grow(n)

	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}

	for i := 0; i < n; i++ {
		id.WriteByte(defaultAlphabet[int(bytes[i])%len(defaultAlphabet)])
	}

	return id.String()
}
