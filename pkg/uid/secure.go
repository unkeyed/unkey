package uid

import (
	"crypto/rand"
	"io"
	"strings"
)

// secureAlphabetMax is the largest byte value range that can be split evenly
// across defaultAlphabet. A byte has 256 possible values, but the alphabet has
// 62 characters. Since 256 is not divisible by 62, directly using b % 62 would
// make the first 8 characters more likely than the rest. Rejection sampling
// avoids that bias by only accepting byte values 0..247, because 248 is exactly
// 4 full copies of a 62-character alphabet.
const secureAlphabetMax = 256 - (256 % len(defaultAlphabet))

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

	return secure(n, rand.Reader)
}

// secure accepts a reader so tests can inject edge-case bytes without replacing
// the global crypto/rand.Reader used by Secure.
func secure(n int, reader io.Reader) string {
	if n <= 0 {
		return ""
	}

	var id strings.Builder
	id.Grow(n)

	bytes := make([]byte, n)
	for id.Len() < n {
		if _, err := io.ReadFull(reader, bytes); err != nil {
			panic("crypto/rand failed: " + err.Error())
		}

		for _, b := range bytes {
			if int(b) >= secureAlphabetMax {
				continue
			}

			id.WriteByte(defaultAlphabet[int(b)%len(defaultAlphabet)])
			if id.Len() == n {
				break
			}
		}
	}

	return id.String()
}
