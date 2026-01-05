package keyring

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/agent/pkg/encryption"
	"github.com/unkeyed/unkey/svc/agent/pkg/tracing"
	"google.golang.org/protobuf/proto"
)

func (k *Keyring) DecodeAndDecryptKey(ctx context.Context, b []byte) (*vaultv1.DataEncryptionKey, string, error) {
	_, span := tracing.Start(ctx, tracing.NewSpanName("keyring", "DecodeAndDecryptKey"))
	defer span.End()
	encrypted := &vaultv1.EncryptedDataEncryptionKey{}
	err := proto.Unmarshal(b, encrypted)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, "", fmt.Errorf("failed to unmarshal encrypted dek: %w", err)
	}

	kek, ok := k.decryptionKeys[encrypted.Encrypted.EncryptionKeyId]
	if !ok {
		err = fmt.Errorf("no kek found for key id: %s", encrypted.Encrypted.EncryptionKeyId)
		tracing.RecordError(span, err)
		return nil, "", err
	}

	plaintext, err := encryption.Decrypt(kek.Key, encrypted.Encrypted.Nonce, encrypted.Encrypted.Ciphertext)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, "", fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	dek := &vaultv1.DataEncryptionKey{}
	err = proto.Unmarshal(plaintext, dek)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, "", fmt.Errorf("failed to unmarshal dek: %w", err)
	}
	return dek, encrypted.Encrypted.EncryptionKeyId, nil

}
