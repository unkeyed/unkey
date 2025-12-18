package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewClient creates a Kubernetes client using in-cluster configuration.
//
// This function automatically detects and uses the service account configuration
// available within a Kubernetes cluster. It's the standard way to create
// a client when running inside a pod with proper RBAC permissions.
//
// The client is configured with the provided scheme for type registration
// and uses the default in-cluster configuration (service account token,
// API server discovery, etc.).
//
// Parameters:
//   - scheme: Kubernetes runtime scheme for type registration
//
// Returns an error if in-cluster configuration cannot be detected or
// if the client cannot be created with the provided scheme.
func NewClient(scheme *runtime.Scheme) (client.Client, error) {
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	return client.New(inClusterConfig, client.Options{Scheme: scheme})
}

// NewManager creates a controller-runtime manager using in-cluster configuration.
//
// This function initializes a controller-runtime manager that can host
// multiple controllers and handle webhook admission. The manager uses
// the same in-cluster configuration as NewClient() and is configured
// with the provided scheme for type registration.
//
// The manager provides:
//   - Shared client for all controllers
//   - Cache for efficient Kubernetes API access
//   - Leader election for multi-replica deployments
//   - Graceful shutdown handling
//
// Parameters:
//   - scheme: Kubernetes runtime scheme for type registration
//
// Returns an error if in-cluster configuration cannot be detected or
// if the manager cannot be initialized with the provided scheme.
func NewManager(scheme *runtime.Scheme) (controllerruntime.Manager, error) {
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	// nolint:exhaustruct
	return controllerruntime.NewManager(inClusterConfig, manager.Options{
		Scheme: scheme,
	})
}
