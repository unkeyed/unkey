package sentinelcontroller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrlruntime "sigs.k8s.io/controller-runtime"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type SentinelController struct {
	logger    logging.Logger
	clientset *kubernetes.Clientset
	// incoming events to be applied to sentinels
	events *buffer.Buffer[*ctrlv1.SentinelEvent]

	scheme *runtime.Scheme
	mgr    ctrlruntime.Manager
}

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger logging.Logger
	Events *buffer.Buffer[*ctrlv1.SentinelEvent]
}

func New(cfg Config) (*SentinelController, error) {

	ctx := context.Background()

	ctrlruntime.SetLogger(compatibleLogger(cfg.Logger))

	clientset, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	scheme := runtime.NewScheme()

	err = sentinelv1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add sentinelv1 scheme: %w", err)
	}

	mgr, err := k8s.NewManager(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	c := &SentinelController{
		logger:    cfg.Logger,
		clientset: clientset,
		events:    cfg.Events,
		scheme:    scheme,
		mgr:       mgr,
	}

	err = ctrlruntime.NewControllerManagedBy(mgr).
		For(&sentinelv1.Sentinel{}). // nolint:exhaustruct
		Complete(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller: %w", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			logger.Error("failed to start manager", "error", err.Error())
		}
	}()

	go c.route()
	return c, nil
}
