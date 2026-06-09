package ops

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// RebuildDeployment delegates to deployment.Service.Rebuild after bearer
// authentication. See the proto doc for semantics and guardrails.
func (s *Service) RebuildDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.RebuildDeploymentRequest],
) (*connect.Response[ctrlv1.RebuildDeploymentResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	newID, err := s.deployment.Rebuild(ctx, req.Msg.GetDeploymentId(), req.Msg.GetReason(), req.Msg.GetForce())
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.RebuildDeploymentResponse{
		DeploymentId: newID,
	}), nil
}
