package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/db"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

// bootstrap streams the full desired state for a region.
//
// This captures the current max sequence as a snapshot boundary, then streams all
// running deployments and sentinels. The sequence returned is NOT a true snapshot
// (state may have changed during streaming), but convergence is guaranteed because:
// 1. All apply/delete operations are idempotent
// 2. Any changes during bootstrap will be picked up on the next sync
//
// Stream closing without error signals bootstrap completion.
func (s *Service) bootstrap(ctx context.Context, region string, stream *connect.ServerStream[ctrlv1.State]) (uint64, error) {
	maxSequence, err := db.Query.GetMaxStateChangeSequence(ctx, s.db.RW(), region)
	if err != nil {
		return 0, fmt.Errorf("get max sequence region=%q: %w", region, err)
	}
	sequenceBoundary := uint64(maxSequence)

	if err := s.bootstrapDeployments(ctx, region, sequenceBoundary, stream); err != nil {
		return 0, err
	}

	if err := s.bootstrapSentinels(ctx, region, sequenceBoundary, stream); err != nil {
		return 0, err
	}

	s.logger.Info("bootstrap complete", "sequenceBoundary", sequenceBoundary)
	return sequenceBoundary, nil
}

func (s *Service) bootstrapDeployments(ctx context.Context, region string, sequence uint64, stream *connect.ServerStream[ctrlv1.State]) error {
	cursor := ""
	for {
		topologies, err := db.Query.ListDesiredDeploymentTopology(ctx, s.db.RW(), db.ListDesiredDeploymentTopologyParams{
			Region:           region,
			DesiredState:     db.DeploymentsDesiredStateRunning,
			PaginationCursor: cursor,
			Limit:            1000,
		})
		if err != nil {
			return fmt.Errorf("list deployment topologies cursor=%q: %w", cursor, err)
		}
		if len(topologies) == 0 {
			break
		}
		cursor = topologies[len(topologies)-1].Deployment.ID

		for _, topology := range topologies {
			if err := s.sendDeploymentApplyFromTopology(stream, sequence, topology); err != nil {
				return fmt.Errorf("send deployment id=%q: %w", topology.Deployment.ID, err)
			}
		}
	}
	return nil
}

func (s *Service) bootstrapSentinels(ctx context.Context, region string, sequence uint64, stream *connect.ServerStream[ctrlv1.State]) error {
	cursor := ""
	for {
		sentinels, err := db.Query.ListDesiredSentinels(ctx, s.db.RW(), db.ListDesiredSentinelsParams{
			Region:           region,
			DesiredState:     db.SentinelsDesiredStateRunning,
			PaginationCursor: cursor,
			Limit:            100,
		})
		if err != nil {
			return fmt.Errorf("list sentinels cursor=%q: %w", cursor, err)
		}
		if len(sentinels) == 0 {
			break
		}
		cursor = sentinels[len(sentinels)-1].ID

		for _, sentinel := range sentinels {
			if err := s.sendSentinelApply(stream, sequence, sentinel); err != nil {
				return fmt.Errorf("send sentinel id=%q: %w", sentinel.ID, err)
			}
		}
	}
	return nil
}
