package cluster

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *Service) EmitSentinelState(ctx context.Context, labels map[string]string, event *ctrlv1.SentinelState) error {

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
			krane.sentinelStates.Buffer(event)
			emitted++
		}
	}
	s.logger.Info("Emitting sentinel event", "event", event, "labels", labels, "emitted", emitted)
	return nil
}

func (s *Service) EmitDeploymentState(ctx context.Context, labels map[string]string, event *ctrlv1.DeploymentState) error {

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
			krane.deploymentStates.Buffer(event)
			emitted++
		}
	}
	s.logger.Info("Emitting deployment event", "event", event, "labels", labels, "emitted", emitted)
	return nil
}
