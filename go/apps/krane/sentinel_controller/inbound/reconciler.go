package inbound

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/controlplane"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InboundReconciler struct {
	client client.Client
	logger logging.Logger

	watcher *controlplane.Watcher[ctrlv1.SentinelState]
	cluster ctrlv1connect.ClusterServiceClient
}
type Config struct {
	Client  client.Client
	Logger  logging.Logger
	Watcher *controlplane.Watcher[ctrlv1.SentinelState]
	Cluster ctrlv1connect.ClusterServiceClient
}

func New(cfg Config) *InboundReconciler {

	r := &InboundReconciler{
		client:  cfg.Client,
		logger:  cfg.Logger,
		watcher: cfg.Watcher,
		cluster: cfg.Cluster,
	}

	return r
}

func (r *InboundReconciler) Start(ctx context.Context) {

	queue := buffer.New[*ctrlv1.SentinelState](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "sentinel_controller_inbound_reconciler",
	})

	r.watcher.Sync(ctx, queue)
	r.watcher.Watch(ctx, queue)
	r.refreshCurrentSentinels(ctx, queue)

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-queue.Consume():
			switch state := e.GetState().(type) {
			case *ctrlv1.SentinelState_Apply:
				if err := r.applySentinel(ctx, state.Apply); err != nil {
					r.logger.Error("unable to apply sentinel", "error", err.Error(), "sentinel_id", state.Apply.GetSentinelId())
				}
			case *ctrlv1.SentinelState_Delete:
				if err := r.deleteSentinel(ctx, state.Delete); err != nil {
					r.logger.Error("unable to delete sentinel", "error", err.Error(), "sentinel_id", state.Delete.GetSentinelId())
				}
			default:
				r.logger.Error("Unknown sentinel event", "event", state)
			}
		}
	}

}
