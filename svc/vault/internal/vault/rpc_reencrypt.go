package vault

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

func (s *Service) ReEncrypt(ctx context.Context, req *connect.Request[vaultv1.ReEncryptRequest]) (*connect.Response[vaultv1.ReEncryptResponse], error) {
	if err := s.authenticate(req); err != nil {
		return nil, err
	}
	ctx, span := tracing.Start(ctx, "vault.ReEncrypt")
	defer span.End()
	s.logger.Info("reencrypting",
		"keyring", req.Msg.GetKeyring(),
	)

	decrypted, err := s.decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   req.Msg.GetKeyring(),
		Encrypted: req.Msg.GetEncrypted(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	s.keyCache.Clear(ctx)

	encrypted, err := s.encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: req.Msg.GetKeyring(),
		Data:    decrypted.GetPlaintext(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return connect.NewResponse(&vaultv1.ReEncryptResponse{
		Encrypted: encrypted.GetEncrypted(),
		KeyId:     encrypted.GetKeyId(),
	}), nil

}
