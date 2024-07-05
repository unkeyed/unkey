package keyring

import (
	"context"
	"errors"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/services/vault/storage"
)

func (k *Keyring) GetKey(ctx context.Context, ringID, keyID string) (*vaultv1.DataEncryptionKey, error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("keyring", "GetKey"))
	defer span.End()

	b, err := k.store.GetObject(ctx, k.buildLookupKey(ringID, keyID))

	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return nil, storage.ErrObjectNotFound
		}
		return nil, fmt.Errorf("failed to get object: %w", err)

	}
	dek, _, err := k.DecodeAndDecryptKey(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("failed to decode and decrypt key: %w", err)
	}
	return dek, nil
}
