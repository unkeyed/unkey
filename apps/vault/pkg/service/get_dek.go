package service

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
)

func (s *Service) getDEK(ctx context.Context, keyringID string, keyID string) (*vaultv1.DataEncryptionKey, error) {

	dek, err := s.keyring.GetKey(ctx, keyringID, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}
	return dek, nil
}
