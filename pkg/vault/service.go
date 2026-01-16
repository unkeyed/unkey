package vault

import (
	"encoding/base64"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	cacheMiddleware "github.com/unkeyed/unkey/pkg/cache/middleware"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/vault/keyring"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	"google.golang.org/protobuf/proto"
)

// LATEST is a version identifier that refers to the most recent version of an
// encrypted value. Use this when you want to decrypt using the current key
// without specifying an explicit version number.
const LATEST = "LATEST"

// Service provides encryption and decryption operations using a hierarchical
// key management scheme. It manages data encryption keys (DEKs) which are
// themselves encrypted by key encryption keys (KEKs/master keys). The service
// caches DEKs to reduce storage lookups and handles key rotation transparently.
type Service struct {
	logger   logging.Logger
	keyCache cache.Cache[string, *vaultv1.DataEncryptionKey]

	storage storage.Storage

	decryptionKeys map[string]*vaultv1.KeyEncryptionKey
	encryptionKey  *vaultv1.KeyEncryptionKey

	keyring *keyring.Keyring
}

// Config holds configuration for creating a new vault [Service].
type Config struct {
	Logger     logging.Logger
	Storage    storage.Storage
	MasterKeys []string
}

// New creates a new vault [Service] with the provided configuration. The last
// key in [Config.MasterKeys] is used for encryption, while all keys can be used
// for decryption, enabling seamless key rotation.
func New(cfg Config) (*Service, error) {

	encryptionKey, decryptionKeys, err := loadMasterKeys(cfg.MasterKeys)
	if err != nil {
		return nil, fmt.Errorf("unable to load master keys: %w", err)

	}

	kr, err := keyring.New(keyring.Config{
		Store:          cfg.Storage,
		Logger:         cfg.Logger,
		DecryptionKeys: decryptionKeys,
		EncryptionKey:  encryptionKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	cache, err := cache.New(cache.Config[string, *vaultv1.DataEncryptionKey]{
		Fresh:    time.Hour,
		Stale:    24 * time.Hour,
		MaxSize:  10000,
		Logger:   cfg.Logger,
		Resource: "data_encryption_key",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &Service{
		logger:         cfg.Logger,
		storage:        cfg.Storage,
		keyCache:       cacheMiddleware.WithTracing(cache),
		decryptionKeys: decryptionKeys,

		encryptionKey: encryptionKey,
		keyring:       kr,
	}, nil
}

func loadMasterKeys(masterKeys []string) (*vaultv1.KeyEncryptionKey, map[string]*vaultv1.KeyEncryptionKey, error) {
	if len(masterKeys) == 0 {
		return nil, nil, fmt.Errorf("no master keys provided")
	}
	encryptionKey := &vaultv1.KeyEncryptionKey{} // nolint:exhaustruct
	decryptionKeys := make(map[string]*vaultv1.KeyEncryptionKey)

	for _, mk := range masterKeys {
		kek := &vaultv1.KeyEncryptionKey{} // nolint:exhaustruct
		b, err := base64.StdEncoding.DecodeString(mk)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode master key: %w", err)
		}

		err = proto.Unmarshal(b, kek)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal master key: %w", err)
		}

		decryptionKeys[kek.GetId()] = kek
		// this way, the last key in the list is used for encryption
		encryptionKey = kek

	}
	return encryptionKey, decryptionKeys, nil
}
