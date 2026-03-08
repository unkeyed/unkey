package cluster

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Heartbeat registers or refreshes a krane agent's cluster and region in the
// control plane. Agents call this periodically so the control plane knows which
// regions are available.
//
// The method upserts into regions (keyed by region name) and clusters
// (keyed by region_id), updating the heartbeat timestamp on each call.
func (s *Service) Heartbeat(ctx context.Context, req *connect.Request[ctrlv1.HeartbeatRequest]) (*connect.Response[ctrlv1.HeartbeatResponse], error) {
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	regionName := req.Msg.GetRegion()
	platform := req.Msg.GetPlatform()

	if err := assert.All(
		assert.NotEmpty(regionName, "region is required"),
		assert.NotEmpty(platform, "platform is required"),
	); err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	err := db.Query.UpsertRegion(ctx, s.db.RW(), db.UpsertRegionParams{
		ID:       uid.New(uid.RegionPrefix),
		Name:     regionName,
		Platform: platform,
	})
	if err != nil {
		logger.Error("failed to upsert region", "error", err, "platform", platform, "region_name", regionName)
		return nil, err
	}

	region, err := db.Query.FindRegionByNameAndPlatform(ctx, s.db.RW(), db.FindRegionByNameAndPlatformParams{
		Name:     regionName,
		Platform: platform,
	})
	if err != nil {
		logger.Error("failed to find region", "error", err, "region_id", region.ID)
		return nil, err
	}

	err = db.Query.UpsertCluster(ctx, s.db.RW(), db.UpsertClusterParams{
		ID:              uid.New(uid.ClusterPrefix),
		RegionID:        region.ID,
		LastHeartbeatAt: uint64(now),
	})
	if err != nil {
		logger.Error("failed to upsert cluster", "error", err, "region", region)
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.HeartbeatResponse{}), nil
}
