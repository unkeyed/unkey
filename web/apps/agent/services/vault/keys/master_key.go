package keys

import (
	"encoding/base64"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/vault/v1"
	"google.golang.org/protobuf/proto"
)

func GenerateMasterKey() (*vaultv1.KeyEncryptionKey, string, error) {
	id, key, err := GenerateKey("kek")
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key: %w", err)
	}

	kek := &vaultv1.KeyEncryptionKey{
		Id:        id,
		CreatedAt: time.Now().UnixMilli(),
		Key:       key,
	}

	b, err := proto.Marshal(kek)

	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal key: %w", err)
	}

	return kek, base64.StdEncoding.EncodeToString(b), nil
}
