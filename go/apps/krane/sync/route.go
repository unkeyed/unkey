package sync

import (
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *SyncEngine) route(infraEvent *ctrlv1.InfraEvent) {

	switch e := infraEvent.GetEvent().(type) {
	case *ctrlv1.InfraEvent_DeploymentEvent:
		s.deploymentcontroller.BufferEvent(e.DeploymentEvent)

	case *ctrlv1.InfraEvent_SentinelEvent:
		s.sentinelcontroller.BufferEvent(e.SentinelEvent)

	default:
		s.logger.Warn("Unknown event type", "event", e)
	}
}
