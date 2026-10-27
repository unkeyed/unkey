package vault

import (
	"context"
	"encoding/base64"
	"fmt"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"google.golang.org/protobuf/proto"
)

func (s *Service) ReEncrypt(ctx context.Context, req *connect.Request[vaultv1.ReEncryptRequest]) (_ *connect.Response[vaultv1.ReEncryptResponse], err error) {
	defer func() { observeOperation("reencrypt", err) }()
	if err := s.authenticate(req); err != nil {
		return nil, err
	}
	ctx, span := tracing.Start(ctx, "vault.ReEncrypt")
	defer span.End()
	logger.Info("reencrypting",
		"keyring", req.Msg.GetKeyring(),
	)

	decrypted, err := s.decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   req.Msg.GetKeyring(),
		Encrypted: req.Msg.GetEncrypted(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	keyID := req.Msg.GetKeyId()
	if keyID == "" {
		keyID = LATEST
	}
	dek, err := s.keyring.GetKey(ctx, req.Msg.GetKeyring(), keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get re-encryption key: %w", err)
	}
	if keyID == LATEST {
		s.keyCache.Set(ctx, fmt.Sprintf("%s-%s", req.Msg.GetKeyring(), LATEST), dek)
	}
	encoded, decodeErr := base64.StdEncoding.DecodeString(req.Msg.GetEncrypted())
	if decodeErr != nil {
		return nil, decodeErr
	}
	var original vaultv1.Encrypted
	if decodeErr = proto.Unmarshal(encoded, &original); decodeErr != nil {
		return nil, decodeErr
	}
	if original.GetEncryptionKeyId() == dek.GetId() {
		return connect.NewResponse(&vaultv1.ReEncryptResponse{
			Encrypted: req.Msg.GetEncrypted(), KeyId: dek.GetId(),
		}), nil
	}
	encrypted, err := encryptWithKey(decrypted.GetPlaintext(), dek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return connect.NewResponse(&vaultv1.ReEncryptResponse{
		Encrypted: encrypted.GetEncrypted(),
		KeyId:     encrypted.GetKeyId(),
	}), nil

}
