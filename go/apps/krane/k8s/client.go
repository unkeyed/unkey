package k8s

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
)

func NewClient() (*kubernetes.Clientset, error) {

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	return kubernetes.NewForConfig(inClusterConfig)

}

func NewManager(scheme *runtime.Scheme) (ctrlruntime.Manager, error) {

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}
	// nolint:exhaustruct
	mgr, err := ctrlruntime.NewManager(inClusterConfig, ctrlruntime.Options{
		Scheme: scheme,
		Controller: config.Controller{
			MaxConcurrentReconciles: 1,
			ReconciliationTimeout:   10 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	return mgr, nil
}
