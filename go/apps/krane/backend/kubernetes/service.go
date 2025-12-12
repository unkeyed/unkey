package kubernetes

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// k8s implements kranev1connect.DeploymentServiceHandler using the Kubernetes API.
type k8s struct {
	logger    logging.Logger
	clientset *kubernetes.Clientset
	region    string
	kranev1connect.UnimplementedDeploymentServiceHandler
	kranev1connect.UnimplementedGatewayServiceHandler
}

var _ kranev1connect.DeploymentServiceHandler = (*k8s)(nil)
var _ kranev1connect.GatewayServiceHandler = (*k8s)(nil)

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger logging.Logger

	// Region where this krane instance is deployed.
	Region string

	// DeploymentEvictionTTL for automatic cleanup of old deployments.
	// Set to 0 to to disable automatic eviction.
	DeploymentEvictionTTL time.Duration
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
		UnimplementedDeploymentServiceHandler: kranev1connect.UnimplementedDeploymentServiceHandler{},
		UnimplementedGatewayServiceHandler:    kranev1connect.UnimplementedGatewayServiceHandler{},
		logger:                                cfg.Logger,
		clientset:                             clientset,
		region:                                cfg.Region,
	}

	return k, nil
}

// GetClientset returns the Kubernetes clientset for use by other services
// such as the token validator for service account authentication.
func (k *k8s) GetClientset() kubernetes.Interface {
	return k.clientset
}
