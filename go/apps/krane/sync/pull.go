package sync

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

// pull retrieves desired state updates from the control plane.
//
// This method establishes a streaming connection with the control plane
// and continuously receives infrastructure events. It routes deployment
// events to the deployment controller and sentinel events to the sentinel
// controller for processing.
//
// The pull operation includes:
//   - Streaming gRPC connection to control plane
//   - Event type routing to appropriate controllers
//   - Error handling and logging
//   - Connection resilience through retry mechanisms
//
// Returns an error if the streaming connection fails or encounters
// communication problems.
func (s *SyncEngine) pull() error {
	s.logger.Info("starting pull from control plane")

	stream, err := s.ctrl.GetDesiredState(context.Background(), connect.NewRequest(&ctrlv1.GetDesiredStateRequest{
		ClientId: s.instanceID,
		Selectors: map[string]string{
			"region": s.region,
			"shard":  s.shard,
		},
	}))
	if err != nil {
		s.logger.Warn("failed to pull desired state", "error", err)
		return err
	}

	totalEvents := 0
	for stream.Receive() {
		msg := stream.Msg()
		err = s.route(context.Background(), msg)
		if err != nil {
			s.logger.Error("failed to handle message", "error", err, "message", msg)
		}
	}
	err = stream.Err()
	if err != nil {
		s.logger.Warn("failed to pull desired state", "error", err)
		return err
	}
	s.logger.Debug("completed pull from control plane",
		"total_events", totalEvents,
	)

	return nil

}
