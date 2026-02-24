package vault

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func (s *Service) DecryptBulk(
	ctx context.Context,
	req *connect.Request[vaultv1.DecryptBulkRequest],
) (*connect.Response[vaultv1.DecryptBulkResponse], error) {
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	ctx, span := tracing.Start(ctx, "vault.DecryptBulk")
	defer span.End()
	span.SetAttributes(
		attribute.String("keyring", req.Msg.GetKeyring()),
		attribute.Int("count", len(req.Msg.GetItems())),
	)

	responseItems := make(map[string]string, len(req.Msg.GetItems()))
	for id, encrypted := range req.Msg.GetItems() {
		res, err := s.decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   req.Msg.GetKeyring(),
			Encrypted: encrypted,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decrypt item %s: %w", id, err))
		}
		responseItems[id] = res.GetPlaintext()
	}

	return connect.NewResponse(&vaultv1.DecryptBulkResponse{
		Items: responseItems,
	}), nil
}
