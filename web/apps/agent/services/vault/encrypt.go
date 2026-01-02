package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/encryption"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/proto"
)

func (s *Service) Encrypt(
	ctx context.Context,
	req *vaultv1.EncryptRequest,
) (*vaultv1.EncryptResponse, error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("service.vault", "Encrypt"))
	defer span.End()
	span.SetAttributes(attribute.String("keyring", req.Keyring))

	cacheKey := fmt.Sprintf("%s-%s", req.Keyring, LATEST)

	dek, hit := s.keyCache.Get(ctx, cacheKey)
	if hit != cache.Hit {
		var err error
		dek, err = s.keyring.GetOrCreateKey(ctx, req.Keyring, LATEST)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest dek in keyring %s: %w", req.Keyring, err)
		}
		s.keyCache.Set(ctx, cacheKey, dek)
	}

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
