package vault

import (
	"encoding/base64"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	cacheMiddleware "github.com/unkeyed/unkey/apps/agent/pkg/cache/middleware"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/services/vault/keyring"
	"github.com/unkeyed/unkey/apps/agent/services/vault/storage"
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
	Metrics    metrics.Metrics
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

	cache, err := cache.New[*vaultv1.DataEncryptionKey](cache.Config[*vaultv1.DataEncryptionKey]{
		Fresh:    time.Hour,
		Stale:    24 * time.Hour,
		MaxSize:  10000,
		Logger:   cfg.Logger,
		Metrics:  cfg.Metrics,
		Resource: "data_encryption_key",
	})

	return &Service{
		logger:         cfg.Logger,
		storage:        cfg.Storage,
		keyCache:       cacheMiddleware.WithTracing(cache),
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
