package deploymentcontroller

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type DeploymentController struct {
	logger    logging.Logger
	clientset *kubernetes.Clientset

	buffer *buffer.Buffer[*ctrlv1.UpdateDeploymentStatusRequest]
}

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger logging.Logger
	Buffer *buffer.Buffer[*ctrlv1.UpdateDeploymentStatusRequest]
}

func New(cfg Config) (*DeploymentController, error) {

	clientset, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	c := &DeploymentController{
		logger:    cfg.Logger,
		clientset: clientset,
		buffer:    cfg.Buffer,
	}
	err = c.watch()
	if err != nil {
		return nil, fmt.Errorf("failed to start watch: %w", err)
	}

	return c, nil
}
