package sync

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *SyncEngine) route(ctx context.Context, infraEvent *ctrlv1.InfraEvent) error {

	switch event := infraEvent.GetEvent().(type) {
	case *ctrlv1.InfraEvent_DeploymentEvent:
		return s.routeDeploymentEvent(ctx, event.DeploymentEvent)

	case *ctrlv1.InfraEvent_SentinelEvent:
		return s.routeSentinelEvent(ctx, event.SentinelEvent)

	default:
		return fmt.Errorf("unknown event type: %v", event)
	}
}

func (s *SyncEngine) routeDeploymentEvent(ctx context.Context, event *ctrlv1.DeploymentEvent) error {
	switch deploymentEvent := event.GetEvent().(type) {
	case *ctrlv1.DeploymentEvent_Apply:
		return s.deploymentcontroller.ApplyDeployment(ctx, deploymentEvent.Apply)

	case *ctrlv1.DeploymentEvent_Delete:
		return s.deploymentcontroller.DeleteDeployment(ctx, deploymentEvent.Delete)

	default:
		return fmt.Errorf("unknown deployment event type: %v", deploymentEvent)

	}

}

func (s *SyncEngine) routeSentinelEvent(ctx context.Context, event *ctrlv1.SentinelEvent) error {
	switch SentinelEvent := event.GetEvent().(type) {
	case *ctrlv1.SentinelEvent_Apply:
		return s.sentinelcontroller.ApplySentinel(ctx, SentinelEvent.Apply)

	case *ctrlv1.SentinelEvent_Delete:
		return s.sentinelcontroller.DeleteSentinel(ctx, SentinelEvent.Delete)

	default:
		return fmt.Errorf("unknown sentinel event type: %v", SentinelEvent)

	}

}
