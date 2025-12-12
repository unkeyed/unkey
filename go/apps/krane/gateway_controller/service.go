package gatewaycontroller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrlruntime "sigs.k8s.io/controller-runtime"

	"github.com/bytedance/gopkg/util/logger"
	gatewayv1 "github.com/unkeyed/unkey/go/apps/krane/gateway_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type GatewayController struct {
	logger    logging.Logger
	clientset *kubernetes.Clientset
	// incoming events to be applied to gateways
	events *buffer.Buffer[*ctrlv1.GatewayEvent]

	scheme *runtime.Scheme
	mgr    ctrlruntime.Manager
}

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger logging.Logger
	Events *buffer.Buffer[*ctrlv1.GatewayEvent]
}

func New(cfg Config) (*GatewayController, error) {

	ctx := context.Background()

	ctrlruntime.SetLogger(compatibleLogger(cfg.Logger))

	clientset, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	scheme := runtime.NewScheme()

	err = gatewayv1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add gatewayv1 scheme: %w", err)
	}

	mgr, err := k8s.NewManager(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	c := &GatewayController{
		logger:    cfg.Logger,
		clientset: clientset,
		events:    cfg.Events,
		scheme:    scheme,
		mgr:       mgr,
	}

	err = ctrlruntime.NewControllerManagedBy(mgr).
		For(&gatewayv1.Gateway{}). // nolint:exhaustruct
		Complete(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller: %w", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			logger.Error("failed to start manager", "error", err.Error())
		}
	}()

	go c.apply()
	return c, nil
}
