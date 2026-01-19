package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// processStateChange routes a state change to the appropriate handler.
//
// Invariant: if we cannot prove a resource should be running in this region,
// we instruct the edge to delete it. This ensures stale resources are cleaned up.
func (s *Service) processStateChange(ctx context.Context, region string, change db.ListStateChangesRow, stream *connect.ServerStream[ctrlv1.State]) error {
	switch change.ResourceType {
	case db.StateChangesResourceTypeDeployment:
		return s.processDeploymentChange(ctx, region, change, stream)
	case db.StateChangesResourceTypeSentinel:
		return s.processSentinelChange(ctx, region, change, stream)
	default:
		return fmt.Errorf("unknown resource type: %q", change.ResourceType)
	}
}

func (s *Service) processDeploymentChange(ctx context.Context, region string, change db.ListStateChangesRow, stream *connect.ServerStream[ctrlv1.State]) error {
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RW(), change.ResourceID)
	if err != nil {
		if db.IsNotFound(err) {
			// Resource already deleted, nothing to sync.
			return nil
		}
		return fmt.Errorf("find deployment id=%q: %w", change.ResourceID, err)
	}

	workspace, err := db.Query.FindWorkspaceByID(ctx, s.db.RW(), deployment.WorkspaceID)
	if err != nil {
		return fmt.Errorf("find workspace id=%q: %w", deployment.WorkspaceID, err)
	}

	if change.Op == db.StateChangesOpDelete {
		return s.sendDeploymentDelete(stream, change.Sequence, workspace.K8sNamespace.String, deployment.K8sName)
	}

	topology, err := db.Query.FindDeploymentTopologyByIDAndRegion(ctx, s.db.RW(), db.FindDeploymentTopologyByIDAndRegionParams{
		DeploymentID: change.ResourceID,
		Region:       region,
	})
	if err != nil {
		if db.IsNotFound(err) {
			// No topology for this region means delete.
			return s.sendDeploymentDelete(stream, change.Sequence, workspace.K8sNamespace.String, deployment.K8sName)
		}
		return fmt.Errorf("find topology deployment=%q region=%q: %w", change.ResourceID, region, err)
	}

	if shouldDeleteDeployment(topology.DesiredState) {
		return s.sendDeploymentDelete(stream, change.Sequence, workspace.K8sNamespace.String, deployment.K8sName)
	}

	return s.sendDeploymentApply(stream, change.Sequence, newApplyDeploymentFromTopology(topology))
}

func shouldDeleteDeployment(desiredState db.DeploymentsDesiredState) bool {
	return desiredState != db.DeploymentsDesiredStateRunning
}

func (s *Service) processSentinelChange(ctx context.Context, region string, change db.ListStateChangesRow, stream *connect.ServerStream[ctrlv1.State]) error {
	sentinel, err := db.Query.FindSentinelByID(ctx, s.db.RW(), change.ResourceID)
	if err != nil {
		if db.IsNotFound(err) {
			// Resource already deleted, nothing to sync.
			return nil
		}
		return fmt.Errorf("find sentinel id=%q: %w", change.ResourceID, err)
	}

	if shouldDeleteSentinel(change.Op, sentinel.Region, region, sentinel.DesiredState) {
		return s.sendSentinelDelete(stream, change.Sequence, sentinel.K8sName)
	}

	return s.sendSentinelApply(stream, change.Sequence, sentinel)
}

func shouldDeleteSentinel(op db.StateChangesOp, sentinelRegion, requestRegion string, desiredState db.SentinelsDesiredState) bool {
	return op == db.StateChangesOpDelete ||
		sentinelRegion != requestRegion ||
		desiredState != db.SentinelsDesiredStateRunning
}
