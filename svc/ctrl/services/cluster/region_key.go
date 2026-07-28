package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// resolvedRegion is the cluster-derived region metadata needed by agent RPCs.
type resolvedRegion struct {
	ID       string
	Name     string
	Platform string
}

// validateRegionKey returns an InvalidArgument error when the RegionKey is
// missing or either of its fields is blank. Handlers call this before doing
// any database work so a misconfigured agent fails fast with a clear message
// rather than a downstream "region not found" error. The proto-generated
// getters are nil-safe, so this works even when rk itself is nil.
func validateRegionKey(rk *ctrlv1.RegionKey) error {
	if err := assert.All(
		assert.NotEmpty(rk.GetPlatform(), "region.platform is required"),
		assert.NotEmpty(rk.GetName(), "region.name is required"),
	); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return nil
}

// resolveRegion validates the RegionKey then loads the matching cluster-derived
// region via an SWR cache. Returns InvalidArgument for a missing/blank
// key, NotFound when the region is unknown, and Internal for any other DB
// error. NotFound is intentionally not cached so that a krane agent whose
// Heartbeat just registered the cluster is not stuck behind a stale negative
// lookup.
func (s *Service) resolveRegion(ctx context.Context, rk *ctrlv1.RegionKey) (resolvedRegion, error) {
	if err := validateRegionKey(rk); err != nil {
		return resolvedRegion{}, err
	}
	key := regionCacheKey{platform: rk.GetPlatform(), name: rk.GetName()}
	region, _, err := s.regionCache.SWR(ctx, key,
		func(ctx context.Context) (resolvedRegion, error) {
			row, err := s.db.FindClusterRegionByPlatformAndName(ctx, db.FindClusterRegionByPlatformAndNameParams{
				Platform: key.platform,
				Name:     key.name,
			})
			if err != nil {
				return resolvedRegion{}, err
			}
			return resolvedRegion{ID: row.ID, Name: row.Name, Platform: row.Platform}, nil
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
			return resolvedRegion{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("region %s/%s not found", key.platform, key.name))
		}
		return resolvedRegion{}, connect.NewError(connect.CodeInternal, err)
	}
	return region, nil
}
