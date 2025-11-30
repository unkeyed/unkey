package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
)

func (s *Service) Watch(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], stream *connect.ServerStream[ctrlv1.InfraEvent]) error {
	done := make(chan struct{})

	clientID := req.Msg.GetClientId()
	selectors := req.Msg.GetSelectors()

	s.logger.Info("watch request received",
		"client_id", clientID,
		"selectors", selectors,
	)

	s.clientsMu.Lock()

	c, ok := s.clients[clientID]
	if !ok {
		c = &client{
			clientID:  clientID,
			selectors: selectors,
			buffer: buffer.New[*ctrlv1.InfraEvent](buffer.Config{
				Capacity: 1000,
				Drop:     true,
				Name:     fmt.Sprintf("ctrl_watch_events_%s", clientID),
			}),
			done: make(chan struct{}),
		}

		s.clients[clientID] = c
	}
	s.clientsMu.Unlock()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stream closed due to context cancellation",
				"error", ctx.Err(),
			)
			return nil
		case <-done:
			return nil

		case event := <-c.buffer.Consume():
			err := stream.Send(event)
			if err != nil {
				s.logger.Error("failed to send event", "error", err)
			}
		}

	}

}
