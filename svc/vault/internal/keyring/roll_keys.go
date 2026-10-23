package keyring

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

func (k *Keyring) RollKeys(ctx context.Context, ringID string) error {
	ctx, span := tracing.Start(ctx, "keyring.RollKeys")
	defer span.End()
	lookupKeys, err := k.store.ListObjectKeys(ctx, k.buildLookupKey(ringID, "dek_"))
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	for _, objectKey := range lookupKeys {
		if err := k.RewrapKey(ctx, objectKey); err != nil && !errors.Is(err, ErrDeleted) {
			return err
		}
	}

	return nil
}

func (k *Keyring) RewrapKey(ctx context.Context, objectKey string) error {
	path, ok := strings.CutPrefix(objectKey, "keyring/")
	separator := strings.LastIndex(path, "/")
	if !ok || separator < 0 {
		return fmt.Errorf("invalid key object path: %s", objectKey)
	}
	ringID, keyID := path[:separator], path[separator+1:]
	if err := k.CheckActive(ctx, ringID); err != nil {
		return err
	}
	b, found, err := k.store.GetObject(ctx, objectKey)
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	if !found {
		return nil
	}
	dek, kekID, err := k.DecodeAndDecryptKey(ctx, b)
	if err != nil {
		return fmt.Errorf("failed to decode and decrypt key: %w", err)
	}
	if kekID == k.encryptionKey.GetId() {
		return nil
	}
	reencrypted, err := k.EncryptAndEncodeKey(ctx, dek)
	if err != nil {
		return fmt.Errorf("failed to re-encrypt key: %w", err)
	}
	if err := k.storeKey(ctx, ringID, keyID, reencrypted); err != nil {
		return fmt.Errorf("failed to put re-encrypted key: %w", err)
	}
	return nil
}
