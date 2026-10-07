package cluster

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

// deploymentSyncPageSize bounds each database read during a full sync.
const deploymentSyncPageSize = 10000

// SyncDesiredState streams the full desired state for a region then closes.
// It paginates through all running deployments. Krane calls this on startup
// and periodically as a safety net.
func (s *Service) SyncDesiredState(
	ctx context.Context,
	req *connect.Request[ctrlv1.SyncDesiredStateRequest],
	stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent],
) error {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return err
	}

	cluster, err := s.resolveCluster(ctx, req.Msg.GetCluster())
	if err != nil {
		metrics.SyncDesiredStateTotal.WithLabelValues("error").Inc()
		return err
	}

	fullSyncStart := time.Now()

	if err := s.syncDeployments(ctx, stream, cluster.RegionID); err != nil {
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
		rows, err := s.db.ListAllDeploymentTopologiesByRegion(ctx, db.ListAllDeploymentTopologiesByRegionParams{
			RegionID: regionID,
			AfterPk:  afterPk,
			Limit:    deploymentSyncPageSize,
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for _, row := range rows {
			afterPk = row.TopologyPk
			state, err := deploymentRowToState(row)
			if err != nil {
				logger.Error("full sync: failed to convert deployment row", "error", err)
				continue
			}
			if err := stream.Send(&ctrlv1.DeploymentChangeEvent{
				Event: &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: state},
			}); err != nil {
				return err
			}
			metrics.SyncDesiredStateEventsSentTotal.WithLabelValues("deployment").Inc()
		}
		if len(rows) < deploymentSyncPageSize {
			return nil
		}
	}
}
