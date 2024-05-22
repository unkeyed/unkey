package service

import (
	"encoding/base64"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/cache"
	"github.com/unkeyed/unkey/apps/vault/pkg/keyring"
	"github.com/unkeyed/unkey/apps/vault/pkg/logging"
	"github.com/unkeyed/unkey/apps/vault/pkg/storage"
	"google.golang.org/protobuf/proto"
)

const LATEST = "LATEST"

type Service struct {
	logger   logging.Logger
	keyCache cache.Cache[*vaultv1.DataEncryptionKey]

	storage storage.Storage

	decryptionKeys map[string]*vaultv1.KeyEncryptionKey
	encryptionKey  *vaultv1.KeyEncryptionKey

	keyring *keyring.Keyring
}

type Config struct {
	Logger     logging.Logger
	Storage    storage.Storage
	MasterKeys []string
}

func New(cfg Config) (*Service, error) {

	encryptionKey, decryptionKeys, err := loadMasterKeys(cfg.MasterKeys)
	if err != nil {
		return nil, fmt.Errorf("unable to load master keys: %w", err)

	}

	keyring, err := keyring.New(keyring.Config{
		Store:          cfg.Storage,
		Logger:         cfg.Logger,
		DecryptionKeys: decryptionKeys,
		EncryptionKey:  encryptionKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)

	}

	return &Service{
		logger:         cfg.Logger,
		storage:        cfg.Storage,
		keyCache:       cache.NewInMemoryCache[*vaultv1.DataEncryptionKey](time.Minute * 5),
		decryptionKeys: decryptionKeys,

		encryptionKey: encryptionKey,
		keyring:       keyring,
	}, nil
}

func loadMasterKeys(masterKeys []string) (*vaultv1.KeyEncryptionKey, map[string]*vaultv1.KeyEncryptionKey, error) {
	if len(masterKeys) == 0 {
		return nil, nil, fmt.Errorf("no master keys provided")
	}
	encryptionKey := &vaultv1.KeyEncryptionKey{}
	decryptionKeys := make(map[string]*vaultv1.KeyEncryptionKey)

	for _, mk := range masterKeys {
		kek := &vaultv1.KeyEncryptionKey{}
		b, err := base64.StdEncoding.DecodeString(mk)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode master key: %w", err)
		}

		err = proto.Unmarshal(b, kek)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal master key: %w", err)
		}

		decryptionKeys[kek.Id] = kek
		// this way, the last key in the list is used for encryption
		encryptionKey = kek

	}
	return encryptionKey, decryptionKeys, nil
}
