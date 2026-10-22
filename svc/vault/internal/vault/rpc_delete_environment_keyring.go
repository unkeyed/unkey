package vault

import (
	"context"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/vault/internal/keyring"
)

func (s *Service) DeleteEnvironmentKeyring(ctx context.Context, req *connect.Request[vaultv1.DeleteEnvironmentKeyringRequest]) (_ *connect.Response[vaultv1.DeleteEnvironmentKeyringResponse], err error) {
	defer func() { observeOperation("delete_environment_keyring", err) }()
	if err := s.authenticate(req); err != nil {
		return nil, err
	}
	if err := keyring.ValidateEnvironmentID(req.Msg.GetEnvironmentId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.keyring.DeleteEnvironment(ctx, req.Msg.GetEnvironmentId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.keyCache.Clear(ctx)
	return connect.NewResponse(&vaultv1.DeleteEnvironmentKeyringResponse{}), nil
}
