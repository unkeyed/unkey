package keyspace

import (
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

type KeySpace struct {
	store  storage.Storage
	logger logging.Logger

	// any of these can be used for decryption
	decryptionKeys map[string]*vaultv1.KeyEncryptionKey
	encryptionKey  *vaultv1.KeyEncryptionKey
}

type Config struct {
	Store  storage.Storage
	Logger logging.Logger

	DecryptionKeys map[string]*vaultv1.KeyEncryptionKey
	EncryptionKey  *vaultv1.KeyEncryptionKey
}

func New(config Config) (*KeySpace, error) {

	return &KeySpace{
		store:          config.Store,
		logger:         config.Logger,
		encryptionKey:  config.EncryptionKey,
		decryptionKeys: config.DecryptionKeys,
	}, nil
}

// The storage layer doesn't know about keyspaces, so we need to prefix the key with the keyspace id
func (k *KeySpace) buildLookupKey(spaceID, dekID string) string {
	return fmt.Sprintf("keyring/%s/%s", spaceID, dekID)
}
