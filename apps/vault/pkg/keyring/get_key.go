package keyring

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
)




func (k *Keyring) GetKey(ctx context.Context, ringID, keyID string) (*vaultv1.DataEncryptionKey, error) {

	b, err := k.store.GetObject(ctx, k.buildLookupKey(ringID, keyID))
	
	if err != nil {

		return nil, fmt.Errorf("store did not return key: %w", err)
	}
dek, _, err := k.DecodeAndDecryptKey(ctx, b)
if err != nil {
	return nil, fmt.Errorf("failed to decode and decrypt key: %w", err)
}
return dek, nil
}
