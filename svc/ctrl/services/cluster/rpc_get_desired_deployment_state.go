package cluster

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
)

// GetDesiredDeploymentState returns the target state for a single deployment in the caller's
// region. This is a point query alternative to [Service.WatchDeployments] for cases where
// an agent needs to fetch state for a specific deployment rather than streaming all changes.
//
// The response contains either an ApplyDeployment (for running state) or DeleteDeployment
// (for archived or standby states) based on the deployment_topology's desired_state in the
// database. Unhandled desired states result in CodeInternal.
//
// Returns CodeUnauthenticated if bearer token is invalid, CodeInvalidArgument if
// region or platform is missing, CodeNotFound if no deployment exists with the given
// ID in the specified region, or CodeInternal for database errors or unhandled states.
func (s *Service) GetDesiredDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {

	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	region := req.Header().Get("X-Krane-Region")
	platform := req.Header().Get("X-Krane-Platform")
	if err := assert.All(
		assert.NotEmpty(region, "region is required"),
		assert.NotEmpty(platform, "platform is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	deployment, err := db.Query.FindDeploymentTopologyByIDAndRegion(ctx, s.db.RO(), db.FindDeploymentTopologyByIDAndRegionParams{
		DeploymentID: req.Msg.GetDeploymentId(),
		Region:       region,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	switch deployment.DesiredStatus {
	case db.DeploymentTopologyDesiredStatusStopped:
		return connect.NewResponse(&ctrlv1.DeploymentState{
			State: &ctrlv1.DeploymentState_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					K8SNamespace: deployment.K8sNamespace.String,
					K8SName:      deployment.K8sName,
				},
			},
		}), nil
	case db.DeploymentTopologyDesiredStatusRunning:

		var buildID *string
		if deployment.BuildID.Valid {
			buildID = &deployment.BuildID.String
		}

		apply := &ctrlv1.ApplyDeployment{
			DeploymentId:                  deployment.ID,
			K8SNamespace:                  deployment.K8sNamespace.String,
			K8SName:                       deployment.K8sName,
			WorkspaceId:                   deployment.WorkspaceID,
			ProjectId:                     deployment.ProjectID,
			EnvironmentId:                 deployment.EnvironmentID,
			AppId:                         deployment.AppID,
			Image:                         deployment.Image.String,
			CpuMillicores:                 int64(deployment.CpuMillicores),
			MemoryMib:                     int64(deployment.MemoryMib),
			EncryptedEnvironmentVariables: deployment.EncryptedEnvironmentVariables,
			BuildId:                       buildID,
			Command:                       deployment.Command,
			Port:                          deployment.Port,
			ShutdownSignal:                string(deployment.ShutdownSignal),
			EnvironmentSlug:               &deployment.EnvironmentSlug,
			Region:                        &deployment.Region,
		}

		if deployment.GitCommitSha.Valid {
			apply.GitCommitSha = &deployment.GitCommitSha.String
		}
		if deployment.GitBranch.Valid {
			apply.GitBranch = &deployment.GitBranch.String
		}
		if deployment.GitCommitMessage.Valid {
			apply.GitCommitMessage = &deployment.GitCommitMessage.String
		}
		if deployment.GitRepo.Valid {
			apply.GitRepo = &deployment.GitRepo.String
		}

		if deployment.Healthcheck.Valid {
			hcBytes, err := json.Marshal(deployment.Healthcheck.Healthcheck)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal healthcheck: %w", err))
			}
			apply.Healthcheck = hcBytes
		}

		policy := &ctrlv1.AutoscalingPolicy{
			MinReplicas: deployment.AutoscalingReplicasMin,
			MaxReplicas: deployment.AutoscalingReplicasMax,
		}
		if deployment.AutoscalingThresholdCpu.Valid {
			policy.CpuThreshold = ptr.P(int32(deployment.AutoscalingThresholdCpu.Int16))
		}
		if deployment.AutoscalingThresholdMemory.Valid {
			policy.MemoryThreshold = ptr.P(int32(deployment.AutoscalingThresholdMemory.Int16))
		}
		apply.Autoscaling = policy

		if deployment.StorageMib > 0 {
			apply.EphemeralStorage = &ctrlv1.EphemeralStorage{
				SizeMib: int64(deployment.StorageMib),
			}
		}

		return connect.NewResponse(&ctrlv1.DeploymentState{
			State: &ctrlv1.DeploymentState_Apply{
				Apply: apply,
			},
		}), nil
	default:
		logger.Error("unhandled deployment topology desired status", "desiredStatus", deployment.DesiredStatus)
	}

	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unhandled deployment topology desired status: %s", deployment.DesiredStatus))
}
