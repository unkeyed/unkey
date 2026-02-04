package testutil

import (
	"context"
	"sync"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
)

var _ ctrlv1connect.DeploymentServiceClient = (*MockDeploymentClient)(nil)

// MockDeploymentClient is a test double for the control plane's deployment service.
//
// Each method has an optional function field that tests can set to customize
// behavior. If the function is nil, the method returns a sensible default.
// The mock also records calls so tests can verify the correct requests were made.
//
// This mock is safe for concurrent use. All call recording is protected by a mutex.
type MockDeploymentClient struct {
	mu                    sync.Mutex
	CreateDeploymentFunc  func(context.Context, *connect.Request[ctrlv1.CreateDeploymentRequest]) (*connect.Response[ctrlv1.CreateDeploymentResponse], error)
	GetDeploymentFunc     func(context.Context, *connect.Request[ctrlv1.GetDeploymentRequest]) (*connect.Response[ctrlv1.GetDeploymentResponse], error)
	RollbackFunc          func(context.Context, *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error)
	PromoteFunc           func(context.Context, *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error)
	CreateDeploymentCalls []*ctrlv1.CreateDeploymentRequest
	GetDeploymentCalls    []*ctrlv1.GetDeploymentRequest
	RollbackCalls         []*ctrlv1.RollbackRequest
	PromoteCalls          []*ctrlv1.PromoteRequest
}

func (m *MockDeploymentClient) CreateDeployment(ctx context.Context, req *connect.Request[ctrlv1.CreateDeploymentRequest]) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
	m.mu.Lock()
	m.CreateDeploymentCalls = append(m.CreateDeploymentCalls, req.Msg)
	m.mu.Unlock()
	if m.CreateDeploymentFunc != nil {
		return m.CreateDeploymentFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{}), nil
}

func (m *MockDeploymentClient) GetDeployment(ctx context.Context, req *connect.Request[ctrlv1.GetDeploymentRequest]) (*connect.Response[ctrlv1.GetDeploymentResponse], error) {
	m.mu.Lock()
	m.GetDeploymentCalls = append(m.GetDeploymentCalls, req.Msg)
	m.mu.Unlock()
	if m.GetDeploymentFunc != nil {
		return m.GetDeploymentFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.GetDeploymentResponse{}), nil
}

func (m *MockDeploymentClient) Rollback(ctx context.Context, req *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error) {
	m.mu.Lock()
	m.RollbackCalls = append(m.RollbackCalls, req.Msg)
	m.mu.Unlock()
	if m.RollbackFunc != nil {
		return m.RollbackFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.RollbackResponse{}), nil
}

func (m *MockDeploymentClient) Promote(ctx context.Context, req *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error) {
	m.mu.Lock()
	m.PromoteCalls = append(m.PromoteCalls, req.Msg)
	m.mu.Unlock()
	if m.PromoteFunc != nil {
		return m.PromoteFunc(ctx, req)
	}
	return connect.NewResponse(&ctrlv1.PromoteResponse{}), nil
}
