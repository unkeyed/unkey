package keys

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcutil/base58"
	"google.golang.org/protobuf/proto"
)

const separator = "_"

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
		return strings.Join([]string{string(prefix), suffix}, separator), nil
	} else {
		return suffix, nil
	}
}

func DecodeKey(s string) (prefix string, version uint8, rest []byte, err error) {
	split := strings.Split(s, separator)

	if len(split) >= 2 {
		prefix = split[0]
	}

	decoded := base58.Decode(split[len(split)-1])
	key := &Key{}
	err = proto.Unmarshal(decoded, key)
	if err != nil {
		return "", 0, nil, fmt.Errorf("unable to unmarshal key")
	}
	version = versionFrombytes(key.Version)

	return prefix, version, key.Random, nil
}

func versionToBytes(i uint8) []byte {
	b := make([]byte, 1)
	b[0] = byte(i)
	return b
}

func versionFrombytes(b []byte) uint8 {
	return uint8(b[0])
}
