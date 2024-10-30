package keyring

import (
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/services/vault/storage"
)

type Keyring struct {
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

func New(config Config) (*Keyring, error) {

	return &Keyring{
		store:          config.Store,
		logger:         config.Logger,
		encryptionKey:  config.EncryptionKey,
		decryptionKeys: config.DecryptionKeys,
	}, nil
}

// The storage layer doesn't know about keyrings, so we need to prefix the key with the keyring id
func (k *Keyring) buildLookupKey(ringID, dekID string) string {
	return fmt.Sprintf("keyring/%s/%s", ringID, dekID)
}
