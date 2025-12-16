package sync

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

// Reconcile synchronizes current cluster state with desired state from control plane.
//
// This method performs bidirectional reconciliation by:
//  1. Getting all scheduled deployment and sentinel IDs from controllers
//  2. Fetching desired state for each resource from control plane
//  3. Buffering events to controllers for application
//  4. Using circuit breakers to prevent cascade failures
//
// This operation is called periodically to ensure cluster state matches
// control plane expectations and to handle any drift or missed events.
//
// Parameters:
//   - ctx: Context for reconciliation operations
//
// Returns an error if reconciliation encounters critical problems,
// but individual resource failures are logged and don't stop the process.
func (s *SyncEngine) Reconcile(ctx context.Context) error {

	s.logger.Info("starting reconciliation of cluster state")
	sentinels := 0
	for sentinelID := range s.sentinelcontroller.GetScheduledSentinelIDs(ctx) {
		e, err := s.reconcileSentinelCircuitBreaker.Do(ctx, func(ctx context.Context) (*connect.Response[ctrlv1.SentinelEvent], error) {
			return s.ctrl.GetDesiredSentinelState(ctx, connect.NewRequest(&ctrlv1.GetDesiredSentinelStateRequest{
				SentinelId: sentinelID,
			}))
		})
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}

		err = s.routeSentinelEvent(ctx, e.Msg)
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}
		sentinels++

	}

	deployments := 0
	for deploymentID := range s.deploymentcontroller.GetScheduledDeploymentIDs(ctx) {
		e, err := s.reconcileDeploymentCircuitBreaker.Do(ctx, func(ctx context.Context) (*connect.Response[ctrlv1.DeploymentEvent], error) {
			return s.ctrl.GetDesiredDeploymentState(ctx, connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{
				DeploymentId: deploymentID,
			}))
		})
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}
		err = s.routeDeploymentEvent(ctx, e.Msg)
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}
		deployments++

	}
	s.logger.Info("reconciled cluster state done",
		"deployments", deployments,
		"sentinels", sentinels,
	)

	return nil

}
