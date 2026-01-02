package uid

import (
	"crypto/sha256"
	"strings"

	"github.com/btcsuite/btcutil/base58"
)

func IdFromHash(s string, prefix ...string) string {

	hash := sha256.New()
	_, _ = hash.Write([]byte(s))

	id := base58.Encode(hash.Sum(nil))
	if len(prefix) > 0 && prefix[0] != "" {
		return strings.Join([]string{prefix[0], id}, "_")
	} else {
		return id
	}

}
