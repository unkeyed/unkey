package krane

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type deploymentManager struct {
	logger  logging.Logger
	backend backend.Backend
}

func (m *deploymentManager) HandleApply(ctx context.Context, event *ctrlv1.ApplyDeployment) error {
	return m.backend.CreateDeployment(ctx, backend.CreateDeploymentRequest{
		Namespace:     event.GetNamespace(),
		WorkspaceID:   event.GetWorkspaceId(),
		ProjectID:     event.GetProjectId(),
		EnvironmentID: event.GetEnvironmentId(),
		DeploymentID:  event.GetDeploymentId(),
		Image:         event.GetImage(),
		Replicas:      int(event.GetReplicas()),
		CpuMillicores: uint32(event.GetCpuMillicores()),
		MemorySizeMib: uint64(event.GetMemorySizeMib()),
	})

}

func (m *deploymentManager) HandleDelete(ctx context.Context, event *ctrlv1.DeleteDeployment) error {
	return m.backend.DeleteDeployment(ctx, backend.DeleteDeploymentRequest{
		Namespace:    event.GetNamespace(),
		DeploymentID: event.GetDeploymentId(),
	})

}
