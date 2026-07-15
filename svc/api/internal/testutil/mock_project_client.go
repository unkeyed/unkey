package testutil

import (
	"context"
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/uid"
)

var _ ctrl.ProjectServiceClient = (*MockProjectClient)(nil)

// MockProjectClient is a test double for the control plane's project service.
//
// Each method has an optional function field that tests can set to customize
// behavior. If the function is nil, the method returns a sensible default.
// The mock also records calls so tests can verify the correct requests were made.
//
// This mock is safe for concurrent use. All call recording is protected by a mutex.
type MockProjectClient struct {
	mu                 sync.Mutex
	CreateProjectFunc  func(context.Context, *ctrlv1.CreateProjectRequest) (*ctrlv1.CreateProjectResponse, error)
	DeleteProjectFunc  func(context.Context, *ctrlv1.DeleteProjectRequest) (*ctrlv1.DeleteProjectResponse, error)
	CreateProjectCalls []*ctrlv1.CreateProjectRequest
	DeleteProjectCalls []*ctrlv1.DeleteProjectRequest
}

func (m *MockProjectClient) CreateProject(ctx context.Context, req *ctrlv1.CreateProjectRequest) (*ctrlv1.CreateProjectResponse, error) {
	m.mu.Lock()
	m.CreateProjectCalls = append(m.CreateProjectCalls, req)
	m.mu.Unlock()
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, req)
	}
	return &ctrlv1.CreateProjectResponse{Id: uid.New(uid.ProjectPrefix)}, nil
}

func (m *MockProjectClient) DeleteProject(ctx context.Context, req *ctrlv1.DeleteProjectRequest) (*ctrlv1.DeleteProjectResponse, error) {
	m.mu.Lock()
	m.DeleteProjectCalls = append(m.DeleteProjectCalls, req)
	m.mu.Unlock()
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(ctx, req)
	}
	return &ctrlv1.DeleteProjectResponse{}, nil
}
