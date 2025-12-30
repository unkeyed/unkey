package cluster

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *Service) EmitState(ctx context.Context, region string, event *ctrlv1.State) error {

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	s.logger.Info("clients", "count", len(s.clients))

	for _, krane := range s.clients {
		s.logger.Info("found krane", "krane", krane)
		if krane.region == region {
			return krane.stream.Send(event)
		}
	}
	return fmt.Errorf("no cluster is listening for events in region %s", region)

}
