package uid

import (
	"math/rand/v2"
	"strings"
)

const defaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// New generates a prefixed random identifier.
//
// The identifier consists of the prefix, an underscore separator, and random
// alphanumeric characters. Default random portion is 8 characters; pass a
// custom length to override.
//
// Pass an empty prefix to generate an identifier without a prefix.
//
// Uses math/rand/v2 which is NOT cryptographically secure. Do not use for
// API keys, tokens, or security-sensitive purposes.
func New(prefix Prefix, length ...int) string {
	n := 8
	if len(length) > 0 {
		n = length[0]
	}

	if n == 0 && prefix == "" {
		return ""
	}

	var id strings.Builder
	if prefix == "" {
		id.Grow(n)
		for i := 0; i < n; i++ {
			id.WriteByte(defaultAlphabet[rand.IntN(len(defaultAlphabet))])
		}
		return id.String()
	}

	id.Grow(len(prefix) + 1 + n)
	id.WriteString(string(prefix))
	id.WriteByte('_')
	for i := 0; i < n; i++ {
		id.WriteByte(defaultAlphabet[rand.IntN(len(defaultAlphabet))])
	}
	return id.String()
}
