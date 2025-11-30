package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) GetDesiredState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredStateRequest], stream *connect.ServerStream[ctrlv1.InfraEvent]) error {

	clientID := req.Msg.GetClientId()
	selectors := req.Msg.GetSelectors()

	s.logger.Info("sync request received",
		"client_id", clientID,
		"selectors", selectors,
	)

	// missing labels means we accept all regions
	region := req.Msg.Selectors["region"]

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
				assert.True(t.K8sNamespace.Valid, "missing k8s namespace"),
				assert.NotEmpty(t.K8sNamespace.String, "missing k8s namespace"),
				assert.True(t.Image.Valid, "missing image"),
				assert.NotEmpty(t.Image.String, "missing image"),
			); err != nil {
				s.logger.Error("invalid configuration", "error", err.Error())
				continue
			}

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

	// gateways
	cursor = ""
	for {
		topologies, err := db.Query.ListDesiredGateways(ctx, s.db.RO(), db.ListDesiredGatewaysParams{
			Region:           region,
			DesiredState:     db.GatewaysDesiredStateRunning,
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
		cursor = topologies[len(topologies)-1].GatewayID

		for _, t := range topologies {

			if err := assert.All(
				assert.True(t.K8sNamespace.Valid, "missing k8s namespace"),
				assert.NotEmpty(t.K8sNamespace.String, "missing k8s namespace"),
			); err != nil {
				s.logger.Error("invalid configuration", "error", err.Error())
				continue
			}

			err = stream.Send(&ctrlv1.InfraEvent{
				Event: &ctrlv1.InfraEvent_GatewayEvent{
					GatewayEvent: &ctrlv1.GatewayEvent{
						Event: &ctrlv1.GatewayEvent_Apply{
							Apply: &ctrlv1.ApplyGateway{
								Namespace:     t.K8sNamespace.String,
								WorkspaceId:   t.WorkspaceID,
								EnvironmentId: t.EnvironmentID,
								ProjectId:     t.ProjectID,
								GatewayId:     t.GatewayID,
								Image:         t.Image,
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
	return nil

}
