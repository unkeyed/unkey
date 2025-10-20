package keyspace

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	"go.opentelemetry.io/otel/attribute"
)

func (k *KeySpace) GetKey(ctx context.Context, spaceID, keyID string) (*vaultv1.DataEncryptionKey, error) {
	ctx, span := tracing.Start(ctx, "keyspace.GetKey")
	defer span.End()

	lookupKey := k.buildLookupKey(spaceID, keyID)
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
