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
	"github.com/unkeyed/unkey/svc/krane/internal/keymutex"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	"golang.org/x/sync/semaphore"
)

const (
	fullSyncInterval        = 10 * time.Minute
	reconnectMin            = time.Second
	reconnectMax            = 5 * time.Second
	maxConcurrentDispatches = 10

	// dispatchTimeout bounds one apply or delete. Without it, a hung apiserver
	// or Vault call would stop the stream for the whole region.
	dispatchTimeout = 2 * time.Minute
)

// Watcher applies deployment changes and runs full syncs to repair missed changes.
type Watcher struct {
	cluster     ctrl.ClusterServiceClient
	deployments *deployment.Controller
	sem         *semaphore.Weighted
	locks       keymutex.KeyMutex
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
		locks:       keymutex.KeyMutex{},
		cellID:      cfg.CellID,
		region:      cfg.Region,
		platform:    cfg.Platform,
	}
}

func (s *Watcher) clusterKey() *ctrlv1.ClusterKey {
	return &ctrlv1.ClusterKey{CellId: s.cellID, Platform: s.platform, Region: s.region}
}

// Watch runs the change stream, retries of failed stream events, and periodic
// full syncs at the same time. All share a limit on calls to the Kubernetes API.
// It returns nil when the context is canceled.
func (s *Watcher) Watch(ctx context.Context) error {
	retries := newRetryQueue()
	defer metrics.StreamRetriesPending.Set(0)

	var loops sync.WaitGroup
	var dispatches sync.WaitGroup
	loops.Go(func() { s.runPeriodicFullSync(ctx, &dispatches) })
	loops.Go(func() { s.runStream(ctx, retries) })
	loops.Go(func() { s.runRetries(ctx, retries) })

	loops.Wait()
	dispatches.Wait()
	return nil
}

// runStream reconnects using the last token saved after applying changes.
func (s *Watcher) runStream(ctx context.Context, retries *retryQueue) {
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
			resumeToken, checkpointAccepted = s.consumeStream(ctx, stream, resumeToken, retries)
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
// the last safe token and whether it accepted a checkpoint. An event that fails
// moves to retries, so a checkpoint is safe once every earlier event applied or
// is pending there. If the token cannot be used, it returns an empty token so
// the next watch starts a new copy.
func (s *Watcher) consumeStream(ctx context.Context, stream *connect.ServerStreamForClient[ctrlv1.DeploymentChangeEvent], resumeToken []byte, retries *retryQueue) ([]byte, bool) {
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
		key := eventKey(event)
		unlock := s.locks.Lock(key)
		err := s.dispatch(ctx, event)
		if err != nil {
			retries.add(key, event, time.Now())
		} else {
			retries.remove(key)
		}
		unlock()
		s.sem.Release(1)
		if ctx.Err() != nil {
			break
		}
		recordDispatch("stream", event, err)
	}

	if err := stream.Err(); err != nil && ctx.Err() == nil {
		if shouldResetResumeToken(err) {
			resumeToken = nil
		}
		metrics.StreamErrorsTotal.Inc()
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
		metrics.FullSyncErrorsTotal.WithLabelValues("open").Inc()
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

			recordDispatch("full_sync", event, s.dispatch(ctx, event))
		})
	}

	if err := stream.Err(); err != nil && ctx.Err() == nil {
		metrics.FullSyncErrorsTotal.WithLabelValues("receive").Inc()
		logger.Error("full sync: connection ended", "error", err)
	}
	if err := stream.Close(); err != nil && ctx.Err() == nil {
		logger.Error("full sync: error closing connection", "error", err)
	}

	metrics.FullSyncDurationSeconds.Observe(time.Since(start).Seconds())
}

// runRetries applies failed stream events again once their backoff expires.
func (s *Watcher) runRetries(ctx context.Context, retries *retryQueue) {
	ticker := time.NewTicker(retryDelayMin)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.retryDue(ctx, retries)
		}
	}
}

func (s *Watcher) retryDue(ctx context.Context, retries *retryQueue) {
	var attempts sync.WaitGroup
	defer attempts.Wait()
	for key, event := range retries.due(time.Now()) {
		if err := s.sem.Acquire(ctx, 1); err != nil {
			return
		}
		attempts.Go(func() {
			defer s.sem.Release(1)
			unlock := s.locks.Lock(key)
			defer unlock()
			if !retries.holds(key, event) {
				return
			}
			latest, err := s.latestEvent(ctx, event)
			if err == nil && latest == nil {
				retries.remove(key)
				return
			}
			if err == nil {
				err = s.dispatch(ctx, latest)
			}
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				retries.add(key, event, time.Now())
			} else {
				retries.remove(key)
			}
			recordDispatch("retry", event, err)
		})
	}
}

// latestEvent returns the current desired state for a pending apply, or nil
// when the deployment is gone. A deleted row produces no stream event, so
// replaying the old apply could recreate a deployment that was removed.
func (s *Watcher) latestEvent(ctx context.Context, event *ctrlv1.DeploymentChangeEvent) (*ctrlv1.DeploymentChangeEvent, error) {
	apply := event.GetDeployment().GetApply()
	if apply == nil {
		return event, nil
	}
	state, err := s.cluster.GetDesiredDeploymentState(ctx, &ctrlv1.GetDesiredDeploymentStateRequest{
		Cluster:      s.clusterKey(),
		DeploymentId: apply.GetDeploymentId(),
	})
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get desired deployment state: %w", err)
	}
	return &ctrlv1.DeploymentChangeEvent{Event: &ctrlv1.DeploymentChangeEvent_Deployment{Deployment: state}}, nil
}

func recordDispatch(source string, event *ctrlv1.DeploymentChangeEvent, err error) {
	if err != nil {
		metrics.DispatchTotal.WithLabelValues(source, eventResourceType(event), "error").Inc()
		logger.Error(source+": error dispatching event", "workload", eventKey(event), "deployment_id", event.GetDeployment().GetApply().GetDeploymentId(), "error", err)
		return
	}
	metrics.DispatchTotal.WithLabelValues(source, eventResourceType(event), "success").Inc()
}

// eventKey names the workload an event changes. Apply and delete events for
// the same workload share it.
func eventKey(event *ctrlv1.DeploymentChangeEvent) string {
	state := event.GetDeployment()
	if apply := state.GetApply(); apply != nil {
		return apply.GetK8SNamespace() + "/" + apply.GetK8SName()
	}
	return state.GetDelete().GetK8SNamespace() + "/" + state.GetDelete().GetK8SName()
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
	ctx, cancel := context.WithTimeout(ctx, dispatchTimeout)
	defer cancel()
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
