package sync

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *SyncEngine) Reconcile(ctx context.Context) error {

	s.logger.Info("starting reconciliation for sentinels")
	sentinels := 0
	for sentinelID := range s.sentinelcontroller.GetRunningSentinelIDs(ctx) {
		sentinels++
		e, err := s.reconcileSentinelCircuitBreaker.Do(ctx, func(ctx context.Context) (*connect.Response[ctrlv1.SentinelEvent], error) {
			return s.ctrl.GetDesiredSentinelState(ctx, connect.NewRequest(&ctrlv1.GetDesiredSentinelStateRequest{
				SentinelId: sentinelID,
			}))
		})
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}

		s.sentinelcontroller.BufferEvent(e.Msg)
	}
	s.logger.Info("reconciled sentinels", "count", sentinels)

	s.logger.Info("starting reconciliation for deployments")
	deployments := 0
	for deploymentID := range s.deploymentcontroller.GetRunningDeploymentIds(ctx) {
		deployments++
		e, err := s.reconcileDeploymentCircuitBreaker.Do(ctx, func(ctx context.Context) (*connect.Response[ctrlv1.DeploymentEvent], error) {
			return s.ctrl.GetDesiredDeploymentState(ctx, connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{
				DeploymentId: deploymentID,
			}))
		})
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}
		s.deploymentcontroller.BufferEvent(e.Msg)
	}
	s.logger.Info("reconciled deployments", "count", deployments)

	return nil

}
