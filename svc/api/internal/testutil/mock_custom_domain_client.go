package testutil

import (
	"context"
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
)

var _ ctrl.CustomDomainServiceClient = (*MockCustomDomainClient)(nil)

// MockCustomDomainClient is a test double for the control plane's custom domain
// service.
//
// Each method has an optional function field that tests can set to customize
// behavior. If the function is nil, the method returns a sensible default.
// The mock also records calls so tests can verify the correct requests were made.
//
// This mock is safe for concurrent use. All call recording is protected by a mutex.
type MockCustomDomainClient struct {
	mu                      sync.Mutex
	AddCustomDomainFunc     func(context.Context, *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error)
	DeleteCustomDomainFunc  func(context.Context, *ctrlv1.DeleteCustomDomainRequest) (*ctrlv1.DeleteCustomDomainResponse, error)
	RetryVerificationFunc   func(context.Context, *ctrlv1.RetryVerificationRequest) (*ctrlv1.RetryVerificationResponse, error)
	AddCustomDomainCalls    []*ctrlv1.AddCustomDomainRequest
	DeleteCustomDomainCalls []*ctrlv1.DeleteCustomDomainRequest
	RetryVerificationCalls  []*ctrlv1.RetryVerificationRequest
}

func (m *MockCustomDomainClient) AddCustomDomain(ctx context.Context, req *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
	m.mu.Lock()
	m.AddCustomDomainCalls = append(m.AddCustomDomainCalls, req)
	m.mu.Unlock()
	if m.AddCustomDomainFunc != nil {
		return m.AddCustomDomainFunc(ctx, req)
	}
	return &ctrlv1.AddCustomDomainResponse{}, nil
}

func (m *MockCustomDomainClient) DeleteCustomDomain(ctx context.Context, req *ctrlv1.DeleteCustomDomainRequest) (*ctrlv1.DeleteCustomDomainResponse, error) {
	m.mu.Lock()
	m.DeleteCustomDomainCalls = append(m.DeleteCustomDomainCalls, req)
	m.mu.Unlock()
	if m.DeleteCustomDomainFunc != nil {
		return m.DeleteCustomDomainFunc(ctx, req)
	}
	return &ctrlv1.DeleteCustomDomainResponse{}, nil
}

func (m *MockCustomDomainClient) RetryVerification(ctx context.Context, req *ctrlv1.RetryVerificationRequest) (*ctrlv1.RetryVerificationResponse, error) {
	m.mu.Lock()
	m.RetryVerificationCalls = append(m.RetryVerificationCalls, req)
	m.mu.Unlock()
	if m.RetryVerificationFunc != nil {
		return m.RetryVerificationFunc(ctx, req)
	}
	return &ctrlv1.RetryVerificationResponse{}, nil
}
