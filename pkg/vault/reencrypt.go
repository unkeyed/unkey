package vault

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

func (s *Service) ReEncrypt(ctx context.Context, req *vaultv1.ReEncryptRequest) (*vaultv1.ReEncryptResponse, error) {
	ctx, span := tracing.Start(ctx, "vault.ReEncrypt")
	defer span.End()
	s.logger.Info("reencrypting",
		"keyring", req.GetKeyring(),
	)

	decrypted, err := s.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   req.GetKeyring(),
		Encrypted: req.GetEncrypted(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	s.keyCache.Clear(ctx)

	encrypted, err := s.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: req.GetKeyring(),
		Data:    decrypted.GetPlaintext(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return &vaultv1.ReEncryptResponse{
		Encrypted: encrypted.GetEncrypted(),
		KeyId:     encrypted.GetKeyId(),
	}, nil

}
