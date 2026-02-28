package uid

import (
	"math/rand/v2"
	"strings"
)

const (
	dns1035Alpha    = "abcdefghijklmnopqrstuvwxyz"
	dns1035AlphaNum = dns1035Alpha + "0123456789"
)

// DNS1035 generates a random DNS-1035 compliant label.
//
// DNS-1035 labels must start with a letter and contain only lowercase
// letters and digits. Default length is 8 characters; pass a custom length
// to override.
//
// Uses math/rand/v2 which is NOT cryptographically secure. Do not use for
// security-sensitive purposes.
func DNS1035(length ...int) string {
	n := 8
	if len(length) > 0 {
		n = length[0]
	}

	if n == 0 {
		return ""
	}

	var id strings.Builder
	id.Grow(n)

	id.WriteByte(dns1035Alpha[rand.IntN(len(dns1035Alpha))])
	for i := 1; i < n; i++ {
		id.WriteByte(dns1035AlphaNum[rand.IntN(len(dns1035AlphaNum))])
	}

	return id.String()
}

// New generates a prefixed random identifier.
//
// The identifier consists of the prefix, an underscore separator, and a random
// portion that always starts with a lowercase letter followed by alphanumeric
// characters. Default random portion is 8 characters; pass a custom length to override.
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

		id.WriteByte(dns1035Alpha[rand.IntN(len(dns1035Alpha))])
		for i := 1; i < n; i++ {
			id.WriteByte(dns1035AlphaNum[rand.IntN(len(dns1035AlphaNum))])
		}

		return id.String()
	}

	id.Grow(len(prefix) + 1 + n)
	id.WriteString(string(prefix))
	id.WriteByte('_')

	id.WriteByte(dns1035Alpha[rand.IntN(len(dns1035Alpha))])
	for i := 1; i < n; i++ {
		id.WriteByte(dns1035AlphaNum[rand.IntN(len(dns1035AlphaNum))])
	}

	return id.String()
}
