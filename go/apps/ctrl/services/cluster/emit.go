package cluster

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *Service) EmitEvent(ctx context.Context, labels map[string]string, event *ctrlv1.InfraEvent) error {

	s.logger.Info("emitting event", "event", event, "labels", labels)

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	s.logger.Info("Emitting event", "event", event, "labels", labels)
	for _, krane := range s.clients {
		shouldSend := true
		for wantLabel, wantValue := range krane.selectors {
			if labels[wantLabel] != wantValue {
				shouldSend = false
				break
			}
		}

		if shouldSend {
			krane.buffer.Buffer(event)
		}
	}
	return nil
}
