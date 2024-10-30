package vault

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func (s *Service) EncryptBulk(
	ctx context.Context,
	req *vaultv1.EncryptBulkRequest,
) (*vaultv1.EncryptBulkResponse, error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("service.vault", "EncryptBulk"))
	defer span.End()

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
