package cluster

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

// SyncDesiredState streams the full desired state for a region then closes.
// It paginates through all running deployments, active sentinels, and cilium
// policies. Krane calls this on startup and periodically as a safety net.
func (s *Service) SyncDesiredState(
	ctx context.Context,
	req *connect.Request[ctrlv1.SyncDesiredStateRequest],
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
) error {
	if err := auth.Authenticate(req, s.bearer); err != nil {
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
		logger.Error("failed to find region for SyncDesiredState", "error", err, "region_name", regionName, "platform", platform)
		metrics.SyncDesiredStateTotal.WithLabelValues("error").Inc()
		return connect.NewError(connect.CodeInternal, err)
	}

	fullSyncStart := time.Now()

	if err := s.syncDeployments(ctx, stream, region.ID); err != nil {
		metrics.SyncDesiredStateTotal.WithLabelValues("error").Inc()
		return err
	}
	if err := s.syncSentinels(ctx, stream, region.ID); err != nil {
		metrics.SyncDesiredStateTotal.WithLabelValues("error").Inc()
		return err
	}
	if err := s.syncCiliumPolicies(ctx, stream, region.ID); err != nil {
		metrics.SyncDesiredStateTotal.WithLabelValues("error").Inc()
		return err
	}

	fullSyncDuration := time.Since(fullSyncStart).Seconds()
	metrics.FullSyncDurationSeconds.Observe(fullSyncDuration)
	metrics.SyncDesiredStateTotal.WithLabelValues("success").Inc()

	return nil
}

// syncDeployments paginates through all running deployment topologies for a region.
func (s *Service) syncDeployments(
	ctx context.Context,
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
	regionID string,
) error {
	var afterPk uint64
	for {
		rows, err := db.Query.ListAllDeploymentTopologiesByRegion(ctx, s.db.RO(), db.ListAllDeploymentTopologiesByRegionParams{
			RegionID: regionID,
			AfterPk:  afterPk,
			Limit:    changePageSize,
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for _, row := range rows {
			afterPk = row.DeploymentTopology.Pk
			state, err := s.deploymentRowToState(deploymentRow{
				dt:              row.DeploymentTopology,
				d:               row.Deployment,
				k8sNamespace:    row.K8sNamespace,
				environmentSlug: row.EnvironmentSlug,
				regionName:      row.RegionName,
				gitRepo:         row.GitRepo,
			}, 0)
			if err != nil {
				logger.Error("full sync: failed to convert deployment row", "error", err)
				continue
			}
			if state == nil {
				continue
			}
			if err := stream.Send(&ctrlv1.DeploymentChangeEvent{
				Event: &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: state},
			}); err != nil {
				return err
			}
			metrics.SyncDesiredStateEventsSentTotal.WithLabelValues("deployment").Inc()
		}
		if len(rows) < changePageSize {
			return nil
		}
	}
}

// syncSentinels paginates through all sentinels for a region.
func (s *Service) syncSentinels(
	ctx context.Context,
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
	regionID string,
) error {
	var afterPk uint64
	for {
		rows, err := db.Query.ListAllSentinelsByRegion(ctx, s.db.RO(), db.ListAllSentinelsByRegionParams{
			RegionID: regionID,
			AfterPk:  afterPk,
			Limit:    changePageSize,
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for _, joined := range rows {
			afterPk = joined.Sentinel.Pk
			state := s.sentinelToState(joined.Sentinel, joined.SentinelSubscription, 0)
			if state == nil {
				continue
			}
			if err := stream.Send(&ctrlv1.DeploymentChangeEvent{
				Event: &ctrlv1.DeploymentChangeEvent_Sentinel{Sentinel: state},
			}); err != nil {
				return err
			}
			metrics.SyncDesiredStateEventsSentTotal.WithLabelValues("sentinel").Inc()
		}
		if len(rows) < changePageSize {
			return nil
		}
	}
}

// syncCiliumPolicies paginates through all cilium network policies for a region.
func (s *Service) syncCiliumPolicies(
	ctx context.Context,
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
	regionID string,
) error {
	var afterPk uint64
	for {
		rows, err := db.Query.ListAllCiliumNetworkPoliciesByRegion(ctx, s.db.RO(), db.ListAllCiliumNetworkPoliciesByRegionParams{
			RegionID: regionID,
			AfterPk:  afterPk,
			Limit:    changePageSize,
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for _, policy := range rows {
			if err := stream.Send(&ctrlv1.DeploymentChangeEvent{
				Event: &ctrlv1.DeploymentChangeEvent_CiliumNetworkPolicy{
					CiliumNetworkPolicy: ciliumPolicyToState(policy, 0),
				},
			}); err != nil {
				return err
			}
			metrics.SyncDesiredStateEventsSentTotal.WithLabelValues("cilium_network_policy").Inc()
			afterPk = policy.Pk
		}
		if len(rows) < changePageSize {
			return nil
		}
	}
}
