package sync

import ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"

func (s *SyncEngine) Subscribe() <-chan *ctrlv1.InfraEvent {
	return s.events.Consume()
}
