package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/cache"
	"github.com/unkeyed/unkey/apps/vault/pkg/encryption"
	"google.golang.org/protobuf/proto"
)

func (s *Service) Encrypt(
	ctx context.Context,
	req *vaultv1.EncryptRequest,
) (*vaultv1.EncryptResponse, error) {

	s.logger.Info().Str("keyring", req.Keyring).Msg("encrypting")
	cacheKey := fmt.Sprintf("%s-%s", req.Keyring, LATEST)


	dek, err := s.keyCache.Get(cacheKey)
	if err != nil {
		dek, err = s.keyring.GetOrCreateKey(ctx, req.Keyring, "LATEST")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest dek in keyring %s: %w", req.Keyring, err)
	}
	s.keyCache.Set(cacheKey, dek)


	nonce, ciphertext, err := encryption.Encrypt(dek.Key, []byte(req.GetData()))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	encryptedData := &vaultv1.Encrypted{
		Algorithm:       vaultv1.Algorithm_AES_256_GCM,
		Nonce:           nonce,
		Ciphertext:      ciphertext,
		EncryptionKeyId: dek.GetId(),
		Time:            time.Now().UnixMilli(),
	}

	b, err := proto.Marshal(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal encrypted data: %w", err)
	}

	
	return &vaultv1.EncryptResponse{
		Encrypted: base64.StdEncoding.EncodeToString(b),
		KeyId:     dek.GetId(),
	}, nil
}
