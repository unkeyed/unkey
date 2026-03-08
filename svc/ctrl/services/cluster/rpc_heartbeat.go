package cluster

import (
	"context"
	"fmt"
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

	region := req.Msg.GetRegion()
	platform := req.Msg.GetPlatform()

	if err := assert.All(
		assert.NotEmpty(region, "region is required"),
		assert.NotEmpty(platform, "platform is required"),
	); err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	err := db.Query.UpsertRegion(ctx, s.db.RW(), db.UpsertRegionParams{
		// using a readable id here to make debugging significantly easier
		// do not rely on this schema though. treat ids as opaque strings.
		ID:       fmt.Sprintf("%s::%s", platform, region),
		Name:     region,
		Platform: platform,
	})
	if err != nil {
		logger.Error("failed to upsert region", "error", err, "region", region)
		return nil, err
	}

	regionID, err := db.Query.FindRegionByNameAndPlatform(ctx, s.db.RW(), db.FindRegionByNameAndPlatformParams{
		Name:     region,
		Platform: platform,
	})
	if err != nil {
		logger.Error("failed to find region", "error", err, "region", region)
		return nil, err
	}

	err = db.Query.UpsertCluster(ctx, s.db.RW(), db.UpsertClusterParams{
		ID:              uid.New(uid.ClusterPrefix),
		RegionID:        regionID,
		LastHeartbeatAt: uint64(now),
	})
	if err != nil {
		logger.Error("failed to upsert cluster", "error", err, "region", region)
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.HeartbeatResponse{}), nil
}
