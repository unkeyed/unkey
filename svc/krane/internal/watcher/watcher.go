package watcher

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/deployment"
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
// events to the deployment and cilium controllers.
type Watcher struct {
	cluster     ctrl.ClusterServiceClient
	deployments *deployment.Controller
	sem         *semaphore.Weighted
	cellID      string
	region      string
	platform    string
}

// Config holds the configuration for creating a new [Watcher].
type Config struct {
	Cluster     ctrl.ClusterServiceClient
	Deployments *deployment.Controller
	CellID      string
	Region      string
	Platform    string
}

// New creates a [Watcher] ready to be started with [Watcher.Watch].
func New(cfg Config) *Watcher {
	return &Watcher{
		cluster:     cfg.Cluster,
		deployments: cfg.Deployments,
		sem:         semaphore.NewWeighted(maxConcurrentDispatches),
		cellID:      cfg.CellID,
		region:      cfg.Region,
		platform:    cfg.Platform,
	}
}

func (s *Watcher) clusterKey() *ctrlv1.ClusterKey {
	return &ctrlv1.ClusterKey{CellId: s.cellID, Platform: s.platform, Region: s.region}
}

// Watch runs two independent loops:
//   - A real-time incremental stream for fast delivery of new changes.
//   - A periodic full sync to reconcile any drift.
//
// Both share a semaphore so the k8s API is not overwhelmed.
// Returns nil when the context is cancelled.
func (s *Watcher) Watch(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runPeriodicFullSync(ctx)
	}()
	s.runStream(ctx)
	<-done
	return nil
}

// runStream resumes only from checkpoints whose preceding events were applied.
func (s *Watcher) runStream(ctx context.Context) {
	var resumeToken []byte

	for {
		jitter := reconnectMin + time.Millisecond*time.Duration(rand.Float64()*float64(reconnectMax.Milliseconds()-reconnectMin.Milliseconds()))
		select {
		case <-ctx.Done():
			return
		case <-time.After(jitter):
		}

		stream, err := s.cluster.WatchDeploymentChanges(ctx, &ctrlv1.WatchDeploymentChangesRequest{
			Cluster:     s.clusterKey(),
			ResumeToken: resumeToken,
		})
		if err != nil {
			if shouldResetResumeToken(err) {
				resumeToken = nil
			}
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
			err := s.dispatch(ctx, event)
			s.sem.Release(1)
			if err != nil {
				metrics.DispatchTotal.WithLabelValues("stream", eventResourceType(event), "error").Inc()
				logger.Error("stream: error dispatching event", "deployment_id", event.GetDeployment().GetApply().GetDeploymentId(), "error", err)
				break
			}
			if event.GetEvent() != nil {
				metrics.DispatchTotal.WithLabelValues("stream", eventResourceType(event), "success").Inc()
			}
			if len(event.GetResumeToken()) > 0 {
				resumeToken = event.GetResumeToken()
				metrics.LastSuccessfulCheckpointUnixSeconds.Set(float64(time.Now().Unix()))
			}
		}

		if err := stream.Err(); err != nil && ctx.Err() == nil {
			if shouldResetResumeToken(err) {
				resumeToken = nil
			}
			logger.Error("stream: connection ended", "error", err)
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
		Cluster: s.clusterKey(),
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
				logger.Error("full sync: error dispatching event", "deployment_id", event.GetDeployment().GetApply().GetDeploymentId(), "error", err)
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

func shouldResetResumeToken(err error) bool {
	code := connect.CodeOf(err)
	return code == connect.CodeInvalidArgument || code == connect.CodeOutOfRange
}

// eventResourceType returns a label-safe resource type string for metrics.
func eventResourceType(event *ctrlv1.DeploymentChangeEvent) string {
	switch event.GetEvent().(type) {
	case *ctrlv1.DeploymentChangeEvent_Deployment:
		return "deployment"
	default:
		return "unknown"
	}
}

// dispatch routes an event to the appropriate controller.
func (s *Watcher) dispatch(ctx context.Context, event *ctrlv1.DeploymentChangeEvent) error {
	switch e := event.GetEvent().(type) {
	case *ctrlv1.DeploymentChangeEvent_Deployment:
		if e.Deployment == nil {
			return fmt.Errorf("received deployment change event with nil deployment state")
		}
		switch op := e.Deployment.GetState().(type) {
		case *ctrlv1.DeploymentState_Apply:
			return s.deployments.ApplyDeployment(ctx, op.Apply)
		case *ctrlv1.DeploymentState_Delete:
			return s.deployments.DeleteDeployment(ctx, op.Delete)
		default:
			return fmt.Errorf("unhandled deployment state type %T", op)
		}

	case nil:
		if len(event.GetResumeToken()) > 0 {
			return nil
		}
		return fmt.Errorf("received deployment change event with nil event")

	default:
		return fmt.Errorf("unhandled deployment change event type %T", e)
	}
}
