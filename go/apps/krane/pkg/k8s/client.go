package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func NewClient() (*kubernetes.Clientset, error) {

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	return NewClientWithConfig(inClusterConfig)
}

// NewClientWithConfig creates a new Kubernetes clientset with the given configuration.
// This is useful for testing where we need to provide a custom configuration.
func NewClientWithConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

// NewManagerWithConfig creates a new controller-runtime manager with the given configuration.
// This is useful for testing where we need to provide a custom configuration.
func NewManagerWithConfig(config *rest.Config, scheme *runtime.Scheme) (ctrlruntime.Manager, error) {
	// nolint:exhaustruct
	mgr, err := ctrlruntime.NewManager(config, ctrlruntime.Options{
		Scheme:                 scheme,
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		HealthProbeBindAddress: ":8081",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	return mgr, nil
}
