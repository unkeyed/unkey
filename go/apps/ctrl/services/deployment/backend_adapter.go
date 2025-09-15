package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
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
