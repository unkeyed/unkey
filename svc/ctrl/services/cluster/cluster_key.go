package cluster

import (
	"context"
	"database/sql"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// validateClusterKey returns an InvalidArgument error when the ClusterKey is
// missing or any of its fields is blank. Handlers call this before doing
// any database work so a misconfigured agent fails fast with a clear message
// rather than a downstream "region not found" error. The proto-generated
// getters are nil-safe, so this works even when key itself is nil.
func validateClusterKey(key *ctrlv1.ClusterKey) error {
	if err := assert.All(
		assert.NotEmpty(key.GetCellId(), "cluster.cell_id is required"),
		assert.NotEmpty(key.GetPlatform(), "cluster.platform is required"),
		assert.NotEmpty(key.GetRegion(), "cluster.region is required"),
	); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return nil
}

// resolveCluster validates the ClusterKey then loads its immutable cluster and
// region rows through the cluster cache. Errors are not cached so a legacy row
// can start resolving immediately after its first heartbeat claims a cell ID.
func (s *Service) resolveCluster(ctx context.Context, cluster *ctrlv1.ClusterKey) (db.FindClusterRow, error) {
	if err := validateClusterKey(cluster); err != nil {
		return db.FindClusterRow{}, err
	}
	key := clusterCacheKey{
		cellID:   cluster.GetCellId(),
		platform: cluster.GetPlatform(),
		region:   cluster.GetRegion(),
	}
	resolved, _, err := s.clusterCache.SWR(ctx, key,
		func(ctx context.Context) (db.FindClusterRow, error) {
			return s.db.FindCluster(ctx, db.FindClusterParams{
				CellID:   sql.NullString{String: key.cellID, Valid: true},
				Platform: key.platform,
				Region:   key.region,
			})
		},
		func(err error) cache.Op {
			if err == nil {
				return cache.WriteValue
			}
			return cache.Noop
		},
	)
	if err != nil {
		if db.IsNotFound(err) {
			return db.FindClusterRow{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("cluster %s/%s/%s not found", key.cellID, key.platform, key.region))
		}
		return db.FindClusterRow{}, connect.NewError(connect.CodeInternal, err)
	}
	return resolved, nil
}
