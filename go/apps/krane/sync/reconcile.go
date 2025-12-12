package sync

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *SyncEngine) Reconcile(ctx context.Context) error {

	s.logger.Info("starting reconciliation for gateways")
	gateways := 0
	for gatewayID := range s.gatewaycontroller.GetRunningGatewayIDs(ctx) {
		gateways++
		e, err := s.reconcileGatewayCircuitBreaker.Do(ctx, func(ctx context.Context) (*connect.Response[ctrlv1.GatewayEvent], error) {
			return s.ctrl.GetDesiredGatewayState(ctx, connect.NewRequest(&ctrlv1.GetDesiredGatewayStateRequest{
				GatewayId: gatewayID,
			}))
		})
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}
		b, _ := json.Marshal(e.Msg)

		s.logger.Info("reconciled gateway", "gateway_id", gatewayID, "event", string(b))
		s.gatewaycontroller.BufferEvent(e.Msg)
	}
	s.logger.Info("reconciled gateways", "count", gateways)

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
