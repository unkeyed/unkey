package keyring

import (
	"context"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/encryption"
	"google.golang.org/protobuf/proto"
)

func (k *Keyring) EncryptAndEncodeKey(ctx context.Context, dek *vaultv1.DataEncryptionKey) ([]byte, error) {
	b, err := proto.Marshal(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dek: %w", err)
	}

	nonce, ciphertext, err := encryption.Encrypt(k.encryptionKey.Key, b)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt dek: %w", err)
	}

	encryptedDek := &vaultv1.EncryptedDataEncryptionKey{
		Id:        dek.Id,
		CreatedAt: dek.CreatedAt,
		Encrypted: &vaultv1.Encrypted{
			Algorithm:       vaultv1.Algorithm_AES_256_GCM,
			Nonce:           nonce,
			Ciphertext:      ciphertext,
			EncryptionKeyId: k.encryptionKey.Id,
			Time:            time.Now().UnixMilli(),
		},
	}

	b, err = proto.Marshal(encryptedDek)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal encrypted dek: %w", err)
	}
	return b, nil
}
