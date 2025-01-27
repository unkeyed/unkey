package hash

import (
	"crypto/sha256"
	"encoding/base64"
)

func Sha256(s string) string {
	hash := sha256.New()
	hash.Write([]byte(s))

	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
