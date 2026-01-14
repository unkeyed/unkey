package keyring

import (
	"context"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/vault/internal/keys"
)

func (k *Keyring) CreateKey(ctx context.Context, ringID string) (*vaultv1.DataEncryptionKey, error) {
	ctx, span := tracing.Start(ctx, "keyring.CreateKey")
	defer span.End()
	keyId, key, err := keys.GenerateKey("dek")
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	dek := &vaultv1.DataEncryptionKey{
		Id:        keyId,
		Key:       key,
		CreatedAt: time.Now().UnixMilli(),
	}

	b, err := k.EncryptAndEncodeKey(ctx, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt and encode dek: %w", err)
	}

	err = k.store.PutObject(ctx, k.buildLookupKey(ringID, dek.GetId()), b)
	if err != nil {
		return nil, fmt.Errorf("failed to put encrypted dek: %w", err)
	}
	err = k.store.PutObject(ctx, k.buildLookupKey(ringID, "LATEST"), b)
	if err != nil {
		return nil, fmt.Errorf("failed to put encrypted dek: %w", err)
	}

	return dek, nil
}
