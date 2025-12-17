package cluster

import (
	"context"
	"sync"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) WatchSentinels(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], stream *connect.ServerStream[ctrlv1.SentinelState]) error {

	clientID := req.Msg.GetClientId()
	selectors := req.Msg.GetSelectors()

	s.logger.Info("watch sentinels request received",
		"client_id", clientID,
		"selectors", selectors,
	)

	s.clientsMu.Lock()

	c, ok := s.clients[clientID]
	if !ok {
		c = newClient(clientID, selectors)

		s.clients[clientID] = c
	}
	s.clientsMu.Unlock()

	wg := sync.WaitGroup{}

	if req.Msg.GetSynthetic() {

		synthetic := make(chan *ctrlv1.SentinelState)

		closeChannel := sync.OnceFunc(func() {
			close(synthetic)
		})

		wg.Go(func() {
			if err := s.getSyntheticSentinels(ctx, req, synthetic); err != nil {
				s.logger.Error("failed to get synthetic sentinels", "error", err)
			}
			closeChannel()

		})
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					s.logger.Info("stream closed due to context cancellation",
						"error", ctx.Err(),
					)
					return

				case event, ok := <-synthetic:
					if !ok {
						return
					}
					s.logger.Info("Sending Synthetic Event", "event", event)
					err := stream.Send(event)
					if err != nil {
						s.logger.Error("failed to send event", "error", err)
					}
				}
			}
		})
	}
	if req.Msg.GetLive() {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					s.logger.Info("stream closed due to context cancellation",
						"error", ctx.Err(),
					)
					return

				case event := <-c.sentinelStates.Consume():
					err := stream.Send(event)
					if err != nil {
						s.logger.Error("failed to send event", "error", err)
					}
				}
			}
		})
	}

	wg.Wait()
	return nil

}

func (s *Service) getSyntheticSentinels(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], c chan *ctrlv1.SentinelState) error {

	// missing labels means we accept all regions
	region := req.Msg.GetSelectors()["region"]

	cursor := ""
	for {
		topologies, err := db.Query.ListDesiredSentinels(ctx, s.db.RO(), db.ListDesiredSentinelsParams{
			Region:           region,
			DesiredState:     db.SentinelsDesiredStateRunning,
			PaginationCursor: cursor,
			Limit:            1000,
		})
		if err != nil {
			s.logger.Error("failed to get topologies", "error", err.Error())
			return err
		}

		if len(topologies) == 0 {
			break
		}
		s.logger.Info("Found Sentinels", "topologies", topologies)
		cursor = topologies[len(topologies)-1].Sentinel.ID

		for _, t := range topologies {
			c <- &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Apply{
					Apply: &ctrlv1.ApplySentinel{
						Namespace:     t.Workspace.K8sNamespace.String,
						K8SCrdName:    t.Sentinel.K8sCrdName,
						WorkspaceId:   t.Workspace.ID,
						EnvironmentId: t.Sentinel.EnvironmentID,
						ProjectId:     t.Sentinel.ProjectID,
						SentinelId:    t.Sentinel.ID,
						Image:         t.Sentinel.Image,
						Replicas:      t.Sentinel.Replicas,
						CpuMillicores: int64(t.Sentinel.CpuMillicores),
						MemoryMib:     int64(t.Sentinel.MemoryMib),
					},
				},
			}

		}
	}
	return nil

}
