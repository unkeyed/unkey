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
) (_ *connect.Response[vaultv1.EncryptResponse], err error) {
	defer func() { observeOperation("encrypt", err) }()
	if err := s.authenticate(req); err != nil {
		return nil, err
	}
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

	keyring := req.GetKeyring()
	cacheKey := fmt.Sprintf("%s-%s", keyring, LATEST)

	dek, _, err := s.keyCache.SWR(ctx, cacheKey, func(ctx context.Context) (*vaultv1.DataEncryptionKey, error) {
		return s.keyring.GetOrCreateKey(ctx, keyring, LATEST)
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get latest dek in keyring %s: %w", keyring, err)
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
