package reconciler

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
)

var _ ctrlv1connect.ClusterServiceClient = (*MockClusterClient)(nil)

// MockClusterClient is a test double for the control plane's cluster service.
//
// Each method has an optional function field (e.g., WatchFunc) that tests can set
// to customize behavior. If the function is nil, the method returns a sensible
// default. The mock also records all UpdateDeploymentState and UpdateSentinelState
// calls so tests can verify the reconciler reported the correct state.
type MockClusterClient struct {
	SyncFunc                      func(context.Context, *connect.Request[ctrlv1.SyncRequest]) (*connect.ServerStreamForClient[ctrlv1.State], error)
	GetDesiredSentinelStateFunc   func(context.Context, *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error)
	UpdateSentinelStateFunc       func(context.Context, *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error)
	GetDesiredDeploymentStateFunc func(context.Context, *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error)
	UpdateDeploymentStateFunc     func(context.Context, *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error)
	UpdateDeploymentStateCalls    []*ctrlv1.UpdateDeploymentStateRequest
	UpdateSentinelStateCalls      []*ctrlv1.UpdateSentinelStateRequest
}

func (m *MockClusterClient) Sync(ctx context.Context, req *connect.Request[ctrlv1.SyncRequest]) (*connect.ServerStreamForClient[ctrlv1.State], error) {
	if m.SyncFunc != nil {
		return m.SyncFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockClusterClient) GetDesiredSentinelState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error) {
	if m.GetDesiredSentinelStateFunc != nil {
		return m.GetDesiredSentinelStateFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.SentinelState{}), nil
}

func (m *MockClusterClient) UpdateSentinelState(ctx context.Context, req *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
	m.UpdateSentinelStateCalls = append(m.UpdateSentinelStateCalls, req.Msg)
	if m.UpdateSentinelStateFunc != nil {
		return m.UpdateSentinelStateFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.UpdateSentinelStateResponse{}), nil
}

func (m *MockClusterClient) GetDesiredDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {
	if m.GetDesiredDeploymentStateFunc != nil {
		return m.GetDesiredDeploymentStateFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.DeploymentState{}), nil
}

func (m *MockClusterClient) UpdateDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
	m.UpdateDeploymentStateCalls = append(m.UpdateDeploymentStateCalls, req.Msg)
	if m.UpdateDeploymentStateFunc != nil {
		return m.UpdateDeploymentStateFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.UpdateDeploymentStateResponse{}), nil
}
