package ctrl

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/buildinfo"
)

func (s *Service) Liveness(
	ctx context.Context,
	req *connect.Request[ctrlv1.LivenessRequest],
) (*connect.Response[ctrlv1.LivenessResponse], error) {
	res := connect.NewResponse(&ctrlv1.LivenessResponse{
		Status:     "ok",
		Version:    buildinfo.Version,
		InstanceId: s.instanceID,
	})

	return res, nil
}
