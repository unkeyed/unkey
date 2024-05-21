package service

import (
	"context"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
)

func (s *Service) CreateDEK(ctx context.Context, req *vaultv1.CreateDEKRequest) (*vaultv1.CreateDEKResponse, error) {

	key, err := s.keyring.CreateKey(ctx, req.Keyring)
	if err != nil {
		return nil, err
	}
	return &vaultv1.CreateDEKResponse{
		KeyId: key.Id,
	}, nil
}
