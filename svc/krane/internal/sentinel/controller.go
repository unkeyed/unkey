package sentinel

import (
	"context"
	"fmt"
	"sync"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/keymutex"
	"github.com/unkeyed/unkey/svc/krane/internal/podstatus"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Controller manages sentinel Deployments and Services in a Kubernetes cluster.
//
// It maintains bidirectional state synchronization with the control plane:
// receiving desired state from the unified WatchDeploymentChanges stream
// (dispatched by the watcher) and reporting actual state via ReportSentinelStatus.
type Controller struct {
	clientSet     kubernetes.Interface
	cluster       ctrl.ClusterServiceClient
	dynamicClient dynamic.Interface
	cb            circuitbreaker.CircuitBreaker[any]
	done          chan struct{}
	stopOnce      sync.Once
	region        string
	platform      string

	// fingerprints tracks the most recently reported state per sentinel
	// Deployment so we can skip redundant reports during resync and the
	// list-sync prime. Entries auto-expire via the cache's TTL, preventing
	// unbounded growth from deleted Deployments.
	fingerprints cache.Cache[string, string]

	// reportLocks serializes reportIfChanged per k8s_name so the fingerprint
	// Get and post-RPC Set can't race with another concurrent event for the
	// same sentinel and both report the same state.
	reportLocks keymutex.KeyMutex

	// lagRecorder records pod watch delivery lag, deduplicated per
	// (pod UID, transition time).
	lagRecorder *podstatus.LagRecorder
}

// Config holds the configuration required to create a new [Controller].
type Config struct {
	Cluster       ctrl.ClusterServiceClient
	Region        string
	Platform      string
	ClientSet     kubernetes.Interface
	DynamicClient dynamic.Interface

	// Fingerprints is a cache for deduplicating sentinel status reports.
	Fingerprints cache.Cache[string, string]

	// ObservedTransitions is a cache keyed by pod UID that records which
	// ContainersReady transitions have already been sampled, so the lag
	// histogram isn't skewed by repeat events for the same transition.
	ObservedTransitions cache.Cache[string, time.Time]
}

// New creates a [Controller] ready to be started with [Controller.Start].
func New(cfg Config) *Controller {
	return &Controller{
		clientSet:     cfg.ClientSet,
		dynamicClient: cfg.DynamicClient,
		cluster:       cfg.Cluster,
		cb:            circuitbreaker.New[any]("sentinel_state_update"),
		done:          make(chan struct{}),
		region:        cfg.Region,
		platform:      cfg.Platform,
		stopOnce:      sync.Once{},
		fingerprints:  cfg.Fingerprints,
		reportLocks:   keymutex.KeyMutex{},
		lagRecorder:   podstatus.NewLagRecorder("sentinel", cfg.ObservedTransitions),
	}
}

// Start launches the background control loops:
//
//   - [Controller.runActualStateReportLoop]: Real-time Kubernetes watch for
//     Deployment changes, reports actual state back to the control plane.
//   - [Controller.runActualStateResyncLoop]: Periodic safety net for health
//     reporting (complements the real-time watch).
//   - [Controller.runDesiredStateResyncLoop]: Periodic reconciliation of desired
//     state from the control plane (complements the streaming channel).
//
// The actual-state and desired-state resync loops are decoupled so that slow
// control plane RPCs cannot delay health reporting.
//
// All loops continue until the context is cancelled or [Controller.Stop] is called.
func (c *Controller) Start(ctx context.Context) error {
	go c.runActualStateResyncLoop(ctx)
	go c.runDesiredStateResyncLoop(ctx)

	if err := c.runActualStateReportLoop(ctx); err != nil {
		return err
	}

	if err := c.runPodWatchLoop(ctx); err != nil {
		return err
	}

	return nil
}

// Stop signals all background goroutines to terminate.
func (c *Controller) Stop() error {
	c.stopOnce.Do(func() {
		if c.done != nil {
			close(c.done)
		}
	})
	return nil
}

func (c *Controller) regionKey() *ctrlv1.RegionKey {
	return &ctrlv1.RegionKey{Platform: c.platform, Name: c.region}
}

// reportSentinelStatus pushes sentinel status to the control plane through
// the circuit breaker. The circuit breaker prevents cascading failures during
// control plane outages by failing fast after repeated errors.
//
// On success, the fingerprint for this report is cached so that
// [Controller.reportIfChanged] can skip redundant reports during resync.
func (c *Controller) reportSentinelStatus(ctx context.Context, status *ctrlv1.ReportSentinelStatusRequest) error {
	status.Region = c.regionKey()
	start := time.Now()
	_, err := c.cb.Do(ctx, func(innerCtx context.Context) (any, error) {
		return c.cluster.ReportSentinelStatus(innerCtx, status)
	})
	elapsed := time.Since(start)
	result := "success"
	if err != nil {
		result = "error"
	}
	metrics.ReportStatusDurationSeconds.WithLabelValues("sentinel", result).Observe(elapsed.Seconds())
	logger.Info("report sentinel status rpc",
		"k8s_name", status.GetK8SName(),
		"available_replicas", status.GetAvailableReplicas(),
		"health", status.GetHealth(),
		"duration_ms", elapsed.Milliseconds(),
		"result", result,
	)
	if err != nil {
		return fmt.Errorf("failed to report sentinel status: %w", err)
	}

	if c.fingerprints != nil && status.GetK8SName() != "" {
		c.fingerprints.Set(ctx, status.GetK8SName(), sentinelFingerprint(status))
	}

	return nil
}

// reportIfChanged reports sentinel status only when it differs from the last
// successful report for the same k8s_name. Returns true if a report was sent.
// Falls through to an unconditional report when fingerprinting is disabled.
//
// Serialized per k8s_name: pod events for the same sentinel are processed
// concurrently, so the fingerprint Get/RPC/Set window would otherwise race
// and let two events both pass the dedupe check.
func (c *Controller) reportIfChanged(ctx context.Context, status *ctrlv1.ReportSentinelStatusRequest) (bool, error) {
	if c.fingerprints == nil || status.GetK8SName() == "" {
		return true, c.reportSentinelStatus(ctx, status)
	}
	unlock := c.reportLocks.Lock(status.GetK8SName())
	defer unlock()
	fp := sentinelFingerprint(status)
	if prev, hit := c.fingerprints.Get(ctx, status.GetK8SName()); hit == cache.Hit && prev == fp {
		metrics.ReportDedupedTotal.WithLabelValues("sentinel").Inc()
		return false, nil
	}
	return true, c.reportSentinelStatus(ctx, status)
}

// sentinelFingerprint builds a deterministic string from the fields that
// drive the ctrl convergence check so we can cheaply detect whether the
// observed state changed between events.
func sentinelFingerprint(status *ctrlv1.ReportSentinelStatusRequest) string {
	return hash.Sha256(fmt.Sprintf("%s|%d|%d|%s",
		status.GetK8SName(),
		status.GetAvailableReplicas(),
		int32(status.GetHealth()),
		status.GetRunningImage(),
	))
}
