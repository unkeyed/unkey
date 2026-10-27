package notification

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/notification"
)

// TestDestination queues a saved destination test, or reads the result of the
// same test key. Its status describes provider acceptance, not inbox delivery.
func (s *Service) TestDestination(ctx context.Context, req *connect.Request[ctrlv1.NotificationTestDestinationRequest]) (*connect.Response[ctrlv1.NotificationTestDestinationResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	if !validID(req.Msg.GetIdempotencyKey(), 128) || req.Msg.GetActor() == nil {
		return nil, rpcError(connect.CodeInvalidArgument, "an actor and an idempotency key of at most 128 bytes are required")
	}
	if _, err := findDestination(ctx, s.db, req.Msg.GetWorkspaceId(), req.Msg.GetDestinationId()); err != nil {
		return nil, err
	}
	if s.restate == nil {
		return nil, rpcError(connect.CodeUnavailable, "notification testing is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	key := notification.DestinationTestID(req.Msg.GetWorkspaceId(), req.Msg.GetDestinationId(), req.Msg.GetIdempotencyKey())
	result, err := hydrav1.NewNotificationTestServiceIngressClient(s.restate, key).TestDestination().Request(ctx, &hydrav1.TestNotificationDestinationRequest{
		WorkspaceId:    req.Msg.GetWorkspaceId(),
		DestinationId:  req.Msg.GetDestinationId(),
		IdempotencyKey: req.Msg.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, rpcError(connect.CodeUnavailable, "notification test status is unavailable; retry with the same idempotency key")
	}
	return connect.NewResponse(&ctrlv1.NotificationTestDestinationResponse{
		DeliveryId: result.GetDeliveryId(),
		Status:     result.GetStatus(),
		Reason:     result.GetReason(),
	}), nil
}
