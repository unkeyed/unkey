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
	GetDeploymentFunc        func(context.Context, *ctrlv1.GetDeploymentRequest) (*ctrlv1.GetDeploymentResponse, error)
	GetDeploymentCalls       []*ctrlv1.GetDeploymentRequest
	AuthorizeDeploymentFunc  func(context.Context, *ctrlv1.AuthorizeDeploymentRequest) (*ctrlv1.AuthorizeDeploymentResponse, error)
	AuthorizeDeploymentCalls []*ctrlv1.AuthorizeDeploymentRequest
	CancelDeploymentFunc     func(context.Context, *ctrlv1.CancelDeploymentRequest) (*ctrlv1.CancelDeploymentResponse, error)
	CancelDeploymentCalls    []*ctrlv1.CancelDeploymentRequest
	DeprovisionComputeFunc   func(context.Context, *ctrlv1.DeprovisionComputeRequest) (*ctrlv1.DeprovisionComputeResponse, error)
	DeprovisionComputeCalls  []*ctrlv1.DeprovisionComputeRequest
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

func (m *MockDeploymentClient) DeprovisionCompute(ctx context.Context, req *ctrlv1.DeprovisionComputeRequest) (*ctrlv1.DeprovisionComputeResponse, error) {
	m.mu.Lock()
	m.DeprovisionComputeCalls = append(m.DeprovisionComputeCalls, req)
	m.mu.Unlock()
	if m.DeprovisionComputeFunc != nil {
		return m.DeprovisionComputeFunc(ctx, req)
	}
	return &ctrlv1.DeprovisionComputeResponse{}, nil
}
