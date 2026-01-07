package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/encryption"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/proto"
)

func (s *Service) Encrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.EncryptRequest],
) (*connect.Response[vaultv1.EncryptResponse], error) {
	res, err := s.encrypt(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(res), nil
}

func (s *Service) encrypt(
	ctx context.Context,
	req *vaultv1.EncryptRequest,
) (*vaultv1.EncryptResponse, error) {
	ctx, span := tracing.Start(ctx, "vault.Encrypt")
	defer span.End()
	span.SetAttributes(attribute.String("keyring", req.GetKeyring()))

	cacheKey := fmt.Sprintf("%s-%s", req.GetKeyring(), LATEST)

	dek, hit := s.keyCache.Get(ctx, cacheKey)
	if hit != cache.Hit {
		var err error
		dek, err = s.keyring.GetOrCreateKey(ctx, req.GetKeyring(), LATEST)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest dek in keyring %s: %w", req.GetKeyring(), err)
		}
		s.keyCache.Set(ctx, cacheKey, dek)
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
