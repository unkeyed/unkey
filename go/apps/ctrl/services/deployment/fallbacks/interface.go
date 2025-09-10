package fallbacks

import (
	"context"
	"fmt"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// DeploymentBackend defines the interface for deployment backends
type DeploymentBackend interface {
	// CreateDeployment creates a new deployment and returns VM IDs
	CreateDeployment(ctx context.Context, deploymentID string, image string, vmCount int32) ([]string, error)

	// GetDeploymentStatus checks if deployment VMs are ready
	GetDeploymentStatus(ctx context.Context, deploymentID string) ([]*metaldv1.GetDeploymentResponse_Vm, error)

	// DeleteDeployment removes a deployment and its resources
	DeleteDeployment(ctx context.Context, deploymentID string) error

	// Type returns the backend type name
	Type() string
}

// NewBackend creates a new deployment backend based on the specified type
func NewBackend(backendType string, logger logging.Logger) (DeploymentBackend, error) {
	switch backendType {
	case "k8s":
		return NewK8sBackend(logger)
	case "docker":
		return NewDockerBackend(logger)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}
