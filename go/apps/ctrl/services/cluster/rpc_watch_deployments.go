package cluster

import (
	"context"
	"sync"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) WatchDeployments(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], stream *connect.ServerStream[ctrlv1.DeploymentState]) error {

	clientID := req.Msg.GetClientId()
	selectors := req.Msg.GetSelectors()

	s.logger.Info("watch deployments request received",
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

		synthetic := make(chan *ctrlv1.DeploymentState)

		wg.Go(func() {
			if err := s.getSyntheticDeployments(ctx, req, synthetic); err != nil {
				s.logger.Error("failed to get synthetic deployments", "error", err)
			}
			close(synthetic)
		})
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					s.logger.Info("stream closed due to context cancellation",
						"error", ctx.Err(),
					)
					return

				case event := <-synthetic:
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

				case event := <-c.deploymentStates.Consume():
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

func (s *Service) getSyntheticDeployments(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], c chan *ctrlv1.DeploymentState) error {

	clientID := req.Msg.GetClientId()
	selectors := req.Msg.GetSelectors()

	s.logger.Debug("get all sentinels request received",
		"client_id", clientID,
		"selectors", selectors,
	)

	// missing labels means we accept all regions
	region := req.Msg.GetSelectors()["region"]

	cursor := ""
	for {
		topologies, err := db.Query.ListDesiredDeploymentTopology(ctx, s.db.RO(), db.ListDesiredDeploymentTopologyParams{
			Region:           region,
			DesiredState:     db.DeploymentsDesiredStateRunning,
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
		cursor = topologies[len(topologies)-1].DeploymentID

		for _, t := range topologies {
			c <- &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: &ctrlv1.ApplyDeployment{
						Namespace:     t.K8sNamespace.String,
						K8SCrdName:    t.K8sCrdName,
						WorkspaceId:   t.WorkspaceID,
						EnvironmentId: t.EnvironmentID,
						ProjectId:     t.ProjectID,
						DeploymentId:  t.DeploymentID,
						Image:         t.Image.String,
						Replicas:      t.Replicas,
						CpuMillicores: int64(t.CpuMillicores),
						MemoryMib:     int64(t.MemoryMib),
					},
				},
			}

		}
	}
	return nil

}
