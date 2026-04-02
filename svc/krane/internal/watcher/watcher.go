package watcher

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/deployment"
	"github.com/unkeyed/unkey/svc/krane/internal/sentinel"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
)

const (
	fullSyncInterval = 5 * time.Minute
	reconnectMin     = time.Second
	reconnectMax     = 5 * time.Second
)

// Watcher consumes the unified WatchDeploymentChanges stream and dispatches
// events to the deployment, sentinel, and cilium controllers.
type Watcher struct {
	cluster     ctrl.ClusterServiceClient
	deployments *deployment.Controller
	sentinels   *sentinel.Controller
	cilium      *cilium.Controller
	region      string
	platform    string
}

// Config holds the configuration for creating a new [Watcher].
type Config struct {
	Cluster     ctrl.ClusterServiceClient
	Deployments *deployment.Controller
	Sentinels   *sentinel.Controller
	Cilium      *cilium.Controller
	Region      string
	Platform    string
}

// New creates a [Watcher] ready to be started with [Watcher.Watch].
func New(cfg Config) *Watcher {
	return &Watcher{
		cluster:     cfg.Cluster,
		deployments: cfg.Deployments,
		sentinels:   cfg.Sentinels,
		cilium:      cfg.Cilium,
		region:      cfg.Region,
		platform:    cfg.Platform,
	}
}

// Watch connects to the unified stream and dispatches events. Reconnects with
// jittered backoff on errors. Periodically resets the cursor to 0 for full sync.
// Returns nil when the context is cancelled. Compatible with runner.RunFunc.
func (s *Watcher) Watch(ctx context.Context) error {
	versionLastSeen := uint64(0)
	lastFullSync := time.Now()

	for {
		interval := reconnectMin + time.Millisecond*time.Duration(rand.Float64()*float64(reconnectMax.Milliseconds()-reconnectMin.Milliseconds()))
		time.Sleep(interval)

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if time.Since(lastFullSync) >= fullSyncInterval {
			versionLastSeen = 0
			lastFullSync = time.Now()
			metrics.WatcherFullSyncsTotal.Inc()
			logger.Info("resetting deployment changes cursor for full sync")
		}

		logger.Info("connecting to control plane for deployment changes", "version", versionLastSeen)

		stream, err := s.cluster.WatchDeploymentChanges(ctx, &ctrlv1.WatchDeploymentChangesRequest{
			Region:          s.region,
			VersionLastSeen: versionLastSeen,
		})
		if err != nil {
			logger.Error("error opening deployment changes stream", "error", err)
			continue
		}

		for stream.Receive() {
			event := stream.Msg()

			if err := s.dispatch(ctx, event); err != nil {
				logger.Error("error dispatching deployment change event", "error", err)
				break
			}

			if event.GetVersion() > versionLastSeen {
				versionLastSeen = event.GetVersion()
				metrics.WatcherVersionLastSeen.Set(float64(versionLastSeen))
			}
		}

		if err := stream.Close(); err != nil {
			logger.Error("unable to close deployment changes stream", "error", err)
		}
	}
}

// dispatch routes an event to the appropriate controller. Returns an error if the
// event cannot be dispatched — the caller must not advance the cursor past unhandled events.
func (s *Watcher) dispatch(ctx context.Context, event *ctrlv1.DeploymentChangeEvent) error {
	switch e := event.GetEvent().(type) {
	case *ctrlv1.DeploymentChangeEvent_Deployment:
		if e.Deployment == nil {
			return fmt.Errorf("received deployment change event with nil deployment state at version %d", event.GetVersion())
		}
		switch op := e.Deployment.GetState().(type) {
		case *ctrlv1.DeploymentState_Apply:
			return s.deployments.ApplyDeployment(ctx, op.Apply)
		case *ctrlv1.DeploymentState_Delete:
			return s.deployments.DeleteDeployment(ctx, op.Delete)
		default:
			return fmt.Errorf("unhandled deployment state type %T at version %d", op, event.GetVersion())
		}

	case *ctrlv1.DeploymentChangeEvent_Sentinel:
		if e.Sentinel == nil {
			return fmt.Errorf("received sentinel change event with nil sentinel state at version %d", event.GetVersion())
		}
		switch op := e.Sentinel.GetState().(type) {
		case *ctrlv1.SentinelState_Apply:
			return s.sentinels.ApplySentinel(ctx, op.Apply)
		case *ctrlv1.SentinelState_Delete:
			return s.sentinels.DeleteSentinel(ctx, op.Delete)
		default:
			return fmt.Errorf("unhandled sentinel state type %T at version %d", op, event.GetVersion())
		}

	case *ctrlv1.DeploymentChangeEvent_CiliumNetworkPolicy:
		if e.CiliumNetworkPolicy == nil {
			return fmt.Errorf("received cilium policy change event with nil policy state at version %d", event.GetVersion())
		}
		switch op := e.CiliumNetworkPolicy.GetState().(type) {
		case *ctrlv1.CiliumNetworkPolicyState_Apply:
			return s.cilium.ApplyCiliumNetworkPolicy(ctx, op.Apply)
		case *ctrlv1.CiliumNetworkPolicyState_Delete:
			return s.cilium.DeleteCiliumNetworkPolicy(ctx, op.Delete)
		default:
			return fmt.Errorf("unhandled cilium policy state type %T at version %d", op, event.GetVersion())
		}

	case nil:
		return fmt.Errorf("received deployment change event with nil event at version %d", event.GetVersion())

	default:
		return fmt.Errorf("unhandled deployment change event type %T at version %d", e, event.GetVersion())
	}
}
