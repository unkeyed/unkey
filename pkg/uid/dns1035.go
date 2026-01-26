package uid

import (
	"math/rand/v2"
	"strings"
)

const (
	dns1035Alpha    = "abcdefghijklmnopqrstuvwxyz"
	dns1035AlphaNum = dns1035Alpha + "0123456789"
)

// DNS1035 generates a random string compliant with RFC 1035 DNS label rules.
//
// The first character is always a lowercase letter; subsequent characters are
// lowercase letters or digits. Default length is 8 characters; pass a custom
// length to override.
//
// Uses math/rand/v2 which is NOT cryptographically secure.
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
