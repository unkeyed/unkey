package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploymentstream"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

// changePageSize is the number of rows fetched per page when syncing deployment changes.
const changePageSize = 10000

// WatchDeploymentChanges opens one region-filtered VStream for this watch.
// Checkpoint-only events follow all state events for a committed transaction.
func (s *Service) WatchDeploymentChanges(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchDeploymentChangesRequest],
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
) error {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return err
	}

	cluster, err := s.resolveCluster(ctx, req.Msg.GetCluster())
	if err != nil {
		return err
	}

	if s.deploymentStream == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("deployment VStream is not configured"))
	}
	token := req.Msg.GetResumeToken()
	if req.Msg.GetReplay() {
		token = nil
	}
	err = s.deploymentStream.Watch(ctx, cluster.RegionID, token, func(deploymentID string) error {
		row, err := s.db.FindDeploymentTopologyByDeploymentAndRegion(ctx, db.FindDeploymentTopologyByDeploymentAndRegionParams{
			DeploymentID: deploymentID,
			RegionID:     cluster.RegionID,
		})
		if db.IsNotFound(err) {
			// Krane's per-ReplicaSet reconciliation removes resources absent from desired state.
			metrics.DeploymentChangesProcessedTotal.WithLabelValues("deployment_topology", "not_found").Inc()
			return nil
		}
		if err != nil {
			metrics.DeploymentChangesProcessedTotal.WithLabelValues("deployment_topology", "error").Inc()
			return err
		}
		state, err := deploymentRowToState(row)
		if err != nil {
			return err
		}
		if err := stream.Send(&ctrlv1.DeploymentChangeEvent{Event: &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: state}}); err != nil {
			return err
		}
		metrics.DeploymentChangesProcessedTotal.WithLabelValues("deployment_topology", "success").Inc()
		return nil
	}, func(next []byte) error {
		return stream.Send(&ctrlv1.DeploymentChangeEvent{ResumeToken: next})
	})
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if err != nil {
		logger.Error("deployment VStream ended", "region_id", cluster.RegionID, "error", err)
	}
	if errors.Is(err, deploymentstream.ErrInvalidToken) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, deploymentstream.ErrExpired) {
		return connect.NewError(connect.CodeOutOfRange, err)
	}
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

// deploymentStateRow limits state conversion to the two deployment query results.
type deploymentStateRow interface {
	db.FindDeploymentTopologyByDeploymentAndRegionRow | db.ListAllDeploymentTopologiesByRegionRow
}

// deploymentRowToState converts either deployment query result to a proto DeploymentState message.
func deploymentRowToState[T deploymentStateRow](row T) (*ctrlv1.DeploymentState, error) {
	var deployment db.FindDeploymentTopologyByDeploymentAndRegionRow
	switch row := any(row).(type) {
	case db.FindDeploymentTopologyByDeploymentAndRegionRow:
		deployment = row
	case db.ListAllDeploymentTopologiesByRegionRow:
		deployment = db.FindDeploymentTopologyByDeploymentAndRegionRow{
			DesiredStatus:                 row.TopologyDesiredStatus,
			AutoscalingReplicasMin:        row.TopologyAutoscalingReplicasMin,
			AutoscalingReplicasMax:        row.TopologyAutoscalingReplicasMax,
			AutoscalingThresholdCpu:       row.TopologyAutoscalingThresholdCpu,
			AutoscalingThresholdMemory:    row.TopologyAutoscalingThresholdMemory,
			ID:                            row.DeploymentID,
			K8sName:                       row.DeploymentK8sName,
			WorkspaceID:                   row.DeploymentWorkspaceID,
			ProjectID:                     row.DeploymentProjectID,
			EnvironmentID:                 row.DeploymentEnvironmentID,
			AppID:                         row.DeploymentAppID,
			ImageResolved:                 row.DeploymentImageResolved,
			BuildID:                       row.DeploymentBuildID,
			GitCommitSha:                  row.DeploymentGitCommitSha,
			GitBranch:                     row.DeploymentGitBranch,
			GitCommitMessage:              row.DeploymentGitCommitMessage,
			CpuMillicores:                 row.DeploymentCpuMillicores,
			MemoryMib:                     row.DeploymentMemoryMib,
			StorageMib:                    row.DeploymentStorageMib,
			EncryptedEnvironmentVariables: row.DeploymentEncryptedEnvironmentVariables,
			Command:                       row.DeploymentCommand,
			Port:                          row.DeploymentPort,
			ShutdownSignal:                row.DeploymentShutdownSignal,
			Healthcheck:                   row.DeploymentHealthcheck,
			K8sNamespace:                  row.K8sNamespace,
			EnvironmentSlug:               row.EnvironmentSlug,
			RegionName:                    row.RegionName,
			GitRepo:                       row.GitRepo,
		}
	default:
		return nil, fmt.Errorf("unsupported deployment row type %T", row)
	}

	switch deployment.DesiredStatus {
	case db.DeploymentTopologyDesiredStatusStopped:
		return &ctrlv1.DeploymentState{
			State: &ctrlv1.DeploymentState_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					K8SNamespace: deployment.K8sNamespace.String,
					K8SName:      deployment.K8sName,
				},
			},
		}, nil
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
			Image:                         deployment.ImageResolved.String,
			CpuMillicores:                 int64(deployment.CpuMillicores),
			MemoryMib:                     int64(deployment.MemoryMib),
			EncryptedEnvironmentVariables: deployment.EncryptedEnvironmentVariables,
			BuildId:                       buildID,
			Command:                       deployment.Command,
			Port:                          deployment.Port,
			ShutdownSignal:                string(deployment.ShutdownSignal),
			EnvironmentSlug:               &deployment.EnvironmentSlug,
			Region:                        &deployment.RegionName,
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
				return nil, fmt.Errorf("failed to marshal healthcheck: %w", err)
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

		return &ctrlv1.DeploymentState{
			State: &ctrlv1.DeploymentState_Apply{
				Apply: apply,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown DeploymentTopologyDesiredStatus: %v", deployment.DesiredStatus)
	}
}
