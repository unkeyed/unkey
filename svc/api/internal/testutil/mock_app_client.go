package testutil

import (
	"context"
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
)

var _ ctrl.AppServiceClient = (*MockAppClient)(nil)

// MockAppClient is a test double for the control plane's app service.
//
// Each method has an optional function field that tests can set to customize
// behavior. If the function is nil, the method returns a sensible default.
// The mock also records calls so tests can verify the correct requests were made.
//
// This mock is safe for concurrent use. All call recording is protected by a mutex.
type MockAppClient struct {
	mu             sync.Mutex
	CreateAppFunc  func(context.Context, *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error)
	DeleteAppFunc  func(context.Context, *ctrlv1.DeleteAppRequest) (*ctrlv1.DeleteAppResponse, error)
	CreateAppCalls []*ctrlv1.CreateAppRequest
	DeleteAppCalls []*ctrlv1.DeleteAppRequest
}

func (m *MockAppClient) CreateApp(ctx context.Context, req *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
	m.mu.Lock()
	m.CreateAppCalls = append(m.CreateAppCalls, req)
	m.mu.Unlock()
	if m.CreateAppFunc != nil {
		return m.CreateAppFunc(ctx, req)
	}
	return &ctrlv1.CreateAppResponse{}, nil
}

func (m *MockAppClient) DeleteApp(ctx context.Context, req *ctrlv1.DeleteAppRequest) (*ctrlv1.DeleteAppResponse, error) {
	m.mu.Lock()
	m.DeleteAppCalls = append(m.DeleteAppCalls, req)
	m.mu.Unlock()
	if m.DeleteAppFunc != nil {
		return m.DeleteAppFunc(ctx, req)
	}
	return &ctrlv1.DeleteAppResponse{}, nil
}
