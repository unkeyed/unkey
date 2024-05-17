package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/encryption"
	"github.com/unkeyed/unkey/apps/vault/pkg/keys"
	"github.com/unkeyed/unkey/apps/vault/pkg/logging"
	"github.com/unkeyed/unkey/apps/vault/pkg/storage"
	"google.golang.org/protobuf/proto"
)

const LATEST = "LATEST"

type Service struct {
	logger  logging.Logger
	storage storage.Storage
	// any of these can be used for decryption
	decryptionKeys map[string]*vaultv1.KeyEncryptionKey
	encryptionKey  *vaultv1.KeyEncryptionKey
}

type Config struct {
	Logger     logging.Logger
	Storage    storage.Storage
	MasterKeys []string
}

func New(cfg Config) (*Service, error) {

	var encryptionKey *vaultv1.KeyEncryptionKey
	decryptionKeys := make(map[string]*vaultv1.KeyEncryptionKey)
	for _, mk := range cfg.MasterKeys {
		kek := &vaultv1.KeyEncryptionKey{}
		b, err := base64.StdEncoding.DecodeString(mk)
		if err != nil {
			return nil, fmt.Errorf("failed to decode master key: %w", err)
		}

		err = proto.Unmarshal(b, kek)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal master key: %w", err)
		}

		decryptionKeys[kek.Id] = kek

		if (encryptionKey == nil) || (kek.CreatedAt > encryptionKey.CreatedAt) {
			encryptionKey = kek

		}
	}

	return &Service{
		logger:         cfg.Logger,
		storage:        cfg.Storage,
		encryptionKey:  encryptionKey,
		decryptionKeys: decryptionKeys,
	}, nil
}

func (s *Service) Decrypt(
	ctx context.Context,
	req *vaultv1.DecryptRequest,
) (*vaultv1.DecryptResponse, error) {

	b, err := base64.StdEncoding.DecodeString(req.GetEncrypted())
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}
	encrypted := &vaultv1.Encrypted{}
	err = proto.Unmarshal(b, encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted data: %w", err)
	}

	dek, err := s.getDEK(ctx, req.GetShard(), encrypted.EncryptionKeyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get dek: %w", err)
	}

	decryptedData, err := encryption.Decrypt(dek.GetKey(), encrypted.Nonce, encrypted.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return &vaultv1.DecryptResponse{
		Plaintext: string(decryptedData),
	}, nil
}

func (s *Service) Encrypt(
	ctx context.Context,
	req *vaultv1.EncryptRequest,
) (*vaultv1.EncryptResponse, error) {

	dek, err :=s.loadDekOrCreateNew(ctx, req.GetShard(), LATEST)
	

	if err != nil {
		return nil, fmt.Errorf("failed to get dek: %w", err)
	}

	nonce, ciphertext, err := encryption.Encrypt(dek.GetKey(), []byte(req.GetData()))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	encryptedData := &vaultv1.Encrypted{
		Algorithm:       vaultv1.Algorithm_AES_256_GCM,
		Nonce:           nonce,
		Ciphertext:      ciphertext,
		EncryptionKeyId: dek.GetId(),
	}

	b, err := proto.Marshal(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal encrypted data: %w", err)
	}

	return &vaultv1.EncryptResponse{
		Encrypted: base64.StdEncoding.EncodeToString(b),
		KeyId: 	 dek.GetId(),
	}, nil
}

func (s *Service) loadDekOrCreateNew(ctx context.Context, shard, keyId string) (*vaultv1.DataEncryptionKey, error) {
	dek, err := s.getDEK(ctx, shard, keyId)
	if errors.Is(err, storage.ErrObjectNotFound) {
		dek, err = s.createDEK(ctx, shard)
		if err != nil {
			return nil, fmt.Errorf("failed to create dek: %w", err)
		}
		err = s.storeDEK(ctx, shard, dek)
		if err != nil {
			return nil, fmt.Errorf("failed to store dek: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dek: %w", err)
	}
	return dek, nil
}

func (s *Service) createDEK(ctx context.Context, shard string) (*vaultv1.DataEncryptionKey, error) {

	id, key, error := keys.GenerateKey("dek")
	if error != nil {
		return nil, fmt.Errorf("failed to generate key: %w", error)
	}

	return &vaultv1.DataEncryptionKey{
		Id:        id,
		Key:       key,
		CreatedAt: time.Now().UnixMilli(),
	}, nil

}

func (s *Service) storeDEK(ctx context.Context, shard string, dek *vaultv1.DataEncryptionKey) error {

	marshalledDek, err := proto.Marshal(dek)
	if err != nil {
		return fmt.Errorf("failed to marshal dek: %w", err)

	}

	encrypted, err := s.encrypt(ctx, &vaultv1.EncryptionKey{
		Id:  s.encryptionKey.GetId(),
		Key: s.encryptionKey.GetKey(),
	}, marshalledDek)
	if err != nil {
		return fmt.Errorf("failed to encrypt dek: %w", err)
	}

	encryptedDEK := &vaultv1.EncryptedDEK{
		Id:        dek.GetId(),
		CreatedAt: dek.GetCreatedAt(),
		Encrypted: encrypted,
	}

	marshalledEncryptedDek, err := proto.Marshal(encryptedDEK)
	if err != nil {
		return fmt.Errorf("failed to marshal dek: %w", err)
	}


	err = s.storage.PutObject(ctx, s.storage.Key(shard, dek.GetId()), marshalledEncryptedDek)
	if err != nil {
		return fmt.Errorf("failed to store dek: %w", err)
	}
	err = s.storage.PutObject(ctx, s.storage.Latest(shard), marshalledEncryptedDek)
	if err != nil {
		return fmt.Errorf("failed to store dek: %w", err)
	}

	return nil

}

func (s *Service) CreateDEK(ctx context.Context, req *vaultv1.CreateDEKRequest) (*vaultv1.CreateDEKResponse, error) {

	dek, err := s.createDEK(ctx, req.GetShard())
	if err != nil {
		return nil, fmt.Errorf("failed to create dek: %w", err)
	}

	err = s.storeDEK(ctx, req.GetShard(), dek)
	if err != nil {
		return nil, fmt.Errorf("failed to store dek: %w", err)
	}

	return &vaultv1.CreateDEKResponse{
		KeyId: dek.GetId(),
	}, nil

}

func (s *Service) getDEK(ctx context.Context, shard string, keyID string) (*vaultv1.DataEncryptionKey, error) {
	b, err := s.storage.GetObject(ctx, s.storage.Key(shard, keyID))

	if err != nil {
		return nil, fmt.Errorf("failed to get dek: %w", err)
	}

	encryptedDEK := &vaultv1.EncryptedDEK{}
	err = proto.Unmarshal(b, encryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dek: %w", err)
	}

	return s.decryptDEK(ctx, encryptedDEK)

}

func (s *Service) getKEK(_ctx context.Context, id string) (*vaultv1.KeyEncryptionKey, error) {
	kek := s.decryptionKeys[id]
	if kek == nil {
		return nil, fmt.Errorf("no decryption key found with id %s", id)
	}
	return kek, nil
}

// using kek to encrypt dek kek=nXXa+zU3uBFcpf6Pqy1oCz/znpT6eJLI3lOJxutyphg=
// decryptionKey={"created_at":1715886226741,"id":"kek_2gYtETxUHWUbG23UTTTN75V9e12","key":"nXXa+zU3uBFcpf6Pqy1oCz/znpT6eJLI3lOJxutyphg="}

// storing bytes="{\"id\":\"dek_2gaKg1giiz45w2BPSEyChhSqYpY\",\"created_at\":1715930355688,\"encrypted\":{\"nonce\":\"wz3/9SujvmH5BCDb\",\"ciphertext\":\"NvcaAee+Tu77tqeO82PHBh2kCt8676JCEGggLyLIvj5xNQUBX8RDepdpmgGZOOkY\",\"encrypted_by\":\"kek_2gYtETxUHWUbG23UTTTN75V9e12\"}}"
// loaded bytes=  {\"id\":\"dek_2gaKg1giiz45w2BPSEyChhSqYpY\",\"created_at\":1715930355688,\"encrypted\":{\"nonce\":\"wz3/9SujvmH5BCDb\",\"ciphertext\":\"NvcaAee+Tu77tqeO82PHBh2kCt8676JCEGggLyLIvj5xNQUBX8RDepdpmgGZOOkY\",\"encrypted_by\":\"kek_2gYtETxUHWUbG23UTTTN75V9e12\"}}"
