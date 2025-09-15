package backends

import (
	"context"
	"fmt"
	"strings"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Backend type constants
const (
	BackendTypeK8s    = "k8s"
	BackendTypeDocker = "docker"
)

// DeploymentBackend defines the interface for deployment backends
type DeploymentBackend interface {
	// CreateDeployment creates a new deployment and returns VM IDs
	CreateDeployment(ctx context.Context, deploymentID string, image string, vmCount uint32) ([]string, error)

	// GetDeploymentStatus checks if deployment VMs are ready
	GetDeploymentStatus(ctx context.Context, deploymentID string) ([]*metaldv1.GetDeploymentResponse_Vm, error)

	// DeleteDeployment removes a deployment and its resources
	DeleteDeployment(ctx context.Context, deploymentID string) error

	// Type returns the backend type name
	Type() string
}

// ValidateBackendType checks if a backend type is valid
// Returns nil if valid, error if invalid
func ValidateBackendType(backendType string) error {
	if backendType == "" {
		return nil // Empty string is valid (means no fallback)
	}

	// Normalize backend type to lowercase for case-insensitive comparison
	normalizedType := strings.ToLower(strings.TrimSpace(backendType))

	switch normalizedType {
	case BackendTypeK8s, BackendTypeDocker:
		return nil
	default:
		return fmt.Errorf("unsupported backend type: %s; allowed values: %q, %q, or \"\" (empty)", backendType, BackendTypeK8s, BackendTypeDocker)
	}
}

// NewBackend creates a new deployment backend based on the specified type
func NewBackend(backendType string, logger logging.Logger, isRunningDocker bool) (DeploymentBackend, error) {
	// Validate backend type first
	if err := ValidateBackendType(backendType); err != nil {
		return nil, err
	}

	// Normalize backend type to lowercase for case-insensitive comparison
	normalizedType := strings.ToLower(strings.TrimSpace(backendType))

	switch normalizedType {
	case BackendTypeK8s:
		return NewK8sBackend(logger)
	case BackendTypeDocker:
		return NewDockerBackend(logger, isRunningDocker)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s; allowed values: %q, %q", backendType, BackendTypeK8s, BackendTypeDocker)
	}
}
