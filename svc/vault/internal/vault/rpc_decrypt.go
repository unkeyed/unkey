package vault

import (
	"context"
	"encoding/base64"
	"fmt"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/encryption"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"google.golang.org/protobuf/proto"
)

func (s *Service) Decrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.DecryptRequest],
) (*connect.Response[vaultv1.DecryptResponse], error) {
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	res, err := s.decrypt(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(res), nil

}

func (s *Service) decrypt(
	ctx context.Context,
	req *vaultv1.DecryptRequest,
) (*vaultv1.DecryptResponse, error) {
	ctx, span := tracing.Start(ctx, "vault.Decrypt")
	defer span.End()

	b, err := base64.StdEncoding.DecodeString(req.GetEncrypted())
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}
	encrypted := vaultv1.Encrypted{} // nolint:exhaustruct
	err = proto.Unmarshal(b, &encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted data: %w", err)
	}

	cacheKey := fmt.Sprintf("%s-%s", req.GetKeyring(), encrypted.GetEncryptionKeyId())

	dek, hit := s.keyCache.Get(ctx, cacheKey)
	if hit == cache.Miss {
		dek, err = s.keyring.GetKey(ctx, req.GetKeyring(), encrypted.GetEncryptionKeyId())
		if err != nil {
			return nil, fmt.Errorf("failed to get dek in keyring %s: %w", req.GetKeyring(), err)
		}
		s.keyCache.Set(ctx, cacheKey, dek)
	}

	plaintext, err := encryption.Decrypt(dek.GetKey(), encrypted.GetNonce(), encrypted.GetCiphertext())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	return &vaultv1.DecryptResponse{
		Plaintext: string(plaintext),
	}, nil

}
