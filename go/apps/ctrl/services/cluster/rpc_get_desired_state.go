package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) GetDesiredState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredStateRequest], stream *connect.ServerStream[ctrlv1.InfraEvent]) error {

	if err := s.authenticate(req); err != nil {
		return err
	}

	clientID := req.Msg.GetClientId()
	selectors := req.Msg.GetSelectors()

	s.logger.Debug("sync request received",
		"client_id", clientID,
		"selectors", selectors,
	)

	// missing labels means we accept all regions
	region := req.Msg.GetSelectors()["region"]

	// deployments
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
			return connect.NewError(connect.CodeInternal, err)
		}

		if len(topologies) == 0 {
			break
		}
		cursor = topologies[len(topologies)-1].DeploymentID

		for _, t := range topologies {

			if err := assert.All(
				assert.True(t.Image.Valid, "missing image"),
				assert.NotEmpty(t.Image.String, "missing image"),
			); err != nil {
				s.logger.Error("invalid configuration", "error", err.Error())
				continue
			}

			s.logger.Info("sending deployment event", "t", t)

			err = stream.Send(&ctrlv1.InfraEvent{
				Event: &ctrlv1.InfraEvent_DeploymentEvent{
					DeploymentEvent: &ctrlv1.DeploymentEvent{
						Event: &ctrlv1.DeploymentEvent_Apply{
							Apply: &ctrlv1.ApplyDeployment{
								Namespace:     t.K8sNamespace.String,
								WorkspaceId:   t.WorkspaceID,
								ProjectId:     t.ProjectID,
								EnvironmentId: t.EnvironmentID,
								DeploymentId:  t.DeploymentID,
								Image:         t.Image.String,
								Replicas:      uint32(t.Replicas),
								CpuMillicores: uint32(t.CpuMillicores),
								MemorySizeMib: uint32(t.MemoryMib),
							},
						},
					},
				},
			})
			if err != nil {
				s.logger.Error("failed to send event", "error", err.Error())
				return connect.NewError(connect.CodeInternal, err)
			}

		}
	}

	// sentinels
	cursor = ""
	for {
		topologies, err := db.Query.ListDesiredSentinels(ctx, s.db.RO(), db.ListDesiredSentinelsParams{
			Region:           region,
			DesiredState:     db.SentinelsDesiredStateRunning,
			PaginationCursor: cursor,
			Limit:            1000,
		})
		if err != nil {
			s.logger.Error("failed to get topologies", "error", err.Error())
			return connect.NewError(connect.CodeInternal, err)
		}

		if len(topologies) == 0 {
			break
		}
		cursor = topologies[len(topologies)-1].Sentinel.ID

		for _, t := range topologies {

			err = stream.Send(&ctrlv1.InfraEvent{
				Event: &ctrlv1.InfraEvent_SentinelEvent{
					SentinelEvent: &ctrlv1.SentinelEvent{
						Event: &ctrlv1.SentinelEvent_Apply{
							Apply: &ctrlv1.ApplySentinel{
								Namespace:     t.Workspace.K8sNamespace.String,
								K8SCrdName:    t.Sentinel.K8sCrdName,
								WorkspaceId:   t.Workspace.ID,
								EnvironmentId: t.Sentinel.EnvironmentID,
								ProjectId:     t.Sentinel.ProjectID,
								SentinelId:    t.Sentinel.ID,
								Image:         t.Sentinel.Image,
								Replicas:      uint32(t.Sentinel.DesiredReplicas),
								CpuMillicores: uint32(t.Sentinel.CpuMillicores),
								MemorySizeMib: uint32(t.Sentinel.MemoryMib),
							},
						},
					},
				},
			})
			if err != nil {
				s.logger.Error("failed to send event", "error", err.Error())
				return connect.NewError(connect.CodeInternal, err)
			}

		}
	}
	return nil

}
