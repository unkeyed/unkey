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
		maxVersion, err := s.db.GetDeploymentChangesMaxVersion(ctx, cluster.Region.ID)
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		versionCursor = uint64(maxVersion)
		logger.Info("watch: starting from max version", "region_id", cluster.Region.ID, "cursor", versionCursor)
	}

	// Poll deployment_changes for new entries.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		events, err := s.fetchDeploymentChangeEvents(ctx, cluster.Region.ID, versionCursor)
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

// deploymentRowFromFullSync flattens embedded full-sync models into the point-lookup
// row so both query paths share one DB-to-proto conversion.
func deploymentRowFromFullSync(row db.ListAllDeploymentTopologiesByRegionRow) db.FindDeploymentTopologyByDeploymentAndRegionRow {
	return db.FindDeploymentTopologyByDeploymentAndRegionRow{
		DesiredStatus:                 row.DeploymentTopology.DesiredStatus,
		AutoscalingReplicasMin:        row.DeploymentTopology.AutoscalingReplicasMin,
		AutoscalingReplicasMax:        row.DeploymentTopology.AutoscalingReplicasMax,
		AutoscalingThresholdCpu:       row.DeploymentTopology.AutoscalingThresholdCpu,
		AutoscalingThresholdMemory:    row.DeploymentTopology.AutoscalingThresholdMemory,
		ID:                            row.Deployment.ID,
		K8sName:                       row.Deployment.K8sName,
		WorkspaceID:                   row.Deployment.WorkspaceID,
		ProjectID:                     row.Deployment.ProjectID,
		EnvironmentID:                 row.Deployment.EnvironmentID,
		AppID:                         row.Deployment.AppID,
		Image:                         row.Deployment.Image,
		BuildID:                       row.Deployment.BuildID,
		GitCommitSha:                  row.Deployment.GitCommitSha,
		GitBranch:                     row.Deployment.GitBranch,
		GitCommitMessage:              row.Deployment.GitCommitMessage,
		CpuMillicores:                 row.Deployment.CpuMillicores,
		MemoryMib:                     row.Deployment.MemoryMib,
		StorageMib:                    row.Deployment.StorageMib,
		EncryptedEnvironmentVariables: row.Deployment.EncryptedEnvironmentVariables,
		Command:                       row.Deployment.Command,
		Port:                          row.Deployment.Port,
		ShutdownSignal:                row.Deployment.ShutdownSignal,
		Healthcheck:                   row.Deployment.Healthcheck,
		K8sNamespace:                  row.K8sNamespace,
		EnvironmentSlug:               row.EnvironmentSlug,
		RegionName:                    row.RegionName,
		GitRepo:                       row.GitRepo,
	}
}

// deploymentRowToState converts a deployment row to a proto DeploymentState message.
func deploymentRowToState(row db.FindDeploymentTopologyByDeploymentAndRegionRow, version uint64) (*ctrlv1.DeploymentState, error) {
	switch row.DesiredStatus {
	case db.DeploymentTopologyDesiredStatusStopped:
		return &ctrlv1.DeploymentState{
			Version: version,
			State: &ctrlv1.DeploymentState_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					K8SNamespace: row.K8sNamespace.String,
					K8SName:      row.K8sName,
				},
			},
		}, nil
	case db.DeploymentTopologyDesiredStatusRunning:
		var buildID *string
		if row.BuildID.Valid {
			buildID = &row.BuildID.String
		}

		apply := &ctrlv1.ApplyDeployment{
			DeploymentId:                  row.ID,
			K8SNamespace:                  row.K8sNamespace.String,
			K8SName:                       row.K8sName,
			WorkspaceId:                   row.WorkspaceID,
			ProjectId:                     row.ProjectID,
			EnvironmentId:                 row.EnvironmentID,
			AppId:                         row.AppID,
			Image:                         row.Image.String,
			CpuMillicores:                 int64(row.CpuMillicores),
			MemoryMib:                     int64(row.MemoryMib),
			EncryptedEnvironmentVariables: row.EncryptedEnvironmentVariables,
			BuildId:                       buildID,
			Command:                       row.Command,
			Port:                          row.Port,
			ShutdownSignal:                string(row.ShutdownSignal),
			EnvironmentSlug:               &row.EnvironmentSlug,
			Region:                        &row.RegionName,
		}

		if row.GitCommitSha.Valid {
			apply.GitCommitSha = &row.GitCommitSha.String
		}
		if row.GitBranch.Valid {
			apply.GitBranch = &row.GitBranch.String
		}
		if row.GitCommitMessage.Valid {
			apply.GitCommitMessage = &row.GitCommitMessage.String
		}
		if row.GitRepo.Valid {
			apply.GitRepo = &row.GitRepo.String
		}

		if row.Healthcheck.Valid {
			hcBytes, err := json.Marshal(row.Healthcheck.Healthcheck)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal healthcheck: %w", err)
			}
			apply.Healthcheck = hcBytes
		}

		policy := &ctrlv1.AutoscalingPolicy{
			MinReplicas: row.AutoscalingReplicasMin,
			MaxReplicas: row.AutoscalingReplicasMax,
		}
		if row.AutoscalingThresholdCpu.Valid {
			policy.CpuThreshold = ptr.P(int32(row.AutoscalingThresholdCpu.Int16))
		}
		if row.AutoscalingThresholdMemory.Valid {
			policy.MemoryThreshold = ptr.P(int32(row.AutoscalingThresholdMemory.Int16))
		}
		apply.Autoscaling = policy

		if row.StorageMib > 0 {
			apply.EphemeralStorage = &ctrlv1.EphemeralStorage{
				SizeMib: int64(row.StorageMib),
			}
		}

		return &ctrlv1.DeploymentState{
			Version: version,
			State: &ctrlv1.DeploymentState_Apply{
				Apply: apply,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown DeploymentTopologyDesiredStatus: %v", row.DesiredStatus)
	}
}
