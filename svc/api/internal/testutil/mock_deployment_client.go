package testutil

import (
	"context"
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
)

var _ ctrl.DeployServiceClient = (*MockDeploymentClient)(nil)

// MockDeploymentClient is a test double for the control plane's deployment service.
//
// Each method has an optional function field that tests can set to customize
// behavior. If the function is nil, the method returns a sensible default.
// The mock also records calls so tests can verify the correct requests were made.
//
// This mock is safe for concurrent use. All call recording is protected by a mutex.
type MockDeploymentClient struct {
	mu                       sync.Mutex
	CreateDeploymentFunc     func(context.Context, *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error)
	GetDeploymentFunc        func(context.Context, *ctrlv1.GetDeploymentRequest) (*ctrlv1.GetDeploymentResponse, error)
	RollbackFunc             func(context.Context, *ctrlv1.RollbackRequest) (*ctrlv1.RollbackResponse, error)
	PromoteFunc              func(context.Context, *ctrlv1.PromoteRequest) (*ctrlv1.PromoteResponse, error)
	StopDeploymentFunc       func(context.Context, *ctrlv1.StopDeploymentRequest) (*ctrlv1.StopDeploymentResponse, error)
	WakeDeploymentFunc       func(context.Context, *ctrlv1.WakeDeploymentRequest) (*ctrlv1.WakeDeploymentResponse, error)
	CreateDeploymentCalls    []*ctrlv1.CreateDeploymentRequest
	GetDeploymentCalls       []*ctrlv1.GetDeploymentRequest
	RollbackCalls            []*ctrlv1.RollbackRequest
	PromoteCalls             []*ctrlv1.PromoteRequest
	StopDeploymentCalls      []*ctrlv1.StopDeploymentRequest
	WakeDeploymentCalls      []*ctrlv1.WakeDeploymentRequest
	AuthorizeDeploymentFunc  func(context.Context, *ctrlv1.AuthorizeDeploymentRequest) (*ctrlv1.AuthorizeDeploymentResponse, error)
	AuthorizeDeploymentCalls []*ctrlv1.AuthorizeDeploymentRequest
	CancelDeploymentFunc     func(context.Context, *ctrlv1.CancelDeploymentRequest) (*ctrlv1.CancelDeploymentResponse, error)
	CancelDeploymentCalls    []*ctrlv1.CancelDeploymentRequest
	CancelDeployFunc         func(context.Context, *ctrlv1.CancelDeployRequest) (*ctrlv1.CancelDeployResponse, error)
	CancelDeployCalls        []*ctrlv1.CancelDeployRequest
}

func (m *MockDeploymentClient) CreateDeployment(ctx context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
	m.mu.Lock()
	m.CreateDeploymentCalls = append(m.CreateDeploymentCalls, req)
	m.mu.Unlock()
	if m.CreateDeploymentFunc != nil {
		return m.CreateDeploymentFunc(ctx, req)
	}
	return &ctrlv1.CreateDeploymentResponse{}, nil
}

func (m *MockDeploymentClient) GetDeployment(ctx context.Context, req *ctrlv1.GetDeploymentRequest) (*ctrlv1.GetDeploymentResponse, error) {
	m.mu.Lock()
	m.GetDeploymentCalls = append(m.GetDeploymentCalls, req)
	m.mu.Unlock()
	if m.GetDeploymentFunc != nil {
		return m.GetDeploymentFunc(ctx, req)
	}
	return &ctrlv1.GetDeploymentResponse{}, nil
}

func (m *MockDeploymentClient) Rollback(ctx context.Context, req *ctrlv1.RollbackRequest) (*ctrlv1.RollbackResponse, error) {
	m.mu.Lock()
	m.RollbackCalls = append(m.RollbackCalls, req)
	m.mu.Unlock()
	if m.RollbackFunc != nil {
		return m.RollbackFunc(ctx, req)
	}
	return &ctrlv1.RollbackResponse{}, nil
}

func (m *MockDeploymentClient) Promote(ctx context.Context, req *ctrlv1.PromoteRequest) (*ctrlv1.PromoteResponse, error) {
	m.mu.Lock()
	m.PromoteCalls = append(m.PromoteCalls, req)
	m.mu.Unlock()
	if m.PromoteFunc != nil {
		return m.PromoteFunc(ctx, req)
	}
	return &ctrlv1.PromoteResponse{}, nil
}

func (m *MockDeploymentClient) StopDeployment(ctx context.Context, req *ctrlv1.StopDeploymentRequest) (*ctrlv1.StopDeploymentResponse, error) {
	m.mu.Lock()
	m.StopDeploymentCalls = append(m.StopDeploymentCalls, req)
	m.mu.Unlock()
	if m.StopDeploymentFunc != nil {
		return m.StopDeploymentFunc(ctx, req)
	}
	return &ctrlv1.StopDeploymentResponse{}, nil
}

func (m *MockDeploymentClient) WakeDeployment(ctx context.Context, req *ctrlv1.WakeDeploymentRequest) (*ctrlv1.WakeDeploymentResponse, error) {
	m.mu.Lock()
	m.WakeDeploymentCalls = append(m.WakeDeploymentCalls, req)
	m.mu.Unlock()
	if m.WakeDeploymentFunc != nil {
		return m.WakeDeploymentFunc(ctx, req)
	}
	return &ctrlv1.WakeDeploymentResponse{}, nil
}

func (m *MockDeploymentClient) AuthorizeDeployment(ctx context.Context, req *ctrlv1.AuthorizeDeploymentRequest) (*ctrlv1.AuthorizeDeploymentResponse, error) {
	m.mu.Lock()
	m.AuthorizeDeploymentCalls = append(m.AuthorizeDeploymentCalls, req)
	m.mu.Unlock()
	if m.AuthorizeDeploymentFunc != nil {
		return m.AuthorizeDeploymentFunc(ctx, req)
	}
	return &ctrlv1.AuthorizeDeploymentResponse{}, nil
}

func (m *MockDeploymentClient) CancelDeployment(ctx context.Context, req *ctrlv1.CancelDeploymentRequest) (*ctrlv1.CancelDeploymentResponse, error) {
	m.mu.Lock()
	m.CancelDeploymentCalls = append(m.CancelDeploymentCalls, req)
	m.mu.Unlock()
	if m.CancelDeploymentFunc != nil {
		return m.CancelDeploymentFunc(ctx, req)
	}
	return &ctrlv1.CancelDeploymentResponse{}, nil
}

func (m *MockDeploymentClient) CancelDeploy(ctx context.Context, req *ctrlv1.CancelDeployRequest) (*ctrlv1.CancelDeployResponse, error) {
	m.mu.Lock()
	m.CancelDeployCalls = append(m.CancelDeployCalls, req)
	m.mu.Unlock()
	if m.CancelDeployFunc != nil {
		return m.CancelDeployFunc(ctx, req)
	}
	return &ctrlv1.CancelDeployResponse{}, nil
}
