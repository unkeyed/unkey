package krane

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type gatewayManager struct {
	logger  logging.Logger
	backend backend.Backend
}

func (m *gatewayManager) HandleApply(ctx context.Context, event *ctrlv1.ApplyGateway) error {
	return m.backend.CreateGateway(ctx, backend.CreateGatewayRequest{
		Namespace:     event.GetNamespace(),
		WorkspaceID:   event.GetWorkspaceId(),
		ProjectID:     event.GetProjectId(),
		EnvironmentID: event.GetEnvironmentId(),
		GatewayID:     event.GetGatewayId(),
		Image:         event.GetImage(),
		Replicas:      int(event.GetReplicas()),
		CpuMillicores: uint32(event.GetCpuMillicores()),
		MemorySizeMib: uint64(event.GetMemorySizeMib()),
	})

}

func (m *gatewayManager) HandleDelete(ctx context.Context, event *ctrlv1.DeleteGateway) error {
	return m.backend.DeleteGateway(ctx, backend.DeleteGatewayRequest{
		Namespace: event.GetNamespace(),
		GatewayID: event.GetGatewayId(),
	})

}
