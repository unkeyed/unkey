package keyring

import (
	"context"
	"errors"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	"go.opentelemetry.io/otel/attribute"
)

func (k *Keyring) GetOrCreateKey(ctx context.Context, ringID, keyID string) (*vaultv1.DataEncryptionKey, error) {
	ctx, span := tracing.Start(ctx, "keyring.GetOrCreateKey")
	defer span.End()
	span.SetAttributes(attribute.String("ringID", ringID), attribute.String("keyID", keyID))
	dek, err := k.GetKey(ctx, ringID, keyID)
	if err == nil {
		return dek, nil
	}

	if errors.Is(err, storage.ErrObjectNotFound) {
		return k.CreateKey(ctx, ringID)
	}

	tracing.RecordError(span, err)

	return nil, fmt.Errorf("failed to get key: %w", err)

}
