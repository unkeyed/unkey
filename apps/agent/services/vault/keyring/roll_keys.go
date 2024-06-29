package keyring

import (
	"context"
	"fmt"
)

func (k *Keyring) RollKeys(ctx context.Context, ringID string) error {
	lookupKeys, err := k.store.ListObjectKeys(ctx, k.buildLookupKey(ringID, "dek_"))
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	for _, objectKey := range lookupKeys {
		b, err := k.store.GetObject(ctx, objectKey)
		if err != nil {
			return fmt.Errorf("failed to get object: %w", err)
		}
		dek, encryptionKeyId, err := k.DecodeAndDecryptKey(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to decode and decrypt key: %w", err)
		}
		if encryptionKeyId == k.encryptionKey.Id {
			k.logger.Info().Str("keyId", dek.Id).Msg("key already encrypted with latest kek")
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
