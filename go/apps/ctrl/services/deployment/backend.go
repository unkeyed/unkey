package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/deployment/fallbacks"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metald/v1/metaldv1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// DeploymentBackend provides a unified interface for deployment operations
type DeploymentBackend interface {
	CreateDeployment(ctx context.Context, req *metaldv1.CreateDeploymentRequest) (*metaldv1.CreateDeploymentResponse, error)
	GetDeployment(ctx context.Context, deploymentID string) ([]*metaldv1.GetDeploymentResponse_Vm, error)
}

// MetalDBackend implements DeploymentBackend using the metalD service
type MetalDBackend struct {
	client metaldv1connect.VmServiceClient
	logger logging.Logger
}

func NewMetalDBackend(client metaldv1connect.VmServiceClient, logger logging.Logger) *MetalDBackend {
	return &MetalDBackend{
		client: client,
		logger: logger,
	}
}

func (m *MetalDBackend) CreateDeployment(ctx context.Context, req *metaldv1.CreateDeploymentRequest) (*metaldv1.CreateDeploymentResponse, error) {
	resp, err := m.client.CreateDeployment(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("metald CreateDeployment failed: %w", err)
	}
	return resp.Msg, nil
}

func (m *MetalDBackend) GetDeployment(ctx context.Context, deploymentID string) ([]*metaldv1.GetDeploymentResponse_Vm, error) {
	resp, err := m.client.GetDeployment(ctx, connect.NewRequest(&metaldv1.GetDeploymentRequest{
		DeploymentId: deploymentID,
	}))
	if err != nil {
		return nil, fmt.Errorf("metald GetDeployment failed: %w", err)
	}
	return resp.Msg.GetVms(), nil
}

// FallbackBackend wraps a fallback backend to implement the unified DeploymentBackend interface
type FallbackBackend struct {
	backend fallbacks.DeploymentBackend
	logger  logging.Logger
}

func NewFallbackBackend(backendType string, logger logging.Logger) (*FallbackBackend, error) {
	backend, err := fallbacks.NewBackend(backendType, logger)
	if err != nil {
		return nil, err
	}
	return &FallbackBackend{
		backend: backend,
		logger:  logger,
	}, nil
}

func (f *FallbackBackend) CreateDeployment(ctx context.Context, req *metaldv1.CreateDeploymentRequest) (*metaldv1.CreateDeploymentResponse, error) {
	deployment := req.GetDeployment()
	if deployment == nil {
		return nil, fmt.Errorf("deployment request is nil")
	}

	vmIDs, err := f.backend.CreateDeployment(ctx,
		deployment.GetDeploymentId(),
		deployment.GetImage(),
		int32(deployment.GetVmCount()))
	if err != nil {
		return nil, fmt.Errorf("fallback CreateDeployment failed: %w", err)
	}

	return &metaldv1.CreateDeploymentResponse{
		VmIds: vmIDs,
	}, nil
}

func (f *FallbackBackend) GetDeployment(ctx context.Context, deploymentID string) ([]*metaldv1.GetDeploymentResponse_Vm, error) {
	vms, err := f.backend.GetDeploymentStatus(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("fallback GetDeploymentStatus failed: %w", err)
	}
	return vms, nil
}

// NewDeploymentBackend creates the appropriate backend based on configuration
func NewDeploymentBackend(metalDClient metaldv1connect.VmServiceClient, fallbackType string, logger logging.Logger) (DeploymentBackend, error) {
	if fallbackType != "" {
		logger.Info("using fallback deployment backend", "type", fallbackType)
		return NewFallbackBackend(fallbackType, logger)
	}

	if metalDClient == nil {
		return nil, fmt.Errorf("no deployment backend available: metalD client is nil and no fallback configured")
	}

	logger.Info("using metalD deployment backend")
	return NewMetalDBackend(metalDClient, logger), nil
}
