package keyspace

import (
	"context"
	"errors"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	"go.opentelemetry.io/otel/attribute"
)

func (k *KeySpace) GetOrCreateKey(ctx context.Context, spaceID, keyID string) (*vaultv1.DataEncryptionKey, error) {
	ctx, span := tracing.Start(ctx, "keyspace.GetOrCreateKey")
	defer span.End()
	span.SetAttributes(attribute.String("spaceID", spaceID), attribute.String("keyID", keyID))
	dek, err := k.GetKey(ctx, spaceID, keyID)
	if err == nil {
		return dek, nil
	}

	if errors.Is(err, storage.ErrObjectNotFound) {
		return k.CreateKey(ctx, spaceID)
	}

	tracing.RecordError(span, err)

	return nil, fmt.Errorf("failed to get key: %w", err)

}
