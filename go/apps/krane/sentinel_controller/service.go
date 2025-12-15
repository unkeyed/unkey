package sentinelcontroller

import (
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

type SentinelController struct {
	logger logging.Logger
	// incoming events to be applied to sentinels
	events  *buffer.Buffer[*ctrlv1.SentinelEvent]
	updates *buffer.Buffer[*ctrlv1.UpdateSentinelRequest]
	manager ctrlruntime.Manager
	client  client.Client
	scheme  *runtime.Scheme
}

var _ k8s.Reconciler = (*SentinelController)(nil)

// Config holds configuration for the Kubernetes backend.
type Config struct {
	// Logger for Kubernetes operations.
	Logger  logging.Logger
	Events  *buffer.Buffer[*ctrlv1.SentinelEvent]
	Updates *buffer.Buffer[*ctrlv1.UpdateSentinelRequest]
	Scheme  *runtime.Scheme
	Client  client.Client
}

func New(cfg Config) (*SentinelController, error) {

	ctrlruntime.SetLogger(k8s.CompatibleLogger(cfg.Logger))

	c := &SentinelController{
		logger:  cfg.Logger,
		events:  cfg.Events,
		updates: cfg.Updates,
		client:  cfg.Client,
		scheme:  cfg.Scheme,
	}

	go c.route()

	return c, nil
}
