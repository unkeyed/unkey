package sentinelcontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

type SentinelController struct {
	logger logging.Logger
	// incoming events to be applied to sentinels
	events  *buffer.Buffer[*ctrlv1.SentinelEvent]
	updates *buffer.Buffer[*ctrlv1.UpdateSentinelRequest]
	manager ctrlruntime.Manager
	client  *kubernetes.Clientset
}

var _ k8s.Reconciler = (*SentinelController)(nil)

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger  logging.Logger
	Events  *buffer.Buffer[*ctrlv1.SentinelEvent]
	Updates *buffer.Buffer[*ctrlv1.UpdateSentinelRequest]
	Manager controllerruntime.Manager
	Config  *rest.Config
}

func New(cfg Config) (*SentinelController, error) {

	ctrlruntime.SetLogger(k8s.CompatibleLogger(cfg.Logger))

	client, err := k8s.NewClientWithConfig(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	c := &SentinelController{
		logger:  cfg.Logger,
		events:  cfg.Events,
		updates: cfg.Updates,
		manager: cfg.Manager,
		client:  client,
	}

	hasSynced := false
	for range 5 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		hasSynced = c.manager.GetCache().WaitForCacheSync(ctx)
		cancel()
		if hasSynced {
			break
		}

	}
	if !hasSynced {
		return nil, fmt.Errorf("failed to wait for cache sync: %w", err)
	}

	err = ctrlruntime.NewControllerManagedBy(c.manager).
		For(&sentinelv1.Sentinel{}). // nolint:exhaustruct
		Complete(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller: %w", err)
	}

	go c.route()

	return c, nil
}
