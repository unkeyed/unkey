package keyring

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	"go.opentelemetry.io/otel/attribute"
)

func (k *Keyring) GetKey(ctx context.Context, ringID, keyID string) (*vaultv1.DataEncryptionKey, error) {
	ctx, span := tracing.Start(ctx, "keyring.GetKey")
	defer span.End()

	lookupKey := k.buildLookupKey(ringID, keyID)
	span.SetAttributes(attribute.String("lookupKey", lookupKey))

	b, found, err := k.store.GetObject(ctx, lookupKey)
	span.SetAttributes(attribute.Bool("found", found))
	if err != nil {
		tracing.RecordError(span, err)
		return nil, fmt.Errorf("failed to get object: %w", err)

	}
	if !found {
		return nil, storage.ErrObjectNotFound
	}

	dek, _, err := k.DecodeAndDecryptKey(ctx, b)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, fmt.Errorf("failed to decode and decrypt key: %w", err)
	}
	return dek, nil
}
