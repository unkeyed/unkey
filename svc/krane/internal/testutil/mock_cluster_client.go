// Package testutil provides test utilities shared across krane controller tests.
package testutil

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/pkg/rpc/ctrl"
)

var _ ctrl.ClusterServiceClient = (*MockClusterClient)(nil)

// MockClusterClient is a test double for the control plane's cluster service.
//
// Each method has an optional function field that tests can set to customize
// behavior. If the function is nil, the method returns a sensible default.
// The mock also records ReportDeploymentStatus and ReportSentinelStatus calls
// so tests can verify the controller reported the correct status.
type MockClusterClient struct {
	WatchDeploymentsFunc                   func(context.Context, *ctrlv1.WatchDeploymentsRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentState], error)
	WatchSentinelsFunc                     func(context.Context, *ctrlv1.WatchSentinelsRequest) (*connect.ServerStreamForClient[ctrlv1.SentinelState], error)
	WatchCiliumNetworkPoliciesFunc         func(context.Context, *ctrlv1.WatchCiliumNetworkPoliciesRequest) (*connect.ServerStreamForClient[ctrlv1.CiliumNetworkPolicyState], error)
	GetDesiredSentinelStateFunc            func(context.Context, *ctrlv1.GetDesiredSentinelStateRequest) (*ctrlv1.SentinelState, error)
	ReportSentinelStatusFunc               func(context.Context, *ctrlv1.ReportSentinelStatusRequest) (*ctrlv1.ReportSentinelStatusResponse, error)
	GetDesiredDeploymentStateFunc          func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error)
	ReportDeploymentStatusFunc             func(context.Context, *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error)
	GetDesiredCiliumNetworkPolicyStateFunc func(context.Context, *ctrlv1.GetDesiredCiliumNetworkPolicyStateRequest) (*ctrlv1.CiliumNetworkPolicyState, error)
	ReportDeploymentStatusCalls            []*ctrlv1.ReportDeploymentStatusRequest
	ReportSentinelStatusCalls              []*ctrlv1.ReportSentinelStatusRequest
}

func (m *MockClusterClient) WatchDeployments(ctx context.Context, req *ctrlv1.WatchDeploymentsRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentState], error) {
	if m.WatchDeploymentsFunc != nil {
		return m.WatchDeploymentsFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockClusterClient) WatchSentinels(ctx context.Context, req *ctrlv1.WatchSentinelsRequest) (*connect.ServerStreamForClient[ctrlv1.SentinelState], error) {
	if m.WatchSentinelsFunc != nil {
		return m.WatchSentinelsFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockClusterClient) WatchCiliumNetworkPolicies(ctx context.Context, req *ctrlv1.WatchCiliumNetworkPoliciesRequest) (*connect.ServerStreamForClient[ctrlv1.CiliumNetworkPolicyState], error) {
	if m.WatchCiliumNetworkPoliciesFunc != nil {
		return m.WatchCiliumNetworkPoliciesFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockClusterClient) GetDesiredSentinelState(ctx context.Context, req *ctrlv1.GetDesiredSentinelStateRequest) (*ctrlv1.SentinelState, error) {
	if m.GetDesiredSentinelStateFunc != nil {
		return m.GetDesiredSentinelStateFunc(ctx, req)
	}
	return &ctrlv1.SentinelState{}, nil
}

func (m *MockClusterClient) ReportSentinelStatus(ctx context.Context, req *ctrlv1.ReportSentinelStatusRequest) (*ctrlv1.ReportSentinelStatusResponse, error) {
	m.ReportSentinelStatusCalls = append(m.ReportSentinelStatusCalls, req)
	if m.ReportSentinelStatusFunc != nil {
		return m.ReportSentinelStatusFunc(ctx, req)
	}
	return &ctrlv1.ReportSentinelStatusResponse{}, nil
}

func (m *MockClusterClient) GetDesiredDeploymentState(ctx context.Context, req *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
	if m.GetDesiredDeploymentStateFunc != nil {
		return m.GetDesiredDeploymentStateFunc(ctx, req)
	}
	return &ctrlv1.DeploymentState{}, nil
}

func (m *MockClusterClient) ReportDeploymentStatus(ctx context.Context, req *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
	m.ReportDeploymentStatusCalls = append(m.ReportDeploymentStatusCalls, req)
	if m.ReportDeploymentStatusFunc != nil {
		return m.ReportDeploymentStatusFunc(ctx, req)
	}
	return &ctrlv1.ReportDeploymentStatusResponse{}, nil
}

func (m *MockClusterClient) GetDesiredCiliumNetworkPolicyState(ctx context.Context, req *ctrlv1.GetDesiredCiliumNetworkPolicyStateRequest) (*ctrlv1.CiliumNetworkPolicyState, error) {
	if m.GetDesiredCiliumNetworkPolicyStateFunc != nil {
		return m.GetDesiredCiliumNetworkPolicyStateFunc(ctx, req)
	}
	return &ctrlv1.CiliumNetworkPolicyState{}, nil
}
