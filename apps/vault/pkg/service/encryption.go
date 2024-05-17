package service

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/encryption"
	"google.golang.org/protobuf/proto"
)

func (s *Service) encrypt(_ctx context.Context, key *vaultv1.EncryptionKey, plaintext []byte) (*vaultv1.Encrypted, error) {
	nonce, ciphertext, err := encryption.Encrypt(key.GetKey(), plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt plaintext: %w", err)
	}
	return &vaultv1.Encrypted{
		Algorithm:   vaultv1.Algorithm_AES_256_GCM,
		Nonce:       nonce,
		Ciphertext:  ciphertext,
		EncryptionKeyId: key.GetId(),
	}, nil
}

func (s *Service) encryptDEK(ctx context.Context, dek *vaultv1.DataEncryptionKey) (*vaultv1.EncryptedDEK, error) {
	marshalledDek, err := proto.Marshal(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dek: %w", err)

	}

	encrypted, err := s.encrypt(ctx, &vaultv1.EncryptionKey{
		Id:  s.encryptionKey.GetId(),
		Key: s.encryptionKey.GetKey(),
	}, marshalledDek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt dek: %w", err)
	}

	encryptedDEK := &vaultv1.EncryptedDEK{
		Id:        dek.GetId(),
		CreatedAt: dek.GetCreatedAt(),
		Encrypted: encrypted,
	}

	return encryptedDEK, nil
}

func (s *Service) decrypt(_ctx context.Context, key *vaultv1.EncryptionKey, encrypted *vaultv1.Encrypted) ([]byte, error) {
	plaintext, err := encryption.Decrypt(key.GetKey(), encrypted.GetNonce(), encrypted.GetCiphertext())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}
	return plaintext, nil
}



func (s *Service) decryptDEK(ctx context.Context, encrypted *vaultv1.EncryptedDEK) (*vaultv1.DataEncryptionKey, error) {


	kek, err := s.getKEK(ctx, encrypted.Encrypted.EncryptionKeyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get key decryption key: %w", err)
	}

	decrypted, err := s.decrypt(ctx, &vaultv1.EncryptionKey{
		Id:  kek.GetId(),
		Key: kek.GetKey(),
	}, encrypted.GetEncrypted())

	if err != nil {
		return nil, fmt.Errorf("failed to decrypt dek: %w", err)
	}
	dek := &vaultv1.DataEncryptionKey{}
	err = proto.Unmarshal(decrypted, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dek: %w", err)
	}
	return dek, nil


}