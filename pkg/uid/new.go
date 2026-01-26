package uid

import (
	"math/rand/v2"
	"unsafe"
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

	if prefix == "" {
		buf := make([]byte, n)
		for i := range buf {
			buf[i] = defaultAlphabet[rand.IntN(len(defaultAlphabet))]
		}
		return unsafe.String(&buf[0], n)
	}

	buf := make([]byte, len(prefix)+1+n)
	copy(buf, prefix)
	buf[len(prefix)] = '_'
	for i := len(prefix) + 1; i < len(buf); i++ {
		buf[i] = defaultAlphabet[rand.IntN(len(defaultAlphabet))]
	}
	return unsafe.String(&buf[0], len(buf))
}
