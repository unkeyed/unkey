package deployment

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/deployment/backends"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metald/v1/metaldv1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// DeploymentBackend provides a unified interface for deployment operations
type DeploymentBackend interface {
	Name() string
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

func (m *MetalDBackend) Name() string {
	return "metald"
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

// LocalBackendAdapter wraps a local backend to implement the unified DeploymentBackend interface
type LocalBackendAdapter struct {
	backend backends.DeploymentBackend
	logger  logging.Logger
}

func NewLocalBackendAdapter(backendType string, logger logging.Logger, isRunningDocker bool) (*LocalBackendAdapter, error) {
	backend, err := backends.NewBackend(backendType, logger, isRunningDocker)
	if err != nil {
		return nil, err
	}

	return &LocalBackendAdapter{
		backend: backend,
		logger:  logger,
	}, nil
}

func (f *LocalBackendAdapter) CreateDeployment(ctx context.Context, req *metaldv1.CreateDeploymentRequest) (*metaldv1.CreateDeploymentResponse, error) {
	deployment := req.GetDeployment()
	if deployment == nil {
		return nil, fmt.Errorf("deployment request is nil")
	}

	// Validate image
	image := strings.TrimSpace(deployment.GetImage())
	if image == "" {
		return nil, fmt.Errorf("deployment image cannot be empty or whitespace")
	}

	// Validate VM count
	vmCount := deployment.GetVmCount()
	if vmCount <= 0 {
		return nil, fmt.Errorf("deployment VM count must be greater than 0, got %d", vmCount)
	}

	vmIDs, err := f.backend.CreateDeployment(ctx,
		deployment.GetDeploymentId(),
		image,
		vmCount,
	)
	if err != nil {
		return nil, fmt.Errorf("local backend CreateDeployment failed: %w", err)
	}

	return &metaldv1.CreateDeploymentResponse{
		VmIds: vmIDs,
	}, nil
}

func (f *LocalBackendAdapter) GetDeployment(ctx context.Context, deploymentID string) ([]*metaldv1.GetDeploymentResponse_Vm, error) {
	vms, err := f.backend.GetDeploymentStatus(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("local backend GetDeploymentStatus failed: %w", err)
	}
	return vms, nil
}

func (f *LocalBackendAdapter) Name() string {
	return f.backend.Type()
}

// NewDeploymentBackend creates the appropriate backend based on configuration
func NewDeploymentBackend(metalDClient metaldv1connect.VmServiceClient, fallbackType string, logger logging.Logger, isRunningDocker bool) (DeploymentBackend, error) {
	if fallbackType != "" {
		logger.Info("using local deployment backend", "type", fallbackType)
		return NewLocalBackendAdapter(fallbackType, logger, isRunningDocker)
	}

	if metalDClient == nil {
		return nil, fmt.Errorf("no deployment backend available: metalD client is nil and no fallback configured")
	}

	logger.Info("using metalD deployment backend")
	return NewMetalDBackend(metalDClient, logger), nil
}
