package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Heartbeat registers or refreshes a krane agent's cluster and region in the
// control plane. Agents call this periodically so the control plane knows which
// regions are available.
//
// The method upserts into regions (keyed by the (platform, name) unique index)
// and clusters (keyed by region_id), updating the cell ID and heartbeat timestamp
// on each call.
func (s *Service) Heartbeat(ctx context.Context, req *connect.Request[ctrlv1.HeartbeatRequest]) (*connect.Response[ctrlv1.HeartbeatResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	if err := validateClusterKey(req.Msg.GetCluster()); err != nil {
		return nil, err
	}

	cellID := req.Msg.GetCluster().GetCellId()
	regionName := req.Msg.GetCluster().GetRegion()
	platform := req.Msg.GetCluster().GetPlatform()
	now := time.Now().UnixMilli()

	err := s.db.UpsertRegion(ctx, db.UpsertRegionParams{
		ID:       uid.New(uid.RegionPrefix),
		Name:     regionName,
		Platform: platform,
	})
	if err != nil {
		logger.Error("failed to upsert region", "error", err, "platform", platform, "region_name", regionName)
		return nil, err
	}

	region, err := s.db.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
		Platform: platform,
		Name:     regionName,
	})
	if err != nil {
		logger.Error("failed to find region", "error", err, "region_id", region.ID)
		return nil, err
	}

	err = s.db.UpsertCluster(ctx, db.UpsertClusterParams{
		ID:              uid.New(uid.ClusterPrefix),
		CellID:          sql.NullString{String: cellID, Valid: true},
		RegionID:        region.ID,
		LastHeartbeatAt: uint64(now),
	})
	if err != nil {
		logger.Error("failed to upsert cluster", "error", err, "region", region)
		return nil, err
	}

	// Every region and cell needs a wildcard cert for its frontline so
	// cross-region and cell-specific TLS work.
	// EnsureInfraCertificate is idempotent (a single indexed read once the
	// records exist) and best-effort, so we call it on every heartbeat rather
	// than gating on a one-shot "region created" signal: that repeated call is
	// the retry path if an earlier provisioning attempt failed to write its
	// records.
	if s.regionalDomain != "" {
		regionDomain := fmt.Sprintf("*.%s.%s.%s", regionName, platform, s.regionalDomain)
		s.EnsureInfraCertificate(ctx, regionDomain)

		cellDomain := fmt.Sprintf("*.%s.%s.%s.%s", cellID, regionName, platform, s.regionalDomain)
		s.EnsureInfraCertificate(ctx, cellDomain)
	}

	return connect.NewResponse(&ctrlv1.HeartbeatResponse{}), nil
}
