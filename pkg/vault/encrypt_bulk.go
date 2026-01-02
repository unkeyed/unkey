package vault

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

func (s *Service) EncryptBulk(
	ctx context.Context,
	req *vaultv1.EncryptBulkRequest,
) (*vaultv1.EncryptBulkResponse, error) {
	ctx, span := tracing.Start(ctx, "vault.EncryptBulk")
	defer span.End()

	res := &vaultv1.EncryptBulkResponse{
		Encrypted: make([]*vaultv1.EncryptResponse, len(req.GetData())),
	}

	for i, data := range req.GetData() {
		decryptResponse, err := s.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: req.GetKeyring(),
			Data:    data,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt request %d: %w", i, err)
		}
		res.Encrypted[i] = decryptResponse
	}

	return res, nil
}
