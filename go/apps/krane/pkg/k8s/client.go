package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewClient creates a Kubernetes clientset using in-cluster configuration.
//
// This function automatically detects and uses the service account configuration
// available within a Kubernetes cluster. It's the standard way to create
// a client when running inside a pod.
//
// Returns an error if in-cluster configuration cannot be detected or
// if the clientset cannot be created.
func NewClient(scheme *runtime.Scheme) (client.Client, error) {

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	return client.New(inClusterConfig, client.Options{Scheme: scheme})
}

func NewManager(scheme *runtime.Scheme) (controllerruntime.Manager, error) {

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	return controllerruntime.NewManager(inClusterConfig, manager.Options{
		Scheme: scheme,
	})
}
