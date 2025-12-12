package sync

import (
	"context"

	"connectrpc.com/connect"
	"github.com/bytedance/gopkg/util/logger"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

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
		switch x := stream.Msg().GetEvent().(type) {
		case *ctrlv1.InfraEvent_DeploymentEvent:
			s.deploymentcontroller.BufferEvent(x.DeploymentEvent)

		case *ctrlv1.InfraEvent_GatewayEvent:
			s.gatewaycontroller.BufferEvent(x.GatewayEvent)

		default:
			logger.Warn("Unknown event type", "event", x)
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
