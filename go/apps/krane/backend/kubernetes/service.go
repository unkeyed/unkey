package kubernetes

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// k8s implements backend.Backend using the Kubernetes API.
type k8s struct {
	logger    logging.Logger
	clientset *kubernetes.Clientset
}

var _ backend.Backend = (*k8s)(nil)

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger logging.Logger
}

// New creates a Kubernetes backend using in-cluster configuration.
//
// Requires RBAC permissions for StatefulSets, Services, and Pods.
// Starts automatic eviction if DeploymentEvictionTTL > 0.
func New(cfg Config) (*k8s, error) {
	// Create in-cluster config
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}
	k := &k8s{
		logger:    cfg.Logger,
		clientset: clientset,
	}

	return k, nil
}
