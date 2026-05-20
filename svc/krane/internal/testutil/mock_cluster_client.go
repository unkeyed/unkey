// Package testutil provides test utilities shared across krane controller tests.
package testutil

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
)

var _ ctrl.ClusterServiceClient = (*MockClusterClient)(nil)

// MockClusterClient is a test double for the control plane's cluster service.
//
// Each method has an optional function field that tests can set to customize
// behavior. If the function is nil, the method returns a sensible default.
// The mock records ReportDeploymentStatus calls so tests can verify the
// controller reported the correct status.
type MockClusterClient struct {
	WatchDeploymentChangesFunc    func(context.Context, *ctrlv1.WatchDeploymentChangesRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error)
	GetDesiredDeploymentStateFunc func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error)
	ReportDeploymentStatusFunc    func(context.Context, *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error)
	ReportInstanceEventsFunc      func(context.Context, *ctrlv1.ReportInstanceEventsRequest) (*ctrlv1.ReportInstanceEventsResponse, error)
	HeartbeatFunc                 func(context.Context, *ctrlv1.HeartbeatRequest) (*ctrlv1.HeartbeatResponse, error)
	SyncDesiredStateFunc          func(context.Context, *ctrlv1.SyncDesiredStateRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error)
	ReportDeploymentStatusCalls   []*ctrlv1.ReportDeploymentStatusRequest
	ReportInstanceEventsCalls     []*ctrlv1.ReportInstanceEventsRequest
}

func (m *MockClusterClient) WatchDeploymentChanges(ctx context.Context, req *ctrlv1.WatchDeploymentChangesRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
	if m.WatchDeploymentChangesFunc != nil {
		return m.WatchDeploymentChangesFunc(ctx, req)
	}
	return nil, nil
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

func (m *MockClusterClient) ReportInstanceEvents(ctx context.Context, req *ctrlv1.ReportInstanceEventsRequest) (*ctrlv1.ReportInstanceEventsResponse, error) {
	m.ReportInstanceEventsCalls = append(m.ReportInstanceEventsCalls, req)
	if m.ReportInstanceEventsFunc != nil {
		return m.ReportInstanceEventsFunc(ctx, req)
	}
	return &ctrlv1.ReportInstanceEventsResponse{}, nil
}

func (m *MockClusterClient) Heartbeat(ctx context.Context, req *ctrlv1.HeartbeatRequest) (*ctrlv1.HeartbeatResponse, error) {
	if m.HeartbeatFunc != nil {
		return m.HeartbeatFunc(ctx, req)
	}
	return &ctrlv1.HeartbeatResponse{}, nil
}

func (m *MockClusterClient) SyncDesiredState(ctx context.Context, req *ctrlv1.SyncDesiredStateRequest) (*connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], error) {
	if m.SyncDesiredStateFunc != nil {
		return m.SyncDesiredStateFunc(ctx, req)
	}
	return nil, nil
}
