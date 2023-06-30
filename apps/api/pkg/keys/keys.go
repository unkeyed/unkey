package keys

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcutil/base58"
	"google.golang.org/protobuf/proto"
)

func NewKey(prefix string, byteLength int) (string, error) {
	random := make([]byte, byteLength)
	_, err := rand.Read(random)
	if err != nil {
		return "", fmt.Errorf("unable to get random bytes: %w", err)
	}

	key := &Key{
		Version: versionToBytes(0),
		Random:  random,
	}

	buf, err := proto.Marshal(key)
	if err != nil {
		return "", fmt.Errorf("unable to marshal key: %w", err)
	}

	suffix := base58.Encode(buf)

	if prefix != "" {
		return strings.Join([]string{string(prefix), suffix}, "_"), nil
	} else {
		return suffix, nil
	}
}

func versionToBytes(i uint8) []byte {
	b := make([]byte, 1)
	b[0] = byte(i)
	return b
}

// func versionFrombytes(b []byte) uint8 {
// 	return uint8(b[0])
// }
