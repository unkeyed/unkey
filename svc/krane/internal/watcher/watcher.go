package watcher

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
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

// Watcher applies deployment changes and runs full syncs to repair missed changes.
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

// Watch runs the change stream and periodic full syncs at the same time.
// Both share a limit on calls to the Kubernetes API.
// It returns nil when the context is canceled.
func (s *Watcher) Watch(ctx context.Context) error {
	var loops sync.WaitGroup
	var dispatches sync.WaitGroup
	loops.Go(func() { s.runPeriodicFullSync(ctx, &dispatches) })
	loops.Go(func() { s.runStream(ctx) })

	loops.Wait()
	dispatches.Wait()
	return nil
}

// runStream reconnects using the last token saved after applying changes.
func (s *Watcher) runStream(ctx context.Context) {
	var resumeToken []byte
	failures := 0

	for {
		if !waitReconnect(ctx) {
			return
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
		} else {
			metrics.StreamConnectionsTotal.WithLabelValues("success").Inc()
			var checkpointAccepted bool
			resumeToken, checkpointAccepted = s.consumeStream(ctx, stream, resumeToken)
			if checkpointAccepted {
				failures = 0
			}
		}

		if ctx.Err() != nil {
			return
		}
		failures++
		if failures >= 3 {
			logger.Warn("stream: restarting snapshot after consecutive failures")
			resumeToken = nil
			failures = 0
		}
	}
}

// consumeStream applies events in order and closes the stream before returning
// the last safe token and whether it accepted a checkpoint. If the token cannot
// be used, it returns an empty token so the next watch starts a new copy.
func (s *Watcher) consumeStream(ctx context.Context, stream *connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], resumeToken []byte) ([]byte, bool) {
	checkpointAccepted := false
	for stream.Receive() {
		event := stream.Msg()
		metrics.StreamEventsReceivedTotal.Inc()

		if event.GetEvent() == nil && len(event.GetResumeToken()) > 0 {
			resumeToken = event.GetResumeToken()
			checkpointAccepted = true
			metrics.LastSuccessfulCheckpointUnixSeconds.Set(float64(time.Now().Unix()))
			continue
		}

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
		metrics.DispatchTotal.WithLabelValues("stream", eventResourceType(event), "success").Inc()
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
	return resumeToken, checkpointAccepted
}

func waitReconnect(ctx context.Context) bool {
	jitter := reconnectMin + time.Millisecond*time.Duration(rand.Float64()*float64(reconnectMax.Milliseconds()-reconnectMin.Milliseconds()))
	timer := time.NewTimer(jitter)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// runPeriodicFullSync repairs missed changes on startup and every fullSyncInterval.
func (s *Watcher) runPeriodicFullSync(ctx context.Context, dispatches *sync.WaitGroup) {
	s.doFullSync(ctx, dispatches)

	ticker := time.NewTicker(fullSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doFullSync(ctx, dispatches)
		}
	}
}

func (s *Watcher) doFullSync(ctx context.Context, dispatches *sync.WaitGroup) {
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
		dispatches.Go(func() {
			defer s.sem.Release(1)

			resourceType := eventResourceType(event)
			if err := s.dispatch(ctx, event); err != nil {
				metrics.DispatchTotal.WithLabelValues("full_sync", resourceType, "error").Inc()
				logger.Error("full sync: error dispatching event", "deployment_id", event.GetDeployment().GetApply().GetDeploymentId(), "error", err)
			} else {
				metrics.DispatchTotal.WithLabelValues("full_sync", resourceType, "success").Inc()
			}
		})
	}

	if err := stream.Close(); err != nil && ctx.Err() == nil {
		logger.Error("full sync: error closing connection", "error", err)
	}

	metrics.FullSyncDurationSeconds.Observe(time.Since(start).Seconds())
}

// shouldResetResumeToken reports whether reconnecting needs a new copy of the rows.
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
		return fmt.Errorf("received deployment change event with nil event")

	default:
		return fmt.Errorf("unhandled deployment change event type %T", e)
	}
}
