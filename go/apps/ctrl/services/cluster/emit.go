package cluster

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *Service) EmitSentinelEvent(ctx context.Context, labels map[string]string, event *ctrlv1.SentinelEvent) error {

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	emitted := 0
	for _, krane := range s.clients {
		shouldSend := true
		for wantLabel, wantValue := range krane.selectors {
			if labels[wantLabel] != wantValue {
				shouldSend = false
				break
			}
		}

		if shouldSend {
			krane.sentinelEvents.Buffer(event)
			emitted++
		}
	}
	s.logger.Info("Emitting sentinel event", "event", event, "labels", labels, "emitted", emitted)
	return nil
}

func (s *Service) EmitDeploymentEvent(ctx context.Context, labels map[string]string, event *ctrlv1.DeploymentEvent) error {

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	emitted := 0
	for _, krane := range s.clients {
		shouldSend := true
		for wantLabel, wantValue := range krane.selectors {
			if labels[wantLabel] != wantValue {
				shouldSend = false
				break
			}
		}

		if shouldSend {
			krane.deploymentEvents.Buffer(event)
			emitted++
		}
	}
	s.logger.Info("Emitting deployment event", "event", event, "labels", labels, "emitted", emitted)
	return nil
}
