package cluster

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

// WatchDeploymentChanges streams all resource changes (deployments, sentinels, cilium policies)
// in a single stream. The agent maintains one cursor and dispatches events by type.
//
// The stream operates in two modes:
//   - Full sync (version_last_seen = 0): paginates through all current state from
//     deployment_topology, sentinels, and cilium_network_policies tables, then sets
//     the cursor to the current max deployment_changes pk.
//   - Incremental (version_last_seen > 0): polls the deployment_changes table for new
//     entries and loads current state via point lookups.
func (s *Service) WatchDeploymentChanges(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchDeploymentChangesRequest],
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
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

	region, err := db.Query.FindRegionByNameAndPlatform(ctx, s.db.RO(), db.FindRegionByNameAndPlatformParams{
		Name:     regionName,
		Platform: platform,
	})
	if err != nil {
		logger.Error("failed to find region for WatchDeploymentChanges", "error", err, "region_name", regionName, "platform", platform)
		return connect.NewError(connect.CodeInternal, err)
	}

	versionCursor := req.Msg.GetVersionLastSeen()

	// Full sync: paginate through all current state.
	if versionCursor == 0 {
		fullSyncStart := time.Now()

		maxVersion, err := db.Query.GetDeploymentChangesMaxVersion(ctx, s.db.RO(), region.ID)
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		version := uint64(maxVersion)

		if err := s.fullSyncDeployments(ctx, stream, region.ID, version); err != nil {
			return err
		}
		if err := s.fullSyncSentinels(ctx, stream, region.ID, version); err != nil {
			return err
		}
		if err := s.fullSyncCiliumPolicies(ctx, stream, region.ID, version); err != nil {
			return err
		}

		versionCursor = version
		fullSyncDuration := time.Since(fullSyncStart).Seconds()
		logger.Info("full sync complete", "region_id", region.ID, "cursor", versionCursor, "duration_s", fullSyncDuration)
		metrics.FullSyncDurationSeconds.Observe(fullSyncDuration)
	}

	// Incremental: poll deployment_changes for new entries.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		events, err := s.fetchDeploymentChangeEvents(ctx, region.ID, versionCursor)
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

// fullSyncDeployments paginates through all deployment topologies for a region.
func (s *Service) fullSyncDeployments(
	ctx context.Context,
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
	regionID string,
	version uint64,
) error {
	var afterPk uint64
	for {
		rows, err := db.Query.ListAllDeploymentTopologiesByRegion(ctx, s.db.RO(), db.ListAllDeploymentTopologiesByRegionParams{
			RegionID: regionID,
			AfterPk:  afterPk,
			Limit:    100,
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for _, row := range rows {
			state, err := s.deploymentRowToState(deploymentRow{
				dt:              row.DeploymentTopology,
				d:               row.Deployment,
				k8sNamespace:    row.K8sNamespace,
				environmentSlug: row.EnvironmentSlug,
				regionName:      row.RegionName,
				gitRepo:         row.GitRepo,
			}, version)
			if err != nil {
				logger.Error("failed to convert deployment row", "error", err)
				continue
			}
			if state == nil {
				continue
			}
			if err := stream.Send(&ctrlv1.DeploymentChangeEvent{
				Version: version,
				Event:   &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: state},
			}); err != nil {
				return err
			}
			afterPk = row.DeploymentTopology.Pk
		}
		if len(rows) < 100 {
			return nil
		}
	}
}

// fullSyncSentinels paginates through all sentinels for a region.
func (s *Service) fullSyncSentinels(
	ctx context.Context,
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
	regionID string,
	version uint64,
) error {
	var afterPk uint64
	for {
		rows, err := db.Query.ListAllSentinelsByRegion(ctx, s.db.RO(), db.ListAllSentinelsByRegionParams{
			RegionID: regionID,
			AfterPk:  afterPk,
			Limit:    100,
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for _, sentinel := range rows {
			state := s.sentinelToState(sentinel, version)
			if state == nil {
				continue
			}
			if err := stream.Send(&ctrlv1.DeploymentChangeEvent{
				Version: version,
				Event:   &ctrlv1.DeploymentChangeEvent_Sentinel{Sentinel: state},
			}); err != nil {
				return err
			}
			afterPk = sentinel.Pk
		}
		if len(rows) < 100 {
			return nil
		}
	}
}

// fullSyncCiliumPolicies paginates through all cilium network policies for a region.
func (s *Service) fullSyncCiliumPolicies(
	ctx context.Context,
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
	regionID string,
	version uint64,
) error {
	var afterPk uint64
	for {
		rows, err := db.Query.ListAllCiliumNetworkPoliciesByRegion(ctx, s.db.RO(), db.ListAllCiliumNetworkPoliciesByRegionParams{
			RegionID: regionID,
			AfterPk:  afterPk,
			Limit:    100,
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for _, policy := range rows {
			if err := stream.Send(&ctrlv1.DeploymentChangeEvent{
				Version: version,
				Event: &ctrlv1.DeploymentChangeEvent_CiliumNetworkPolicy{
					CiliumNetworkPolicy: ciliumPolicyToState(policy, version),
				},
			}); err != nil {
				return err
			}
			afterPk = policy.Pk
		}
		if len(rows) < 100 {
			return nil
		}
	}
}

// fetchDeploymentChangeEvents polls deployment_changes for new entries and does a
// point lookup for each row to load current state.
func (s *Service) fetchDeploymentChangeEvents(ctx context.Context, regionID string, afterVersion uint64) ([]*ctrlv1.DeploymentChangeEvent, error) {
	changes, err := db.Query.ListDeploymentChangesByRegionAll(ctx, s.db.RO(), db.ListDeploymentChangesByRegionAllParams{
		RegionID:     regionID,
		AfterVersion: afterVersion,
		Limit:        100,
	})
	if err != nil {
		return nil, err
	}

	events := make([]*ctrlv1.DeploymentChangeEvent, 0, len(changes))
	for _, change := range changes {
		event, err := s.loadChangeEvent(ctx, change)
		if err != nil {
			logger.Error("failed to load state for deployment change",
				"error", err,
				"resource_type", change.ResourceType,
				"resource_id", change.ResourceID,
			)
			// Skip this row but keep advancing the cursor
			events = append(events, &ctrlv1.DeploymentChangeEvent{Version: change.Pk})
			continue
		}
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// loadChangeEvent does a point lookup for a single deployment_changes row based on resource_type.
func (s *Service) loadChangeEvent(ctx context.Context, change db.DeploymentChange) (*ctrlv1.DeploymentChangeEvent, error) {
	switch change.ResourceType {
	case db.DeploymentChangesResourceTypeDeploymentTopology:
		row, err := db.Query.FindDeploymentTopologyByDeploymentAndRegion(ctx, s.db.RO(), db.FindDeploymentTopologyByDeploymentAndRegionParams{
			DeploymentID: change.ResourceID,
			RegionID:     change.RegionID,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil
			}
			return nil, err
		}
		state, err := s.deploymentRowToState(deploymentRow{
			dt:              row.DeploymentTopology,
			d:               row.Deployment,
			k8sNamespace:    row.K8sNamespace,
			environmentSlug: row.EnvironmentSlug,
			regionName:      row.RegionName,
			gitRepo:         row.GitRepo,
		}, change.Pk)
		if err != nil {
			return nil, err
		}
		if state == nil {
			return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil
		}
		return &ctrlv1.DeploymentChangeEvent{
			Version: change.Pk,
			Event:   &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: state},
		}, nil

	case db.DeploymentChangesResourceTypeSentinel:
		sentinel, err := db.Query.FindSentinelByID(ctx, s.db.RO(), change.ResourceID)
		if err != nil {
			if db.IsNotFound(err) {
				return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil
			}
			return nil, err
		}
		state := s.sentinelToState(sentinel, change.Pk)
		if state == nil {
			return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil
		}
		return &ctrlv1.DeploymentChangeEvent{
			Version: change.Pk,
			Event:   &ctrlv1.DeploymentChangeEvent_Sentinel{Sentinel: state},
		}, nil

	case db.DeploymentChangesResourceTypeCiliumNetworkPolicy:
		policy, err := db.Query.FindCiliumNetworkPolicyByIDAndRegion(ctx, s.db.RO(), db.FindCiliumNetworkPolicyByIDAndRegionParams{
			RegionID:              change.RegionID,
			CiliumNetworkPolicyID: change.ResourceID,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("resource %s does not exist in region %s", change.ResourceID, change.RegionID))
			}
			return nil, err
		}
		return &ctrlv1.DeploymentChangeEvent{
			Version: change.Pk,
			Event: &ctrlv1.DeploymentChangeEvent_CiliumNetworkPolicy{
				CiliumNetworkPolicy: ciliumPolicyToState(policy, change.Pk),
			},
		}, nil

	default:
		logger.Error("unknown resource type in deployment_changes", "resource_type", change.ResourceType)
		return &ctrlv1.DeploymentChangeEvent{Version: change.Pk}, nil
	}
}

// deploymentRow holds the common fields from both full sync and incremental query results.
type deploymentRow struct {
	dt              db.DeploymentTopology
	d               db.Deployment
	k8sNamespace    sql.NullString
	environmentSlug string
	regionName      string
	gitRepo         sql.NullString
}

// deploymentRowToState converts a deployment row to a proto DeploymentState message.
func (s *Service) deploymentRowToState(row deploymentRow, version uint64) (*ctrlv1.DeploymentState, error) {
	switch row.dt.DesiredStatus {
	case db.DeploymentTopologyDesiredStatusStopped:
		return &ctrlv1.DeploymentState{
			Version: version,
			State: &ctrlv1.DeploymentState_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					K8SNamespace: row.k8sNamespace.String,
					K8SName:      row.d.K8sName,
				},
			},
		}, nil
	case db.DeploymentTopologyDesiredStatusRunning:
		var buildID *string
		if row.d.BuildID.Valid {
			buildID = &row.d.BuildID.String
		}

		apply := &ctrlv1.ApplyDeployment{
			DeploymentId:                  row.d.ID,
			K8SNamespace:                  row.k8sNamespace.String,
			K8SName:                       row.d.K8sName,
			WorkspaceId:                   row.d.WorkspaceID,
			ProjectId:                     row.d.ProjectID,
			EnvironmentId:                 row.d.EnvironmentID,
			AppId:                         row.d.AppID,
			Image:                         row.d.Image.String,
			CpuMillicores:                 int64(row.d.CpuMillicores),
			MemoryMib:                     int64(row.d.MemoryMib),
			EncryptedEnvironmentVariables: row.d.EncryptedEnvironmentVariables,
			BuildId:                       buildID,
			Command:                       row.d.Command,
			Port:                          row.d.Port,
			ShutdownSignal:                string(row.d.ShutdownSignal),
			EnvironmentSlug:               &row.environmentSlug,
			Region:                        &row.regionName,
		}

		if row.d.GitCommitSha.Valid {
			apply.GitCommitSha = &row.d.GitCommitSha.String
		}
		if row.d.GitBranch.Valid {
			apply.GitBranch = &row.d.GitBranch.String
		}
		if row.d.GitCommitMessage.Valid {
			apply.GitCommitMessage = &row.d.GitCommitMessage.String
		}
		if row.gitRepo.Valid {
			apply.GitRepo = &row.gitRepo.String
		}

		if row.d.Healthcheck.Valid {
			hcBytes, err := json.Marshal(row.d.Healthcheck.Healthcheck)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal healthcheck: %w", err)
			}
			apply.Healthcheck = hcBytes
		}

		policy := &ctrlv1.AutoscalingPolicy{
			MinReplicas: row.dt.AutoscalingReplicasMin,
			MaxReplicas: row.dt.AutoscalingReplicasMax,
		}
		if row.dt.AutoscalingThresholdCpu.Valid {
			policy.CpuThreshold = ptr.P(int32(row.dt.AutoscalingThresholdCpu.Int16))
		}
		if row.dt.AutoscalingThresholdMemory.Valid {
			policy.MemoryThreshold = ptr.P(int32(row.dt.AutoscalingThresholdMemory.Int16))
		}
		apply.Autoscaling = policy

		return &ctrlv1.DeploymentState{
			Version: version,
			State: &ctrlv1.DeploymentState_Apply{
				Apply: apply,
			},
		}, nil
	default:
		logger.Error("unhandled deployment topology desired status", "status", row.dt.DesiredStatus)
		return nil, nil
	}
}

// sentinelToState converts a sentinel DB row to a proto SentinelState message.
func (s *Service) sentinelToState(sentinel db.Sentinel, version uint64) *ctrlv1.SentinelState {
	switch sentinel.DesiredState {
	case db.SentinelsDesiredStateArchived, db.SentinelsDesiredStateStandby:
		return &ctrlv1.SentinelState{
			Version: version,
			State: &ctrlv1.SentinelState_Delete{
				Delete: &ctrlv1.DeleteSentinel{
					K8SName: sentinel.K8sName,
				},
			},
		}
	case db.SentinelsDesiredStateRunning:
		return &ctrlv1.SentinelState{
			Version: version,
			State: &ctrlv1.SentinelState_Apply{
				Apply: &ctrlv1.ApplySentinel{
					SentinelId:    sentinel.ID,
					K8SName:       sentinel.K8sName,
					WorkspaceId:   sentinel.WorkspaceID,
					ProjectId:     sentinel.ProjectID,
					EnvironmentId: sentinel.EnvironmentID,
					Replicas:      sentinel.DesiredReplicas,
					Image:         sentinel.Image,
					CpuMillicores: int64(sentinel.CpuMillicores),
					MemoryMib:     int64(sentinel.MemoryMib),
				},
			},
		}
	default:
		logger.Error("unhandled sentinel desired state", "desiredState", sentinel.DesiredState)
		return nil
	}
}

// ciliumPolicyToState converts a cilium network policy DB row to a proto state message.
func ciliumPolicyToState(policy db.CiliumNetworkPolicy, version uint64) *ctrlv1.CiliumNetworkPolicyState {
	return &ctrlv1.CiliumNetworkPolicyState{
		Version: version,
		State: &ctrlv1.CiliumNetworkPolicyState_Apply{
			Apply: &ctrlv1.ApplyCiliumNetworkPolicy{
				CiliumNetworkPolicyId: policy.ID,
				K8SNamespace:          policy.K8sNamespace,
				K8SName:               policy.K8sName,
				Policy:                policy.Policy,
			},
		},
	}
}
