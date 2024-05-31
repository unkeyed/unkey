package service

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
)

func (s *Service) EncryptBulk(
	ctx context.Context,
	req *vaultv1.EncryptBulkRequest,
) (*vaultv1.EncryptBulkResponse, error) {

	res := &vaultv1.EncryptBulkResponse{
		Encrypted: make([]*vaultv1.EncryptResponse, len(req.Data)),
	}

	for i, data := range req.Data {
		decryptResponse, err := s.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: req.Keyring,
			Data:    data,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt request %d: %w", i, err)
		}
		res.Encrypted[i] = decryptResponse
	}

	return res, nil
}
