package service

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
)

func (s *Service) ReEncrypt(ctx context.Context, req *vaultv1.ReEncryptRequest) (*vaultv1.ReEncryptResponse, error) {

	s.logger.Info().Str("keyring", req.Keyring).Msg("reencrypting")

	decrypted, err := s.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   req.Keyring,
		Encrypted: req.Encrypted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	encrypted, err := s.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: req.Keyring,
		Data:    decrypted.Plaintext,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return &vaultv1.ReEncryptResponse{
		Encrypted: encrypted.Encrypted,
		KeyId:     encrypted.KeyId,
	}, nil

}
