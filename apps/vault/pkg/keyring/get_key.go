package keyring

import (
	"context"
	"errors"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/storage"
)




func (k *Keyring) GetKey(ctx context.Context, ringID, keyID string) (*vaultv1.DataEncryptionKey, error) {

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
