package cluster

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
)

// WatchDeployments streams deployment state changes from the control plane to agents.
// Each deployment controller maintains its own version cursor for resumable streaming.
// The agent applies received state to Kubernetes to converge actual state toward desired state.
//
// This is a long-lived streaming RPC. The server polls the database for new deployment
// versions and streams them to the client. The client should track the max version seen
// and reconnect with that version to resume from where it left off.
func (s *Service) WatchDeployments(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchDeploymentsRequest],
	stream *connect.ServerStream[ctrlv1.DeploymentState],
) error {
	if err := s.authenticate(req); err != nil {
		return err
	}

	region := req.Msg.GetRegion()
	if err := assert.NotEmpty(region, "region is required"); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	versionCursor := req.Msg.GetVersionLastSeen()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		states, err := s.fetchDeploymentStates(ctx, region, versionCursor)
		if err != nil {
			s.logger.Error("failed to fetch deployment states", "error", err)
			return connect.NewError(connect.CodeInternal, err)
		}

		for _, state := range states {
			if err := stream.Send(state); err != nil {
				return err
			}
			if state.GetVersion() > versionCursor {
				versionCursor = state.GetVersion()
			}
		}

		if len(states) == 0 {
			time.Sleep(time.Second)
		}
	}
}

func (s *Service) fetchDeploymentStates(ctx context.Context, region string, afterVersion uint64) ([]*ctrlv1.DeploymentState, error) {
	rows, err := db.Query.ListDeploymentTopologyByRegion(ctx, s.db.RO(), db.ListDeploymentTopologyByRegionParams{
		Region:       region,
		Afterversion: afterVersion,
		Limit:        100,
	})
	if err != nil {
		return nil, err
	}

	states := make([]*ctrlv1.DeploymentState, 0, len(rows))
	for _, row := range rows {
		state, err := s.deploymentRowToState(row)
		if err != nil {
			s.logger.Error("failed to convert deployment row to state", "error", err, "deploymentId", row.Deployment.ID)
			continue
		}
		states = append(states, state)
	}

	return states, nil
}

func (s *Service) deploymentRowToState(row db.ListDeploymentTopologyByRegionRow) (*ctrlv1.DeploymentState, error) {
	switch row.DeploymentTopology.DesiredStatus {
	case db.DeploymentTopologyDesiredStatusStopped, db.DeploymentTopologyDesiredStatusStopping:
		return &ctrlv1.DeploymentState{
			Version: row.DeploymentTopology.Version,
			State: &ctrlv1.DeploymentState_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					K8SNamespace: row.K8sNamespace.String,
					K8SName:      row.Deployment.K8sName,
				},
			},
		}, nil
	case db.DeploymentTopologyDesiredStatusStarted, db.DeploymentTopologyDesiredStatusStarting:
		var buildID *string
		if row.Deployment.BuildID.Valid {
			buildID = &row.Deployment.BuildID.String
		}

		return &ctrlv1.DeploymentState{
			Version: row.DeploymentTopology.Version,
			State: &ctrlv1.DeploymentState_Apply{
				Apply: &ctrlv1.ApplyDeployment{
					DeploymentId:                  row.Deployment.ID,
					K8SNamespace:                  row.K8sNamespace.String,
					K8SName:                       row.Deployment.K8sName,
					WorkspaceId:                   row.Deployment.WorkspaceID,
					ProjectId:                     row.Deployment.ProjectID,
					EnvironmentId:                 row.Deployment.EnvironmentID,
					Replicas:                      row.DeploymentTopology.DesiredReplicas,
					Image:                         row.Deployment.Image.String,
					CpuMillicores:                 int64(row.Deployment.CpuMillicores),
					MemoryMib:                     int64(row.Deployment.MemoryMib),
					EncryptedEnvironmentVariables: row.Deployment.EncryptedEnvironmentVariables,
					BuildId:                       buildID,
				},
			},
		}, nil
	default:
		s.logger.Error("unhandled deployment topology desired status", "status", row.DeploymentTopology.DesiredStatus)
		return nil, nil
	}
}
