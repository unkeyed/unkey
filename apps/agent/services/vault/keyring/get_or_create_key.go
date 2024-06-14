package keyring

import (
	"context"
	"errors"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/services/vault/storage"
)

func (k *Keyring) GetOrCreateKey(ctx context.Context, ringID, keyID string) (*vaultv1.DataEncryptionKey, error) {
	dek, err := k.GetKey(ctx, ringID, keyID)
	if err == nil {
		return dek, nil
	}

	if errors.Is(err, storage.ErrObjectNotFound) {
		return k.CreateKey(ctx, ringID)
	}

	return nil, fmt.Errorf("failed to get key: %w", err)

}
