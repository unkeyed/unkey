package cluster

import (
	"context"
	"sync"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func (s *Service) Watch(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], stream *connect.ServerStream[ctrlv1.State]) error {
	region := req.Msg.GetRegion()
	clusterID := req.Msg.GetClusterId()

	s.logger.Info("watch request received",
		"region", region,
		"clusterID", clusterID,
	)

	wg := sync.WaitGroup{}
	if req.Msg.GetSynthetic() {
		wg.Go(func() {
			if err := s.getSyntheticDeployments(ctx, req, stream); err != nil {
				s.logger.Error("failed to get synthetic deployments", "error", err)
			}
		})

		wg.Go(func() {
			if err := s.getSyntheticSentinels(ctx, req, stream); err != nil {
				s.logger.Error("failed to get synthetic sentinels", "error", err)
			}
		})
	}

	if req.Msg.GetLive() {
		connID := uid.New(uid.ConnectionPrefix)

		s.clientsMu.Lock()
		if s.clients[clusterID] == nil {
			s.clients[clusterID] = make(map[string]*client)
		}

		s.clients[clusterID][connID] = newClient(clusterID, connID, region, stream)
		s.logger.Info("creating new client", "clusterID", clusterID, "connID", connID, "totalConnections", len(s.clients[clusterID]))
		s.clientsMu.Unlock()

		defer func() {
			s.clientsMu.Lock()
			delete(s.clients[clusterID], connID)
			if len(s.clients[clusterID]) == 0 {
				delete(s.clients, clusterID)
			}
			s.logger.Info("deleted client", "clusterID", clusterID, "connID", connID)
			s.clientsMu.Unlock()
		}()
	}

	wg.Wait()
	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) getSyntheticSentinels(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], stream *connect.ServerStream[ctrlv1.State]) error {
	clusterID := req.Msg.GetClusterId()
	region := req.Msg.GetRegion()

	s.logger.Debug("get all sentinels request received",
		"cluster_id", clusterID,
		"region", region,
	)

	cursor := ""
	for {
		sentinels, err := db.Query.ListDesiredSentinels(ctx, s.db.RO(), db.ListDesiredSentinelsParams{
			Region:           region,
			DesiredState:     db.SentinelsDesiredStateRunning,
			PaginationCursor: cursor,
			Limit:            100,
		})

		if err != nil {
			s.logger.Error("failed to get sentinels", "error", err.Error())
			return err
		}

		if len(sentinels) == 0 {
			break
		}
		cursor = sentinels[len(sentinels)-1].ID

		for _, s := range sentinels {
			err = stream.Send(&ctrlv1.State{
				AcknowledgeId: nil,
				Kind: &ctrlv1.State_Sentinel{
					Sentinel: &ctrlv1.SentinelState{
						State: &ctrlv1.SentinelState_Apply{
							Apply: &ctrlv1.ApplySentinel{
								K8SName:       s.K8sName,
								WorkspaceId:   s.WorkspaceID,
								EnvironmentId: s.EnvironmentID,
								ProjectId:     s.ProjectID,
								SentinelId:    s.ID,
								Image:         s.Image,
								Replicas:      s.DesiredReplicas,
								CpuMillicores: int64(s.CpuMillicores),
								MemoryMib:     int64(s.MemoryMib),
							},
						},
					},
				},
			})

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) getSyntheticDeployments(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], stream *connect.ServerStream[ctrlv1.State]) error {
	clusterID := req.Msg.GetClusterId()
	region := req.Msg.GetRegion()

	s.logger.Debug("get all sentinels request received",
		"cluster_id", clusterID,
		"region", region,
	)

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
			var buildID *string
			if t.BuildID.Valid {
				buildID = &t.BuildID.String
			}

			err = stream.Send(&ctrlv1.State{
				AcknowledgeId: nil,
				Kind: &ctrlv1.State_Deployment{
					Deployment: &ctrlv1.DeploymentState{
						State: &ctrlv1.DeploymentState_Apply{
							Apply: &ctrlv1.ApplyDeployment{
								K8SNamespace:                  t.K8sNamespace.String,
								K8SName:                       t.K8sName,
								WorkspaceId:                   t.WorkspaceID,
								EnvironmentId:                 t.EnvironmentID,
								ProjectId:                     t.ProjectID,
								DeploymentId:                  t.DeploymentID,
								Image:                         t.Image.String,
								Replicas:                      t.DesiredReplicas,
								CpuMillicores:                 int64(t.CpuMillicores),
								MemoryMib:                     int64(t.MemoryMib),
								EncryptedEnvironmentVariables: t.EncryptedEnvironmentVariables,
								ReadinessId:                   nil,
								BuildId:                       buildID,
							},
						},
					},
				},
			})

			if err != nil {
				return err
			}
		}
	}

	return nil
}
