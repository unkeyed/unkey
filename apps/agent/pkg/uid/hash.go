package uid

import (
	"crypto/sha256"
	"strings"
)

func IdFromHash(s string, prefix ...string) string {

	hash := sha256.New().Sum([]byte(s))

	id := string(hash)
	if len(prefix) > 0 && prefix[0] != "" {
		return strings.Join([]string{prefix[0], id}, "_")
	} else {
		return id
	}

}
