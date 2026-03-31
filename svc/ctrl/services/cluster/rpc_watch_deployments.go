package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
)

// WatchDeployments streams deployment state changes from the control plane to agents.
// This is the primary mechanism for agents to receive desired state updates for their region.
// Agents apply received state to Kubernetes to converge actual state toward desired state.
//
// The stream uses version-based cursors for resumability. The client provides version_last_seen
// in the request, and the server streams all deployments with versions greater than that cursor.
// Clients should track the maximum version received and use it to reconnect without replaying
// history. When no new versions are available, the server polls the database every second.
//
// Each poll fetches up to 100 deployment topology rows ordered by version. The desired_status
// field determines whether to send an ApplyDeployment (for running) or
// DeleteDeployment (for stopped). Rows with unhandled statuses are logged
// and skipped.
//
// Returns when the context is cancelled, or on database or stream errors.
func (s *Service) WatchDeployments(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchDeploymentsRequest],
	stream *connect.ServerStream[ctrlv1.DeploymentState],
) error {
	if err := s.authenticate(req); err != nil {
		return err
	}

	regionName := req.Header().Get("X-Krane-Region")
	platform := req.Header().Get("X-Krane-Platform")
	if err := assert.All(
		assert.NotEmpty(regionName, "region is required"),
		assert.NotEmpty(platform, "platform is required"),
	); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	logger.Info("starting WatchDeployments stream", "region_name", regionName, "platform", platform)

	region, err := db.Query.FindRegionByNameAndPlatform(ctx, s.db.RO(), db.FindRegionByNameAndPlatformParams{
		Name:     regionName,
		Platform: platform,
	})
	if err != nil {
		logger.Error("failed to find region for WatchDeployments", "error", err, "region_name", regionName, "platform", platform)
		return connect.NewError(connect.CodeInternal, err)
	}

	logger.Info("found region for WatchDeployments", "region_id", region.ID)

	versionCursor := req.Msg.GetVersionLastSeen()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		states, err := s.fetchDeploymentStates(ctx, region.ID, versionCursor)
		if err != nil {
			logger.Error("failed to fetch deployment states", "error", err)
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

// fetchDeploymentStates queries the database for deployment topologies in the given region
// with versions greater than afterVersion, returning up to 100 results. Rows that fail
// conversion are logged and skipped rather than failing the entire batch.
func (s *Service) fetchDeploymentStates(ctx context.Context, regionID string, afterVersion uint64) ([]*ctrlv1.DeploymentState, error) {
	rows, err := db.Query.ListDeploymentTopologyByRegion(ctx, s.db.RO(), db.ListDeploymentTopologyByRegionParams{
		RegionID:     regionID,
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
			logger.Error("failed to convert deployment row to state", "error", err, "deploymentId", row.Deployment.ID)
			continue
		}
		states = append(states, state)
	}

	logger.Info("fetched deployment states", "count", len(states), "region_id", regionID, "after_version", afterVersion)

	return states, nil
}

// deploymentRowToState converts a database row to a proto DeploymentState message. Returns
// a DeleteDeployment for stopped and an ApplyDeployment for running
// statuses. Returns (nil, nil) for unhandled statuses, which the caller should skip.
func (s *Service) deploymentRowToState(row db.ListDeploymentTopologyByRegionRow) (*ctrlv1.DeploymentState, error) {
	switch row.DeploymentTopology.DesiredStatus {
	case db.DeploymentTopologyDesiredStatusStopped:
		return &ctrlv1.DeploymentState{
			Version: row.DeploymentTopology.Version,
			State: &ctrlv1.DeploymentState_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					K8SNamespace: row.K8sNamespace.String,
					K8SName:      row.Deployment.K8sName,
				},
			},
		}, nil
	case db.DeploymentTopologyDesiredStatusRunning:
		var buildID *string
		if row.Deployment.BuildID.Valid {
			buildID = &row.Deployment.BuildID.String
		}

		apply := &ctrlv1.ApplyDeployment{
			DeploymentId:                  row.Deployment.ID,
			K8SNamespace:                  row.K8sNamespace.String,
			K8SName:                       row.Deployment.K8sName,
			WorkspaceId:                   row.Deployment.WorkspaceID,
			ProjectId:                     row.Deployment.ProjectID,
			EnvironmentId:                 row.Deployment.EnvironmentID,
			AppId:                         row.Deployment.AppID,
			Replicas:                      row.DeploymentTopology.DesiredReplicas,
			Image:                         row.Deployment.Image.String,
			CpuMillicores:                 int64(row.Deployment.CpuMillicores),
			MemoryMib:                     int64(row.Deployment.MemoryMib),
			EncryptedEnvironmentVariables: row.Deployment.EncryptedEnvironmentVariables,
			BuildId:                       buildID,
			Command:                       row.Deployment.Command,
			Port:                          row.Deployment.Port,
			ShutdownSignal:                string(row.Deployment.ShutdownSignal),
			EnvironmentSlug:               &row.EnvironmentSlug,
			Region:                        &row.RegionName,
		}

		if row.Deployment.GitCommitSha.Valid {
			apply.GitCommitSha = &row.Deployment.GitCommitSha.String
		}
		if row.Deployment.GitBranch.Valid {
			apply.GitBranch = &row.Deployment.GitBranch.String
		}
		if row.Deployment.GitCommitMessage.Valid {
			apply.GitCommitMessage = &row.Deployment.GitCommitMessage.String
		}
		if row.GitRepo.Valid {
			apply.GitRepo = &row.GitRepo.String
		}

		if row.Deployment.Healthcheck.Valid {
			hcBytes, err := json.Marshal(row.Deployment.Healthcheck.Healthcheck)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal healthcheck: %w", err)
			}
			apply.Healthcheck = hcBytes
		}

		dt := row.DeploymentTopology
		policy := &ctrlv1.AutoscalingPolicy{
			MinReplicas: dt.AutoscalingReplicasMin,
			MaxReplicas: dt.AutoscalingReplicasMax,
		}
		if dt.AutoscalingThresholdCpu.Valid {
			policy.CpuThreshold = ptr.P(int32(dt.AutoscalingThresholdCpu.Int16))
		}
		if dt.AutoscalingThresholdMemory.Valid {
			policy.MemoryThreshold = ptr.P(int32(dt.AutoscalingThresholdMemory.Int16))
		}
		apply.Autoscaling = policy

		return &ctrlv1.DeploymentState{
			Version: row.DeploymentTopology.Version,
			State: &ctrlv1.DeploymentState_Apply{
				Apply: apply,
			},
		}, nil
	default:
		logger.Error("unhandled deployment topology desired status", "status", row.DeploymentTopology.DesiredStatus)
		return nil, nil
	}
}
