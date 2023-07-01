package keys

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcutil/base58"
)

const separator = "_"

// Version 1 keys are constructed of 3 parts
// 1. 1 byte for the version
// 2. 1 byte to let us know the byteLength of the random part
// 3. X bytes of random data
// [VERSION, LEN, X,X,X,X,X,X,X,X,X,X,X,X,X,X,X,X]
type keyV1 struct {
	prefix string
	random []byte
}

func (k keyV1) Marshal() (string, error) {
	if len(k.random) > 255 {
		return "", fmt.Errorf("v1 keys can only handle 255 bytes of randomness")
	}
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(1)
	buf.WriteByte(byte(len(k.random)))
	buf.Write(k.random)

	s := base58.Encode(buf.Bytes())

	if k.prefix != "" {
		return strings.Join([]string{string(k.prefix), s}, separator), nil
	} else {
		return s, nil
	}
}

func (k *keyV1) Unmarshal(key string) error {
	split := strings.Split(key, separator)
	if len(split) == 2 {
		k.prefix = split[0]
	}

	rest := split[len(split)-1]

	buf := base58.Decode(rest)
	if buf[0] != 1 {
		return fmt.Errorf("key has wrong version, expected 1, got %d", buf[0])
	}
	byteLength := buf[1]

	k.random = buf[2 : 2+byteLength]
	return nil

}

func NewV1Key(prefix string, byteLength int) (string, error) {
	if byteLength > 255 {
		return "", fmt.Errorf("v1 keys can only handle 255 bytes of randomness")
	}
	random := make([]byte, byteLength)
	read, err := rand.Read(random)
	if err != nil {
		return "", fmt.Errorf("unable to read random data")
	}
	if read != byteLength {
		return "", fmt.Errorf("unable to read enough random data")
	}
	key := keyV1{
		prefix: prefix,
		random: random,
	}

	return key.Marshal()

}
