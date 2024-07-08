package vault

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func (s *Service) ReEncrypt(ctx context.Context, req *vaultv1.ReEncryptRequest) (*vaultv1.ReEncryptResponse, error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("service.vault", "ReEncrypt"))
	defer span.End()
	s.logger.Info().Str("keyring", req.Keyring).Msg("reencrypting")

	decrypted, err := s.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   req.Keyring,
		Encrypted: req.Encrypted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// TODO: this is very inefficient, as it clears the entire cache for every key re-encryption
	s.keyCache.Clear(ctx)

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
