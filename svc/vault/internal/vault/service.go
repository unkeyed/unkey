package vault

import (
	"encoding/base64"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	cacheMiddleware "github.com/unkeyed/unkey/pkg/cache/middleware"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/vault/internal/keyring"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"google.golang.org/protobuf/proto"
)

const LATEST = "LATEST"

type Service struct {
	keyCache cache.Cache[string, *vaultv1.DataEncryptionKey]

	storage storage.Storage

	decryptionKeys map[string]*vaultv1.KeyEncryptionKey
	encryptionKey  *vaultv1.KeyEncryptionKey

	keyring *keyring.Keyring
	bearer  string
}

var _ vaultv1connect.VaultServiceHandler = (*Service)(nil)

type Config struct {
	Storage           storage.Storage
	MasterKey         string
	PreviousMasterKey *string
	BearerToken       string
}

func New(cfg Config) (*Service, error) {

	encryptionKey, decryptionKeys, err := loadMasterKeys(cfg.MasterKey, cfg.PreviousMasterKey)
	if err != nil {
		return nil, fmt.Errorf("unable to load master keys: %w", err)

	}

	kr, err := keyring.New(keyring.Config{
		Store:          cfg.Storage,
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
		Resource: "data_encryption_key",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &Service{
		storage:        cfg.Storage,
		keyCache:       cacheMiddleware.WithTracing(cache),
		decryptionKeys: decryptionKeys,

		encryptionKey: encryptionKey,
		keyring:       kr,
		bearer:        cfg.BearerToken,
	}, nil
}

func loadMasterKeys(masterKey string, previousMasterKey *string) (*vaultv1.KeyEncryptionKey, map[string]*vaultv1.KeyEncryptionKey, error) {
	if masterKey == "" {
		return nil, nil, fmt.Errorf("no master key provided")
	}
	decryptionKeys := make(map[string]*vaultv1.KeyEncryptionKey)

	kek, err := parseMasterKey(masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse master key: %w", err)
	}
	decryptionKeys[kek.GetId()] = kek

	if previousMasterKey != nil && *previousMasterKey != "" {
		oldKek, err := parseMasterKey(*previousMasterKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse previous master key: %w", err)
		}
		decryptionKeys[oldKek.GetId()] = oldKek
	}

	return kek, decryptionKeys, nil
}

func parseMasterKey(masterKey string) (*vaultv1.KeyEncryptionKey, error) {
	kek := &vaultv1.KeyEncryptionKey{} // nolint:exhaustruct
	b, err := base64.StdEncoding.DecodeString(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}

	err = proto.Unmarshal(b, kek)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal master key: %w", err)
	}
	return kek, nil
}
