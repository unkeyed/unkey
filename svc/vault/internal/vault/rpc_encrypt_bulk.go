package vault

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func (s *Service) EncryptBulk(
	ctx context.Context,
	req *connect.Request[vaultv1.EncryptBulkRequest],
) (*connect.Response[vaultv1.EncryptBulkResponse], error) {
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	ctx, span := tracing.Start(ctx, "vault.EncryptBulk")
	defer span.End()
	span.SetAttributes(
		attribute.String("keyring", req.Msg.GetKeyring()),
		attribute.Int("count", len(req.Msg.GetItems())),
	)

	responseItems := make(map[string]*vaultv1.EncryptBulkResponseItem, len(req.Msg.GetItems()))
	for id, data := range req.Msg.GetItems() {
		res, err := s.encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: req.Msg.GetKeyring(),
			Data:    data,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to encrypt item %s: %w", id, err))
		}
		responseItems[id] = &vaultv1.EncryptBulkResponseItem{
			Encrypted: res.GetEncrypted(),
			KeyId:     res.GetKeyId(),
		}
	}

	return connect.NewResponse(&vaultv1.EncryptBulkResponse{
		Items: responseItems,
	}), nil
}
