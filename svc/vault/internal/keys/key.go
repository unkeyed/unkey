package keys

import (
	"crypto/rand"
	"fmt"

	"github.com/unkeyed/unkey/pkg/uid"
)

func GenerateKey(prefix uid.Prefix) (id string, key []byte, err error) {

	key = make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate random data: %w", err)
	}

	return uid.New(prefix), key, nil

}
