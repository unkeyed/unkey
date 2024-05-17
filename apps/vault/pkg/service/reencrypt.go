package service

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
)


func (s *Service) ReEncrypt(ctx context.Context, req *vaultv1.ReEncryptRequest) (*vaultv1.ReEncryptResponse, error){

	decrypted, err := s.Decrypt(ctx, &vaultv1.DecryptRequest{
		Shard: req.GetShard(),
		Encrypted: req.GetEncrypted(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	encrypted, err := s.Encrypt(ctx, &vaultv1.EncryptRequest{
		Shard: req.GetShard(),
		Data: decrypted.GetPlaintext(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return &vaultv1.ReEncryptResponse{
		Encrypted: encrypted.GetEncrypted(),
	}, nil

}