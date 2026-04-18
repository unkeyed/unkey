package cluster

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
)

// findTopologyMinReplicas returns the per-region minimum replica requirements
// for a deployment, cached by deployment_id. Topology rows are written once
// at deploy time, so a long TTL is safe; empty results are NOT cached to
// avoid sealing in a race where the RPC arrives before the topology INSERTs
// commit. Returns (nil, nil) when no rows are found.
func (s *Service) findTopologyMinReplicas(ctx context.Context, deploymentID string) ([]db.FindDeploymentTopologyMinReplicasRow, error) {
	rows, _, err := s.topologyCache.SWR(ctx, deploymentID,
		func(ctx context.Context) ([]db.FindDeploymentTopologyMinReplicasRow, error) {
			loaded, err := db.Query.FindDeploymentTopologyMinReplicas(ctx, s.db.RO(), deploymentID)
			if err != nil {
				return nil, err
			}
			if len(loaded) == 0 {
				// Treat "no rows yet" as a miss so the cache Op below
				// decides not to store the absence.
				return nil, sql.ErrNoRows
			}
			return loaded, nil
		},
		topologyCacheOp,
	)
	if err != nil && !db.IsNotFound(err) {
		return nil, err
	}
	return rows, nil
}

// topologyCacheOp writes hits, never seals in an absence. Topology rows are
// written once and then stable, so a "not found" result is almost always a
// transient race with the deploy workflow's INSERT, and caching it would
// keep maybeNotifyInstancesReady stuck until the stale window elapsed.
func topologyCacheOp(err error) cache.Op {
	if err == nil {
		return cache.WriteValue
	}
	return cache.Noop
}
