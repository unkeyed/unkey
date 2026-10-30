package vault

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func (s *Service) ReEncryptBulk(ctx context.Context, req *connect.Request[vaultv1.ReEncryptBulkRequest]) (_ *connect.Response[vaultv1.ReEncryptBulkResponse], err error) {
	defer func() { observeOperation("reencrypt_bulk", err) }()
	if err := s.authenticate(req); err != nil {
		return nil, err
	}
	if len(req.Msg.GetItems()) > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at most 100 items are allowed"))
	}
	ctx, span := tracing.Start(ctx, "vault.ReEncryptBulk")
	defer span.End()
	span.SetAttributes(attribute.Int("count", len(req.Msg.GetItems())))

	items := make(map[string]*vaultv1.ReEncryptBulkResponseItem, len(req.Msg.GetItems()))
	for id, item := range req.Msg.GetItems() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		decrypted, err := s.decrypt(ctx, item)
		if err != nil {
			span.RecordError(err)
			items[id] = &vaultv1.ReEncryptBulkResponseItem{
				Result: &vaultv1.ReEncryptBulkResponseItem_Error{Error: "failed to decrypt item"},
			}
			continue
		}
		encrypted, err := s.encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: item.GetKeyring(), Data: decrypted.GetPlaintext(),
		})
		if err != nil {
			span.RecordError(err)
			items[id] = &vaultv1.ReEncryptBulkResponseItem{
				Result: &vaultv1.ReEncryptBulkResponseItem_Error{Error: "failed to encrypt item"},
			}
			continue
		}
		items[id] = &vaultv1.ReEncryptBulkResponseItem{
			Result: &vaultv1.ReEncryptBulkResponseItem_Success{Success: encrypted},
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return connect.NewResponse(&vaultv1.ReEncryptBulkResponse{Items: items}), nil
}
