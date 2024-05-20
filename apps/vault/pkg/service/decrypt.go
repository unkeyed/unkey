package service

import (
	"context"
	"encoding/base64"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/encryption"
	"google.golang.org/protobuf/proto"
)

func (s *Service) Decrypt(
	ctx context.Context,
	req *vaultv1.DecryptRequest,
) (*vaultv1.DecryptResponse, error) {
	

	b, err := base64.StdEncoding.DecodeString(req.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}
	encrypted := &vaultv1.Encrypted{}
	err = proto.Unmarshal(b, encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted data: %w", err)
	}

	dek, err := s.keyring.GetKey(ctx, req.Keyring, encrypted.EncryptionKeyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	plaintext, err := encryption.Decrypt(dek.Key, encrypted.Nonce, encrypted.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	return &vaultv1.DecryptResponse{
		Plaintext: string(plaintext),
	}, nil

}
