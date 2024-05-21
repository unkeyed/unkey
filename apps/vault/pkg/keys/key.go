package keys

import (
	"crypto/rand"
	"fmt"
	"github.com/segmentio/ksuid"
)

func GenerateKey(prefix string) (id string, key []byte, err error) {

	key = make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate random data: %w", err)
	}

	return fmt.Sprintf("%s_%s", prefix, ksuid.New().String()), key, nil

}
