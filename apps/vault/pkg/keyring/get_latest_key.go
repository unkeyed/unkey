package keyring

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/storage"
)

// GetLatestKey returns the latest key from the keyring. If no key is found, it creates a new key.
func (k *Keyring) GetLatestKey(ctx context.Context, ringID string) (*vaultv1.DataEncryptionKey, error) {
	dek, err := k.GetKey(ctx, ringID, "LATEST")

	if err == nil {
		return dek, nil
	}

	if err != storage.ErrObjectNotFound {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	return k.CreateKey(ctx, ringID)
}
