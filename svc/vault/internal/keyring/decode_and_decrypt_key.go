package keyring

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/encryption"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"google.golang.org/protobuf/proto"
)

func (k *Keyring) DecodeAndDecryptKey(ctx context.Context, b []byte) (*vaultv1.DataEncryptionKey, string, error) {
	_, span := tracing.Start(ctx, "keyring.DecodeAndDecryptKey")
	defer span.End()
	encrypted := &vaultv1.EncryptedDataEncryptionKey{} // nolint:exhaustruct
	err := proto.Unmarshal(b, encrypted)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, "", fmt.Errorf("failed to unmarshal encrypted dek: %w", err)
	}

	kek, ok := k.decryptionKeys[encrypted.GetEncrypted().GetEncryptionKeyId()]
	if !ok {
		err = fmt.Errorf("no kek found for key id: %s", encrypted.GetEncrypted().GetEncryptionKeyId())
		tracing.RecordError(span, err)
		return nil, "", err
	}

	plaintext, err := encryption.Decrypt(kek.GetKey(), encrypted.GetEncrypted().GetNonce(), encrypted.GetEncrypted().GetCiphertext())
	if err != nil {
		tracing.RecordError(span, err)
		return nil, "", fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	dek := &vaultv1.DataEncryptionKey{} // nolint:exhaustruct
	err = proto.Unmarshal(plaintext, dek)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, "", fmt.Errorf("failed to unmarshal dek: %w", err)
	}
	return dek, encrypted.GetEncrypted().GetEncryptionKeyId(), nil

}
