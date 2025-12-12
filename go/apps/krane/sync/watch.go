package sync

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

// Watch starts watching for events from the control plane
// stream disconnects are automatically handled
// and
func (s *SyncEngine) watch() {

	consecutiveFailures := 0
	var stream *connect.ServerStreamForClient[ctrlv1.InfraEvent]
	var err error
	for {
		if stream == nil {
			stream, err = s.newStream()
			if err != nil {
				consecutiveFailures++
				s.logger.Error("unable to connect to control plane", "consecutive_failures", consecutiveFailures)
				time.Sleep(time.Duration(min(60, consecutiveFailures)) * time.Second)
				continue
			} else {
				consecutiveFailures = 0
			}
		}

		hasMsg := stream.Receive()
		if !hasMsg {
			s.logger.Info("Stream ended, reconnecting...",
				"error", stream.Err(),
			)
			stream = nil
			time.Sleep(time.Second)
			continue
		}
		s.route(stream.Msg())
	}

}

func (s *SyncEngine) newStream() (*connect.ServerStreamForClient[ctrlv1.InfraEvent], error) {
	s.logger.Info("connecting to control plane to start stream")

	stream, err := s.ctrl.Watch(context.Background(), connect.NewRequest(&ctrlv1.WatchRequest{
		ClientId: s.instanceID,
		Selectors: map[string]string{
			"region": s.region,
			"shard":  s.shard,
		},
	}))
	s.logger.Info("stream", "stream", stream, "err", err)

	return stream, err

}
