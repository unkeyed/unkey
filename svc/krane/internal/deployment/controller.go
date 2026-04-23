package deployment

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/keymutex"
	"github.com/unkeyed/unkey/svc/krane/internal/podstatus"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Controller manages deployment ReplicaSets in a Kubernetes cluster by maintaining
// bidirectional state synchronization with the control plane.
//
// The controller receives desired state from the unified WatchDeploymentChanges stream
// (dispatched by the watcher) and reports actual state via ReportDeploymentStatus.
//
// Create a Controller with [New] and start it with [Controller.Start]. The controller
// runs until the context is cancelled or [Controller.Stop] is called.
type Controller struct {
	clientSet        kubernetes.Interface
	dynamicClient    dynamic.Interface
	cluster          ctrl.ClusterServiceClient
	vault            vault.VaultServiceClient
	registry         *RegistryConfig
	imagePullSecrets []corev1.LocalObjectReference
	cb               circuitbreaker.CircuitBreaker[any]
	done             chan struct{}
	region           string
	platform         string

	// fingerprints tracks the most recently reported state per ReplicaSet
	// so we can skip redundant reports during resync. Entries auto-expire
	// via the cache's TTL, preventing unbounded growth from deleted RSs.
	fingerprints cache.Cache[string, string]

	// reportLocks serializes reportIfChanged per k8s_name so the fingerprint
	// Get and post-RPC Set can't race with another concurrent event for the
	// same ReplicaSet and both report the same state.
	reportLocks keymutex.KeyMutex

	// lagRecorder records pod watch delivery lag, deduplicated per
	// (pod UID, transition time).
	lagRecorder *podstatus.LagRecorder

	// storageClassName is the Kubernetes StorageClass for ephemeral volumes.
	storageClassName string
}

// Config holds the configuration required to create a new [Controller].
//
// All fields are required. The ClientSet and DynamicClient are used for Kubernetes
// operations, while Cluster provides the control plane RPC client for state
// synchronization. Region determines which deployments this controller manages.
type Config struct {
	// ClientSet provides typed Kubernetes API access for ReplicaSet and Pod operations.
	ClientSet kubernetes.Interface

	// DynamicClient provides unstructured Kubernetes API access for CiliumNetworkPolicy
	// resources that don't have generated Go types.
	DynamicClient dynamic.Interface

	// Cluster is the control plane RPC client for WatchDeployments and
	// ReportDeploymentStatus calls.
	Cluster ctrl.ClusterServiceClient

	// Region identifies the cluster region for filtering deployment streams.
	Region string

	// Platform identifies the infrastructure provider (e.g. "aws", "gcp", "local").
	Platform string

	// Vault provides secrets decryption. Nil disables deploy-time secret decryption.
	Vault vault.VaultServiceClient

	// Registry holds container registry credentials for creating imagePullSecrets.
	// Nil disables pull secret creation.
	Registry *RegistryConfig

	// Fingerprints is a cache for deduplicating deployment status reports.
	Fingerprints cache.Cache[string, string]

	// ObservedTransitions is a cache keyed by pod UID that records which
	// ContainersReady transitions have already been sampled, so the lag
	// histogram isn't skewed by repeat events for the same transition.
	ObservedTransitions cache.Cache[string, time.Time]

	// StorageClassName is the Kubernetes StorageClass for ephemeral volumes.
	StorageClassName string
}

// New creates a [Controller] ready to be started with [Controller.Start].
//
// The controller initializes with versionLastSeen=0, meaning it will receive all
// pending deployments on first connection. The circuit breaker starts in a closed
// (healthy) state.
func New(cfg Config) *Controller {
	var pullSecrets []corev1.LocalObjectReference
	if cfg.Registry != nil {
		pullSecrets = []corev1.LocalObjectReference{{Name: registryPullSecretName}}
	}

	return &Controller{
		clientSet:        cfg.ClientSet,
		dynamicClient:    cfg.DynamicClient,
		cluster:          cfg.Cluster,
		vault:            cfg.Vault,
		registry:         cfg.Registry,
		imagePullSecrets: pullSecrets,
		cb:               circuitbreaker.New[any]("deployment_state_update"),
		done:             make(chan struct{}),
		region:           cfg.Region,
		platform:         cfg.Platform,
		fingerprints:     cfg.Fingerprints,
		reportLocks:      keymutex.KeyMutex{},
		lagRecorder:      podstatus.NewLagRecorder("deployment", cfg.ObservedTransitions),
		storageClassName: cfg.StorageClassName,
	}
}

// Start launches the background control loops.
//
// Four independent loops run concurrently:
//   - [Controller.runActualStateResyncLoop]: periodic safety net for instance
//     state reporting (complements the real-time pod watch).
//   - [Controller.runDesiredStateResyncLoop]: periodic reconciliation of desired
//     state from the control plane (complements the streaming channel).
//   - [Controller.runImagePullMetricsLoop]: publishes image-pull-failure gauges
//     derived from a periodic ListPods sweep.
//   - [Controller.runPodWatchLoop]: real-time Kubernetes watch for pod events.
//
// The actual-state and desired-state loops are decoupled so that slow control
// plane RPCs cannot delay instance reporting.
//
// If watch initialization fails, Start returns the error.
// All loops continue until the context is cancelled or [Controller.Stop] is called.
func (c *Controller) Start(ctx context.Context) error {
	go c.runActualStateResyncLoop(ctx)
	go c.runImagePullMetricsLoop(ctx)
	go c.runDesiredStateResyncLoop(ctx)

	if err := c.runPodWatchLoop(ctx); err != nil {
		return err
	}

	return nil
}

// Stop signals all background goroutines to terminate by closing the done channel.
// Returns nil; the error return exists for interface compatibility.
func (c *Controller) Stop() error {
	close(c.done)
	return nil
}

// reportDeploymentStatus reports actual deployment state to the control plane
// through the circuit breaker. The circuit breaker prevents cascading failures
// during control plane outages by failing fast after repeated errors.
//
// On success, the fingerprint for this report is cached so that
// [Controller.reportIfChanged] can skip redundant reports during resync.
func (c *Controller) reportDeploymentStatus(ctx context.Context, status *ctrlv1.ReportDeploymentStatusRequest) error {
	start := time.Now()
	_, err := c.cb.Do(ctx, func(innerCtx context.Context) (any, error) {
		return c.cluster.ReportDeploymentStatus(innerCtx, status)
	})
	elapsed := time.Since(start)
	result := "success"
	if err != nil {
		result = "error"
	}
	metrics.ReportStatusDurationSeconds.WithLabelValues("deployment", result).Observe(elapsed.Seconds())
	rsName := ""
	instanceCount := 0
	if update := status.GetUpdate(); update != nil {
		rsName = update.GetK8SName()
		instanceCount = len(update.GetInstances())
	} else if del := status.GetDelete(); del != nil {
		rsName = del.GetK8SName()
	}
	logger.Info("report deployment status rpc",
		"replicaSet", rsName,
		"instances", instanceCount,
		"duration_ms", elapsed.Milliseconds(),
		"result", result,
	)
	if err != nil {
		return fmt.Errorf("failed to report deployment status: %w", err)
	}

	if update := status.GetUpdate(); update != nil {
		c.fingerprints.Set(ctx, update.GetK8SName(), instanceFingerprint(update.GetInstances()))
	}

	return nil
}

// reportIfChanged reports deployment status only when the instance list differs
// from the last successful report. Returns true if a report was sent.
//
// Serialized per k8s_name: pod events for the same ReplicaSet are processed
// concurrently, so the fingerprint Get/RPC/Set window would otherwise race
// and let two events both pass the dedupe check.
func (c *Controller) reportIfChanged(ctx context.Context, status *ctrlv1.ReportDeploymentStatusRequest) (bool, error) {
	update := status.GetUpdate()
	if update == nil {
		// Deletes are always forwarded.
		return true, c.reportDeploymentStatus(ctx, status)
	}

	unlock := c.reportLocks.Lock(update.GetK8SName())
	defer unlock()

	fp := instanceFingerprint(update.GetInstances())
	if prev, hit := c.fingerprints.Get(ctx, update.GetK8SName()); hit == cache.Hit && prev == fp {
		metrics.ReportDedupedTotal.WithLabelValues("deployment").Inc()
		return false, nil
	}

	return true, c.reportDeploymentStatus(ctx, status)
}

// instanceFingerprint builds a deterministic string from the instance list so
// we can cheaply detect whether the actual state changed between resync ticks.
func instanceFingerprint(instances []*ctrlv1.ReportDeploymentStatusRequest_Update_Instance) string {
	parts := make([]string, 0, len(instances))
	for _, inst := range instances {
		parts = append(parts, fmt.Sprintf("%s|%s|%d", inst.GetK8SName(), inst.GetAddress(), inst.GetStatus()))
	}
	sort.Strings(parts)
	return hash.Sha256(strings.Join(parts, ";"))
}
