package keyring

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
)

func (k *Keyring) RollKeys(ctx context.Context, ringID string) error {
	ctx, span := tracing.Start(ctx, "keyring.RollKeys")
	defer span.End()
	lookupKeys, err := k.store.ListObjectKeys(ctx, k.buildLookupKey(ringID, "dek_"))
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	for _, objectKey := range lookupKeys {
		b, found, err := k.store.GetObject(ctx, objectKey)
		if err != nil {
			return fmt.Errorf("failed to get object: %w", err)
		}
		if !found {
			return storage.ErrObjectNotFound
		}

		dek, encryptionKeyId, err := k.DecodeAndDecryptKey(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to decode and decrypt key: %w", err)
		}
		if encryptionKeyId == k.encryptionKey.GetId() {
			k.logger.Info("key already encrypted with latest kek",
				"keyId", dek.GetId(),
			)
			continue
		}
		reencrypted, err := k.EncryptAndEncodeKey(ctx, dek)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt key: %w", err)
		}
		err = k.store.PutObject(ctx, objectKey, reencrypted)
		if err != nil {
			return fmt.Errorf("failed to put re-encrypted key: %w", err)
		}
	}

	return nil

}
