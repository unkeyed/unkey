package sync

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *SyncEngine) Reconcile(ctx context.Context) error {

	s.logger.Info("starting reconciliation for gateways")
	for gatewayID := range s.gatewaycontroller.GetRunningGatewayIds(ctx) {
		e, err := s.reconcileGatewayCircuitBreaker.Do(ctx, func(ctx context.Context) (*connect.Response[ctrlv1.GatewayEvent], error) {
			return s.ctrl.GetDesiredGatewayState(ctx, connect.NewRequest(&ctrlv1.GetDesiredGatewayStateRequest{
				GatewayId: gatewayID,
			}))
		})
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}
		s.gatewaycontroller.BufferEvent(e.Msg)
	}

	s.logger.Info("starting reconciliation for deployments")
	for deploymentID := range s.deploymentcontroller.GetRunningDeploymentIds(ctx) {
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

	return nil

}
