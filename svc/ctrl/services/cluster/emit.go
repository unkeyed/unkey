package cluster

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

func (s *Service) EmitState(ctx context.Context, region string, event *ctrlv1.State) error {

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	sent := false
	for clusterID, connections := range s.clients {
		for connID, krane := range connections {
			if krane.region == region {
				s.logger.Info("sending to krane", "clusterID", clusterID, "connID", connID)
				if err := krane.stream.Send(event); err != nil {
					s.logger.Error("failed to send to krane", "clusterID", clusterID, "connID", connID, "error", err)
					continue
				}
				sent = true
			}
		}
	}
	if !sent {
		return fmt.Errorf("no cluster is listening for events in region %s", region)
	}
	return nil

}
