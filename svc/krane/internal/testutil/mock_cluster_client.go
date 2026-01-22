// Package testutil provides test utilities shared across krane controller tests.
package testutil

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
)

var _ ctrlv1connect.ClusterServiceClient = (*MockClusterClient)(nil)

// MockClusterClient is a test double for the control plane's cluster service.
//
// Each method has an optional function field that tests can set to customize
// behavior. If the function is nil, the method returns a sensible default.
// The mock also records ReportDeploymentStatus and ReportSentinelStatus calls
// so tests can verify the controller reported the correct status.
type MockClusterClient struct {
	WatchDeploymentsFunc          func(context.Context, *connect.Request[ctrlv1.WatchDeploymentsRequest]) (*connect.ServerStreamForClient[ctrlv1.DeploymentState], error)
	WatchSentinelsFunc            func(context.Context, *connect.Request[ctrlv1.WatchSentinelsRequest]) (*connect.ServerStreamForClient[ctrlv1.SentinelState], error)
	GetDesiredSentinelStateFunc   func(context.Context, *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error)
	ReportSentinelStatusFunc      func(context.Context, *connect.Request[ctrlv1.ReportSentinelStatusRequest]) (*connect.Response[ctrlv1.ReportSentinelStatusResponse], error)
	GetDesiredDeploymentStateFunc func(context.Context, *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error)
	ReportDeploymentStatusFunc    func(context.Context, *connect.Request[ctrlv1.ReportDeploymentStatusRequest]) (*connect.Response[ctrlv1.ReportDeploymentStatusResponse], error)
	ReportDeploymentStatusCalls   []*ctrlv1.ReportDeploymentStatusRequest
	ReportSentinelStatusCalls     []*ctrlv1.ReportSentinelStatusRequest
}

func (m *MockClusterClient) WatchDeployments(ctx context.Context, req *connect.Request[ctrlv1.WatchDeploymentsRequest]) (*connect.ServerStreamForClient[ctrlv1.DeploymentState], error) {
	if m.WatchDeploymentsFunc != nil {
		return m.WatchDeploymentsFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockClusterClient) WatchSentinels(ctx context.Context, req *connect.Request[ctrlv1.WatchSentinelsRequest]) (*connect.ServerStreamForClient[ctrlv1.SentinelState], error) {
	if m.WatchSentinelsFunc != nil {
		return m.WatchSentinelsFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockClusterClient) GetDesiredSentinelState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error) {
	if m.GetDesiredSentinelStateFunc != nil {
		return m.GetDesiredSentinelStateFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.SentinelState{}), nil
}

func (m *MockClusterClient) ReportSentinelStatus(ctx context.Context, req *connect.Request[ctrlv1.ReportSentinelStatusRequest]) (*connect.Response[ctrlv1.ReportSentinelStatusResponse], error) {
	m.ReportSentinelStatusCalls = append(m.ReportSentinelStatusCalls, req.Msg)
	if m.ReportSentinelStatusFunc != nil {
		return m.ReportSentinelStatusFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.ReportSentinelStatusResponse{}), nil
}

func (m *MockClusterClient) GetDesiredDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {
	if m.GetDesiredDeploymentStateFunc != nil {
		return m.GetDesiredDeploymentStateFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.DeploymentState{}), nil
}

func (m *MockClusterClient) ReportDeploymentStatus(ctx context.Context, req *connect.Request[ctrlv1.ReportDeploymentStatusRequest]) (*connect.Response[ctrlv1.ReportDeploymentStatusResponse], error) {
	m.ReportDeploymentStatusCalls = append(m.ReportDeploymentStatusCalls, req.Msg)
	if m.ReportDeploymentStatusFunc != nil {
		return m.ReportDeploymentStatusFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.ReportDeploymentStatusResponse{}), nil
}
