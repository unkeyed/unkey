package cluster

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
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
		state, err := deploymentRowToState(deploymentRowFromPointLookup(row), change.Pk)
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

// deploymentRow holds the common fields from both full sync and incremental query results.
type deploymentRow struct {
	desiredStatus                 db.DeploymentTopologyDesiredStatus
	autoscalingReplicasMin        uint32
	autoscalingReplicasMax        uint32
	autoscalingThresholdCPU       sql.NullInt16
	autoscalingThresholdMemory    sql.NullInt16
	deploymentID                  string
	k8sName                       string
	workspaceID                   string
	projectID                     string
	environmentID                 string
	appID                         string
	image                         sql.NullString
	buildID                       sql.NullString
	gitCommitSHA                  sql.NullString
	gitBranch                     sql.NullString
	gitCommitMessage              sql.NullString
	cpuMillicores                 int32
	memoryMiB                     int32
	storageMiB                    uint32
	encryptedEnvironmentVariables []byte
	command                       mysqltype.StringSlice
	port                          int32
	shutdownSignal                db.DeploymentsShutdownSignal
	healthcheck                   mysqltype.NullHealthcheck
	k8sNamespace                  sql.NullString
	environmentSlug               string
	regionName                    string
	gitRepo                       sql.NullString
}

func deploymentRowFromPointLookup(row db.FindDeploymentTopologyByDeploymentAndRegionRow) deploymentRow {
	return deploymentRow{
		desiredStatus:                 row.DesiredStatus,
		autoscalingReplicasMin:        row.AutoscalingReplicasMin,
		autoscalingReplicasMax:        row.AutoscalingReplicasMax,
		autoscalingThresholdCPU:       row.AutoscalingThresholdCpu,
		autoscalingThresholdMemory:    row.AutoscalingThresholdMemory,
		deploymentID:                  row.ID,
		k8sName:                       row.K8sName,
		workspaceID:                   row.WorkspaceID,
		projectID:                     row.ProjectID,
		environmentID:                 row.EnvironmentID,
		appID:                         row.AppID,
		image:                         row.Image,
		buildID:                       row.BuildID,
		gitCommitSHA:                  row.GitCommitSha,
		gitBranch:                     row.GitBranch,
		gitCommitMessage:              row.GitCommitMessage,
		cpuMillicores:                 row.CpuMillicores,
		memoryMiB:                     row.MemoryMib,
		storageMiB:                    row.StorageMib,
		encryptedEnvironmentVariables: row.EncryptedEnvironmentVariables,
		command:                       row.Command,
		port:                          row.Port,
		shutdownSignal:                row.ShutdownSignal,
		healthcheck:                   row.Healthcheck,
		k8sNamespace:                  row.K8sNamespace,
		environmentSlug:               row.EnvironmentSlug,
		regionName:                    row.RegionName,
		gitRepo:                       row.GitRepo,
	}
}

func deploymentRowFromFullSync(row db.ListAllDeploymentTopologiesByRegionRow) deploymentRow {
	return deploymentRow{
		desiredStatus:                 row.DeploymentTopology.DesiredStatus,
		autoscalingReplicasMin:        row.DeploymentTopology.AutoscalingReplicasMin,
		autoscalingReplicasMax:        row.DeploymentTopology.AutoscalingReplicasMax,
		autoscalingThresholdCPU:       row.DeploymentTopology.AutoscalingThresholdCpu,
		autoscalingThresholdMemory:    row.DeploymentTopology.AutoscalingThresholdMemory,
		deploymentID:                  row.Deployment.ID,
		k8sName:                       row.Deployment.K8sName,
		workspaceID:                   row.Deployment.WorkspaceID,
		projectID:                     row.Deployment.ProjectID,
		environmentID:                 row.Deployment.EnvironmentID,
		appID:                         row.Deployment.AppID,
		image:                         row.Deployment.Image,
		buildID:                       row.Deployment.BuildID,
		gitCommitSHA:                  row.Deployment.GitCommitSha,
		gitBranch:                     row.Deployment.GitBranch,
		gitCommitMessage:              row.Deployment.GitCommitMessage,
		cpuMillicores:                 row.Deployment.CpuMillicores,
		memoryMiB:                     row.Deployment.MemoryMib,
		storageMiB:                    row.Deployment.StorageMib,
		encryptedEnvironmentVariables: row.Deployment.EncryptedEnvironmentVariables,
		command:                       row.Deployment.Command,
		port:                          row.Deployment.Port,
		shutdownSignal:                row.Deployment.ShutdownSignal,
		healthcheck:                   row.Deployment.Healthcheck,
		k8sNamespace:                  row.K8sNamespace,
		environmentSlug:               row.EnvironmentSlug,
		regionName:                    row.RegionName,
		gitRepo:                       row.GitRepo,
	}
}

// deploymentRowToState converts a deployment row to a proto DeploymentState message.
func deploymentRowToState(row deploymentRow, version uint64) (*ctrlv1.DeploymentState, error) {
	switch row.desiredStatus {
	case db.DeploymentTopologyDesiredStatusStopped:
		return &ctrlv1.DeploymentState{
			Version: version,
			State: &ctrlv1.DeploymentState_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					K8SNamespace: row.k8sNamespace.String,
					K8SName:      row.k8sName,
				},
			},
		}, nil
	case db.DeploymentTopologyDesiredStatusRunning:
		var buildID *string
		if row.buildID.Valid {
			buildID = &row.buildID.String
		}

		apply := &ctrlv1.ApplyDeployment{
			DeploymentId:                  row.deploymentID,
			K8SNamespace:                  row.k8sNamespace.String,
			K8SName:                       row.k8sName,
			WorkspaceId:                   row.workspaceID,
			ProjectId:                     row.projectID,
			EnvironmentId:                 row.environmentID,
			AppId:                         row.appID,
			Image:                         row.image.String,
			CpuMillicores:                 int64(row.cpuMillicores),
			MemoryMib:                     int64(row.memoryMiB),
			EncryptedEnvironmentVariables: row.encryptedEnvironmentVariables,
			BuildId:                       buildID,
			Command:                       row.command,
			Port:                          row.port,
			ShutdownSignal:                string(row.shutdownSignal),
			EnvironmentSlug:               &row.environmentSlug,
			Region:                        &row.regionName,
		}

		if row.gitCommitSHA.Valid {
			apply.GitCommitSha = &row.gitCommitSHA.String
		}
		if row.gitBranch.Valid {
			apply.GitBranch = &row.gitBranch.String
		}
		if row.gitCommitMessage.Valid {
			apply.GitCommitMessage = &row.gitCommitMessage.String
		}
		if row.gitRepo.Valid {
			apply.GitRepo = &row.gitRepo.String
		}

		if row.healthcheck.Valid {
			hcBytes, err := json.Marshal(row.healthcheck.Healthcheck)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal healthcheck: %w", err)
			}
			apply.Healthcheck = hcBytes
		}

		policy := &ctrlv1.AutoscalingPolicy{
			MinReplicas: row.autoscalingReplicasMin,
			MaxReplicas: row.autoscalingReplicasMax,
		}
		if row.autoscalingThresholdCPU.Valid {
			policy.CpuThreshold = ptr.P(int32(row.autoscalingThresholdCPU.Int16))
		}
		if row.autoscalingThresholdMemory.Valid {
			policy.MemoryThreshold = ptr.P(int32(row.autoscalingThresholdMemory.Int16))
		}
		apply.Autoscaling = policy

		if row.storageMiB > 0 {
			apply.EphemeralStorage = &ctrlv1.EphemeralStorage{
				SizeMib: int64(row.storageMiB),
			}
		}

		return &ctrlv1.DeploymentState{
			Version: version,
			State: &ctrlv1.DeploymentState_Apply{
				Apply: apply,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown DeploymentTopologyDesiredStatus: %v", row.desiredStatus)
	}
}
