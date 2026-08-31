package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

// changePageSize is the number of rows fetched per page when syncing deployment changes.
const changePageSize = 10000

// errUnrecoverable marks a deployment_changes row that can never be processed
// successfully, no matter how often it is retried (for example a row whose
// state cannot be converted because of an unknown desired_status enum).
// Retrying would stall the stream on the same row forever, so callers skip
// such rows and advance the cursor.
var errUnrecoverable = errors.New("deployment change is unrecoverable")

// WatchDeploymentChanges streams incremental resource changes from the
// deployment_changes outbox table. When version_last_seen is 0, the server
// jumps to the current max pk and polls from there — it never replays
// historical changes.
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

	versionCursor := req.Msg.GetVersionLastSeen()

	// When version is 0 and replay is not requested, jump to the current max pk
	// so we only see new changes.
	if versionCursor == 0 && !req.Msg.GetReplay() {
		maxVersion, err := s.db.GetDeploymentChangesMaxVersion(ctx, cluster.RegionID)
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		versionCursor = uint64(maxVersion)
		logger.Info("watch: starting from max version", "region_id", cluster.RegionID, "cursor", versionCursor)
	}

	// Poll deployment_changes for new entries.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		events, err := s.fetchDeploymentChangeEvents(ctx, cluster.RegionID, versionCursor)
		if err != nil {
			logger.Error("failed to fetch deployment change events", "error", err)
			return connect.NewError(connect.CodeInternal, err)
		}

		for _, event := range events {
			if err := stream.Send(event); err != nil {
				return err
			}
			if event.GetVersion() > versionCursor {
				versionCursor = event.GetVersion()
			}
		}

		if len(events) == 0 {
			jitter := time.Duration(500+rand.IntN(1000)) * time.Millisecond
			time.Sleep(jitter)
		}
	}
}

// fetchDeploymentChangeEvents polls deployment_changes for new entries and does a
// point lookup for each row to load current state.
//
// Rows whose resource is gone (not found) or that can never be processed
// (errUnrecoverable) are skipped with a bare version event so the cursor
// advances past them. Any other error is transient and returned to the caller,
// which aborts the stream without advancing the cursor; the client reconnects
// and retries from its last seen version.
func (s *Service) fetchDeploymentChangeEvents(ctx context.Context, regionID string, afterVersion uint64) ([]*ctrlv1.DeploymentChangeEvent, error) {
	changes, err := s.db.ListDeploymentChangesByRegionAll(ctx, db.ListDeploymentChangesByRegionAllParams{
		RegionID:     regionID,
		AfterVersion: afterVersion,
		Limit:        changePageSize,
	})
	if err != nil {
		return nil, err
	}

	events := make([]*ctrlv1.DeploymentChangeEvent, 0, len(changes))
	for _, change := range changes {
		resourceType := string(change.ResourceType)
		event, err := s.loadChangeEvent(ctx, change)
		if err != nil {
			switch {
			case db.IsNotFound(err):
				// The resource is gone. The row is safe to skip.
				metrics.DeploymentChangesProcessedTotal.WithLabelValues(resourceType, "not_found").Inc()
			case errors.Is(err, errUnrecoverable):
				// Terminal per-row failure. Retrying would block the stream
				// on this row forever, so log loudly and skip it.
				metrics.DeploymentChangesProcessedTotal.WithLabelValues(resourceType, "error").Inc()
				logger.Error("skipping unrecoverable deployment change",
					"error", err,
					"resource_type", change.ResourceType,
					"resource_id", change.ResourceID,
				)
			default:
				// Transient failure (for example a DB error on the point
				// lookup). Do not advance past this row: abort the stream so
				// the client reconnects and retries from its last seen
				// version.
				return nil, fmt.Errorf("load state for deployment change pk=%d (%s %s): %w",
					change.Pk, change.ResourceType, change.ResourceID, err)
			}
			// Skip this row but keep advancing the cursor
			events = append(events, &ctrlv1.DeploymentChangeEvent{Version: change.Pk})
			continue
		}
		metrics.DeploymentChangesProcessedTotal.WithLabelValues(resourceType, "success").Inc()
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// loadChangeEvent does a point lookup for a single deployment_changes row based on resource_type.
// Uses the control plane connection because deployment_changes rows arrive
// immediately after the data is written.
func (s *Service) loadChangeEvent(ctx context.Context, change db.DeploymentChange) (*ctrlv1.DeploymentChangeEvent, error) {
	switch change.ResourceType {
	case db.DeploymentChangesResourceTypeDeploymentTopology:
		row, err := s.db.FindDeploymentTopologyByDeploymentAndRegion(ctx, db.FindDeploymentTopologyByDeploymentAndRegionParams{
			DeploymentID: change.ResourceID,
			RegionID:     change.RegionID,
		})
		if err != nil {
			return nil, err
		}
		state, err := deploymentRowToState(row, change.Pk)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errUnrecoverable, err)
		}
		if state == nil {
			return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil
		}
		return &ctrlv1.DeploymentChangeEvent{
			Version: change.Pk,
			Event:   &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: state},
		}, nil

	case db.DeploymentChangesResourceTypeCiliumNetworkPolicy:
		// Cilium resources are no longer dispatched — frontline took
		// over the request path. The outbox row exists during the
		// cutover so we just acknowledge it and advance the version.
		return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil

	case db.DeploymentChangesResourceTypeSentinel:
		// This legacy resource type is no longer dispatched. Frontline owns
		// the request path. The outbox row exists during the
		// cutover so we just acknowledge it and advance the version.
		return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil

	default:
		logger.Error("unknown resource type in deployment_changes", "resource_type", change.ResourceType)
		return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil
	}
}

// deploymentStateRow limits state conversion to the two deployment query results.
type deploymentStateRow interface {
	db.FindDeploymentTopologyByDeploymentAndRegionRow | db.ListAllDeploymentTopologiesByRegionRow
}

// deploymentRowToState converts either deployment query result to a proto DeploymentState message.
func deploymentRowToState[T deploymentStateRow](row T, version uint64) (*ctrlv1.DeploymentState, error) {
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
			Image:                         row.DeploymentImage,
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
			Version: version,
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
			Image:                         deployment.Image.String,
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
			Version: version,
			State: &ctrlv1.DeploymentState_Apply{
				Apply: apply,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown DeploymentTopologyDesiredStatus: %v", deployment.DesiredStatus)
	}
}
