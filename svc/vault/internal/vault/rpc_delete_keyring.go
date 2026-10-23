package vault

import (
	"context"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

func (s *Service) DeleteKeyring(ctx context.Context, req *connect.Request[vaultv1.DeleteKeyringRequest]) (_ *connect.Response[vaultv1.DeleteKeyringResponse], err error) {
	defer func() { observeOperation("delete_keyring", err) }()
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	if err := s.keyring.Delete(ctx, req.Msg.GetKeyring()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.keyCache.Clear(ctx)
	return connect.NewResponse(&vaultv1.DeleteKeyringResponse{}), nil
}
