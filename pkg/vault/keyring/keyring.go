package keyring

import (
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/vault/storage"
)

type Keyring struct {
	store storage.Storage

	// any of these can be used for decryption
	decryptionKeys map[string]*vaultv1.KeyEncryptionKey
	encryptionKey  *vaultv1.KeyEncryptionKey
}

type Config struct {
	Store storage.Storage

	DecryptionKeys map[string]*vaultv1.KeyEncryptionKey
	EncryptionKey  *vaultv1.KeyEncryptionKey
}

func New(config Config) (*Keyring, error) {

	return &Keyring{
		store:          config.Store,
		encryptionKey:  config.EncryptionKey,
		decryptionKeys: config.DecryptionKeys,
	}, nil
}

// The storage layer doesn't know about keyrings, so we need to prefix the key with the keyring id
func (k *Keyring) buildLookupKey(ringID, dekID string) string {
	return fmt.Sprintf("keyring/%s/%s", ringID, dekID)
}
