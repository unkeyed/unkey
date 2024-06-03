package service

import (
	"context"
	"encoding/base64"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/encryption"
	"google.golang.org/protobuf/proto"
)

func (s *Service) Decrypt(
	ctx context.Context,
	req *vaultv1.DecryptRequest,
) (*vaultv1.DecryptResponse, error) {
	s.logger.Info().Str("keyring", req.Keyring).Msg("decrypting")

	b, err := base64.StdEncoding.DecodeString(req.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}
	encrypted := &vaultv1.Encrypted{}
	err = proto.Unmarshal(b, encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted data: %w", err)
	}

	cacheKey := fmt.Sprintf("%s-%s", req.Keyring, encrypted.EncryptionKeyId)

	dek, err := s.keyCache.Get(cacheKey)
	if err != nil {
		dek, err = s.keyring.GetKey(ctx, req.Keyring, encrypted.EncryptionKeyId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dek in keyring %s: %w", req.Keyring, err)
	}
	s.keyCache.Set(cacheKey, dek)

	plaintext, err := encryption.Decrypt(dek.Key, encrypted.Nonce, encrypted.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	return &vaultv1.DecryptResponse{
		Plaintext: string(plaintext),
	}, nil

}
