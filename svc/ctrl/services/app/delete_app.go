package app

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// DeleteApp enqueues a durable Restate workflow that cascades through
// the app's environments and cleans up all associated resources.
// Returns immediately after the workflow is enqueued; actual deletion
// is eventually consistent.
func (s *Service) DeleteApp(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteAppRequest],
) (*connect.Response[ctrlv1.DeleteAppResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.NotEmpty(req.Msg.GetAppId(), "app_id is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	_, err := db.Query.FindAppById(ctx, s.db.RW(), req.Msg.GetAppId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app not found: %s", req.Msg.GetAppId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find app: %w", err))
	}

	client := hydrav1.NewAppServiceIngressClient(s.restate, req.Msg.GetAppId())
	_, err = client.Delete().Send(ctx, &hydrav1.DeleteAppRequest{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger app deletion: %w", err))
	}

	return connect.NewResponse(&ctrlv1.DeleteAppResponse{}), nil
}
