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
	"golang.org/x/sync/semaphore"
)

const (
	fullSyncInterval        = 10 * time.Minute
	reconnectMin            = time.Second
	reconnectMax            = 5 * time.Second
	maxConcurrentDispatches = 10
)

// Watcher consumes the unified WatchDeploymentChanges stream and dispatches
// events to the deployment, sentinel, and cilium controllers.
type Watcher struct {
	cluster     ctrl.ClusterServiceClient
	deployments *deployment.Controller
	sentinels   *sentinel.Controller
	cilium      *cilium.Controller
	sem         *semaphore.Weighted
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
		sem:         semaphore.NewWeighted(maxConcurrentDispatches),
		region:      cfg.Region,
		platform:    cfg.Platform,
	}
}

func (s *Watcher) regionKey() *ctrlv1.RegionKey {
	return &ctrlv1.RegionKey{Platform: s.platform, Name: s.region}
}

// Watch runs two independent loops:
//   - A real-time incremental stream for fast delivery of new changes.
//   - A periodic full sync to reconcile any drift.
//
// Both share a semaphore so the k8s API is not overwhelmed.
// Returns nil when the context is cancelled.
func (s *Watcher) Watch(ctx context.Context) error {
	go s.runPeriodicFullSync(ctx)
	s.runStream(ctx)
	return nil
}

// runStream maintains a long-lived incremental stream. On first connect it
// sends version=0 and the server jumps to the current max version. On
// reconnect it resumes from the last seen version.
func (s *Watcher) runStream(ctx context.Context) {
	versionLastSeen := uint64(0)

	for {
		jitter := reconnectMin + time.Millisecond*time.Duration(rand.Float64()*float64(reconnectMax.Milliseconds()-reconnectMin.Milliseconds()))
		time.Sleep(jitter)

		select {
		case <-ctx.Done():
			return
		default:
		}

		stream, err := s.cluster.WatchDeploymentChanges(ctx, &ctrlv1.WatchDeploymentChangesRequest{
			Region:          s.regionKey(),
			VersionLastSeen: versionLastSeen,
		})
		if err != nil {
			metrics.StreamConnectionsTotal.WithLabelValues("error").Inc()
			logger.Error("stream: error opening connection", "error", err)
			continue
		}
		metrics.StreamConnectionsTotal.WithLabelValues("success").Inc()

		for stream.Receive() {
			event := stream.Msg()
			metrics.StreamEventsReceivedTotal.Inc()

			if err := s.sem.Acquire(ctx, 1); err != nil {
				break
			}
			go func() {
				defer s.sem.Release(1)
				resourceType := eventResourceType(event)
				if err := s.dispatch(ctx, event); err != nil {
					metrics.DispatchTotal.WithLabelValues("stream", resourceType, "error").Inc()
					logger.Error("stream: error dispatching event", "error", err, "version", event.GetVersion())
				} else {
					metrics.DispatchTotal.WithLabelValues("stream", resourceType, "success").Inc()
				}
			}()

			if event.GetVersion() > versionLastSeen {
				versionLastSeen = event.GetVersion()
				metrics.WatcherVersionLastSeen.Set(float64(versionLastSeen))
			}
		}

		if err := stream.Close(); err != nil && ctx.Err() == nil {
			logger.Error("stream: error closing connection", "error", err)
		}
	}
}

// runPeriodicFullSync calls SyncDesiredState every fullSyncInterval to
// reconcile the full desired state. Runs independently of the incremental
// stream so it never blocks real-time event delivery.
func (s *Watcher) runPeriodicFullSync(ctx context.Context) {
	// Run one immediately on startup.
	s.doFullSync(ctx)

	ticker := time.NewTicker(fullSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doFullSync(ctx)
		}
	}
}

func (s *Watcher) doFullSync(ctx context.Context) {
	metrics.WatcherFullSyncsTotal.Inc()
	start := time.Now()

	stream, err := s.cluster.SyncDesiredState(ctx, &ctrlv1.SyncDesiredStateRequest{
		Region: s.regionKey(),
	})
	if err != nil {
		logger.Error("full sync: error opening connection", "error", err)
		return
	}

	for stream.Receive() {
		event := stream.Msg()
		metrics.FullSyncEventsReceivedTotal.Inc()

		if err := s.sem.Acquire(ctx, 1); err != nil {
			break
		}
		go func() {
			defer s.sem.Release(1)
			resourceType := eventResourceType(event)
			if err := s.dispatch(ctx, event); err != nil {
				metrics.DispatchTotal.WithLabelValues("full_sync", resourceType, "error").Inc()
				logger.Error("full sync: error dispatching event", "error", err)
			} else {
				metrics.DispatchTotal.WithLabelValues("full_sync", resourceType, "success").Inc()
			}
		}()
	}

	if err := stream.Close(); err != nil && ctx.Err() == nil {
		logger.Error("full sync: error closing connection", "error", err)
	}

	metrics.FullSyncDurationSeconds.Observe(time.Since(start).Seconds())
}

// eventResourceType returns a label-safe resource type string for metrics.
func eventResourceType(event *ctrlv1.DeploymentChangeEvent) string {
	switch event.GetEvent().(type) {
	case *ctrlv1.DeploymentChangeEvent_Deployment:
		return "deployment"
	case *ctrlv1.DeploymentChangeEvent_Sentinel:
		return "sentinel"
	case *ctrlv1.DeploymentChangeEvent_CiliumNetworkPolicy:
		return "cilium_network_policy"
	default:
		return "unknown"
	}
}

// dispatch routes an event to the appropriate controller.
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
