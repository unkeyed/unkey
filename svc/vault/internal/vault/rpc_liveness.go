package vault

import (
	"context"

	"connectrpc.com/connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

func (s *Service) Liveness(ctx context.Context, req *connect.Request[vaultv1.LivenessRequest]) (*connect.Response[vaultv1.LivenessResponse], error) {

	return connect.NewResponse(&vaultv1.LivenessResponse{
		Status: "ok",
	}), nil

}
