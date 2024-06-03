package keyring

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/encryption"
	"google.golang.org/protobuf/proto"
)

func (k *Keyring) DecodeAndDecryptKey(ctx context.Context, b []byte) (*vaultv1.DataEncryptionKey, string, error) {
	encrypted := &vaultv1.EncryptedDataEncryptionKey{}
	err := proto.Unmarshal(b, encrypted)
	if err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal encrypted dek: %w", err)
	}

	kek, ok := k.decryptionKeys[encrypted.Encrypted.EncryptionKeyId]
	if !ok {
		return nil, "", fmt.Errorf("no kek found for key id: %s", encrypted.Encrypted.EncryptionKeyId)
	}

	plaintext, err := encryption.Decrypt(kek.Key, encrypted.Encrypted.Nonce, encrypted.Encrypted.Ciphertext)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	dek := &vaultv1.DataEncryptionKey{}
	err = proto.Unmarshal(plaintext, dek)
	if err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal dek: %w", err)
	}
	return dek, encrypted.Encrypted.EncryptionKeyId, nil

}
