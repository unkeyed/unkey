package testutil

import (
	"context"
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
)

var _ ctrl.PortalDomainServiceClient = (*MockPortalDomainClient)(nil)

// MockPortalDomainClient is a test double for the control plane's portal domain
// service. A nil function field returns an empty response, and every call is
// recorded so tests can assert what reached ctrl. Safe for concurrent use.
type MockPortalDomainClient struct {
	mu                      sync.Mutex
	AddPortalDomainFunc     func(context.Context, *ctrlv1.AddPortalDomainRequest) (*ctrlv1.AddPortalDomainResponse, error)
	DeletePortalDomainFunc  func(context.Context, *ctrlv1.DeletePortalDomainRequest) (*ctrlv1.DeletePortalDomainResponse, error)
	RetryVerificationFunc   func(context.Context, *ctrlv1.RetryPortalDomainVerificationRequest) (*ctrlv1.RetryPortalDomainVerificationResponse, error)
	AddPortalDomainCalls    []*ctrlv1.AddPortalDomainRequest
	DeletePortalDomainCalls []*ctrlv1.DeletePortalDomainRequest
	RetryVerificationCalls  []*ctrlv1.RetryPortalDomainVerificationRequest
}

func (m *MockPortalDomainClient) AddPortalDomain(ctx context.Context, req *ctrlv1.AddPortalDomainRequest) (*ctrlv1.AddPortalDomainResponse, error) {
	m.mu.Lock()
	m.AddPortalDomainCalls = append(m.AddPortalDomainCalls, req)
	m.mu.Unlock()
	if m.AddPortalDomainFunc != nil {
		return m.AddPortalDomainFunc(ctx, req)
	}
	return &ctrlv1.AddPortalDomainResponse{}, nil
}

func (m *MockPortalDomainClient) DeletePortalDomain(ctx context.Context, req *ctrlv1.DeletePortalDomainRequest) (*ctrlv1.DeletePortalDomainResponse, error) {
	m.mu.Lock()
	m.DeletePortalDomainCalls = append(m.DeletePortalDomainCalls, req)
	m.mu.Unlock()
	if m.DeletePortalDomainFunc != nil {
		return m.DeletePortalDomainFunc(ctx, req)
	}
	return &ctrlv1.DeletePortalDomainResponse{}, nil
}

func (m *MockPortalDomainClient) RetryVerification(ctx context.Context, req *ctrlv1.RetryPortalDomainVerificationRequest) (*ctrlv1.RetryPortalDomainVerificationResponse, error) {
	m.mu.Lock()
	m.RetryVerificationCalls = append(m.RetryVerificationCalls, req)
	m.mu.Unlock()
	if m.RetryVerificationFunc != nil {
		return m.RetryVerificationFunc(ctx, req)
	}
	return &ctrlv1.RetryPortalDomainVerificationResponse{}, nil
}
