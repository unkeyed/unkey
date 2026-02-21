package uid

import (
	"crypto/rand"
	"strings"
)

// Secure generates a cryptographically secure random identifier.
//
// The identifier starts with a lowercase letter followed by alphanumeric
// characters. Default length is 24 characters; pass a custom length to override.
//
// Uses crypto/rand for secure random generation. Use this for verification
// tokens, API keys, or any security-sensitive purposes. Panics if crypto/rand
// fails, which should only happen in catastrophic system conditions.
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

	id.WriteByte(dns1035Alpha[int(bytes[0])%(len(dns1035Alpha))])
	for i := 1; i < n; i++ {
		id.WriteByte(dns1035AlphaNum[int(bytes[i])%(len(dns1035AlphaNum))])
	}

	return id.String()
}
