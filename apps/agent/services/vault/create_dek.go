package vault

import (
	"context"

	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func (s *Service) CreateDEK(ctx context.Context, req *vaultv1.CreateDEKRequest) (*vaultv1.CreateDEKResponse, error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("service.vault", "CreateDEK"))
	defer span.End()

	key, err := s.keyring.CreateKey(ctx, req.Keyring)
	if err != nil {
		return nil, err
	}
	return &vaultv1.CreateDEKResponse{
		KeyId: key.Id,
	}, nil
}
