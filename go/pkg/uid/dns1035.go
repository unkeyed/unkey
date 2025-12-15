package uid

import (
	"math/rand/v2"
	"strings"
)

const (
	dns1035AlphabetAlpha    = "abcdefghijklmnopqrstuvwxyz"
	dns1035AlphabetNum      = "0123456789"
	dns1035AlphabetAlphaNum = dns1035AlphabetAlpha + dns1035AlphabetNum
)

func DNS1035(length ...int) string {
	// Default to 8 characters if no length specified
	n := 8
	if len(length) > 0 {
		n = length[0]
	}

	// Pre-allocate builder for efficiency
	// We use strings.Builder to avoid repeated string concatenations
	// which would create O(n) intermediate strings
	var b strings.Builder
	b.Grow(n)

	for i := 0; i < n; i++ {
		if i == 0 {
			b.WriteByte(dns1035AlphabetAlpha[rand.IntN(len(dns1035AlphabetAlpha))])
		} else {
			b.WriteByte(dns1035AlphabetAlphaNum[rand.IntN(len(dns1035AlphabetAlphaNum))])
		}
	}

	return b.String()
}
