package gatewaycontroller

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type GatewayController struct {
	logger    logging.Logger
	clientset *kubernetes.Clientset
	events    *buffer.Buffer[*ctrlv1.GatewayEvent]
	updates   *buffer.Buffer[*ctrlv1.UpdateGatewayRequest]
}

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger  logging.Logger
	Events  *buffer.Buffer[*ctrlv1.GatewayEvent]
	Updates *buffer.Buffer[*ctrlv1.UpdateGatewayRequest]
}

func New(cfg Config) (*GatewayController, error) {

	clientset, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	c := &GatewayController{
		logger:    cfg.Logger,
		clientset: clientset,
		events:    cfg.Events,
		updates:   cfg.Updates,
	}

	if err = c.watch(); err != nil {
		return nil, fmt.Errorf("failed to start watch: %w", err)
	}

	go c.apply()
	return c, nil
}
