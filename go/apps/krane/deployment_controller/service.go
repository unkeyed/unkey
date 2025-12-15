package deploymentcontroller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrlruntime "sigs.k8s.io/controller-runtime"

	"github.com/bytedance/gopkg/util/logger"
	v1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type DeploymentController struct {
	logger    logging.Logger
	clientset *kubernetes.Clientset
	// incoming events to be applied to deployments
	events  *buffer.Buffer[*ctrlv1.DeploymentEvent]
	updates *buffer.Buffer[*ctrlv1.UpdateInstanceRequest]

	scheme  *runtime.Scheme
	manager ctrlruntime.Manager
}

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger  logging.Logger
	Events  *buffer.Buffer[*ctrlv1.DeploymentEvent]
	Updates *buffer.Buffer[*ctrlv1.UpdateInstanceRequest]
	Client  *kubernetes.Clientset
	Manager ctrlruntime.Manager
}

func New(cfg Config) (*DeploymentController, error) {

	ctx := context.Background()

	ctrlruntime.SetLogger(k8s.CompatibleLogger(cfg.Logger))

	// Get the scheme from the provided manager
	scheme := cfg.Manager.GetScheme()

	var clientset *kubernetes.Clientset

	// If client is provided, use it, otherwise create from manager's config
	if cfg.Client != nil {
		clientset = cfg.Client
	} else {
		var err error
		clientset, err = kubernetes.NewForConfig(cfg.Manager.GetConfig())
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset from manager config: %w", err)
		}
	}

	c := &DeploymentController{
		logger:    cfg.Logger,
		clientset: clientset,
		events:    cfg.Events,
		updates:   cfg.Updates,
		scheme:    scheme,
		manager:   cfg.Manager,
	}

	err := ctrlruntime.NewControllerManagedBy(cfg.Manager).
		For(&v1.UnkeyDeployment{}). // nolint:exhaustruct
		Complete(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller: %w", err)
	}

	go func() {
		if err := c.manager.Start(ctx); err != nil {
			logger.Error("failed to start manager", "error", err.Error())
		}
	}()

	if err = c.watch(); err != nil {
		return nil, fmt.Errorf("failed to start watch: %w", err)
	}

	go c.apply()

	return c, nil
}
