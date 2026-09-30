package deployment

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
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
	"google.golang.org/protobuf/proto"
	appsv1 "k8s.io/api/apps/v1"
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
// Create a Controller with [New] and run it with [Controller.Run]. The controller
// runs until the context is cancelled.
type Controller struct {
	clientSet        kubernetes.Interface
	dynamicClient    dynamic.Interface
	cluster          ctrl.ClusterServiceClient
	vault            vault.VaultServiceClient
	registry         *RegistryConfig
	imagePullSecrets []corev1.LocalObjectReference
	cb               circuitbreaker.CircuitBreaker[any]
	cellID           string
	region           string
	platform         string

	fingerprints cache.Cache[string, string]

	eventDedup cache.Cache[string, struct{}]

	reportLocks keymutex.KeyMutex

	// lagRecorder records pod watch delivery lag, deduplicated per
	// (pod UID, transition time).
	lagRecorder *podstatus.LagRecorder

	// storageClassName is the Kubernetes StorageClass for ephemeral volumes.
	storageClassName string

	// disableGvisor drops the gVisor sandbox from user workloads.
	disableGvisor bool
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

	// Cluster is the control plane RPC client for watching deployments and
	// ReportDeploymentStatus calls.
	Cluster ctrl.ClusterServiceClient

	// CellID uniquely identifies the infrastructure cell.
	CellID string

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

	// EventDedup is a cache for deduplicating per-container lifecycle
	// events (terminations and actionable waiting/pod errors). Optional: when nil, the
	// instance event capture path is disabled and only the coarse
	// deployment-status report fires.
	EventDedup cache.Cache[string, struct{}]

	// ObservedTransitions is a cache keyed by pod UID that records which
	// ContainersReady transitions have already been sampled, so the lag
	// histogram isn't skewed by repeat events for the same transition.
	ObservedTransitions cache.Cache[string, time.Time]

	// StorageClassName is the Kubernetes StorageClass for ephemeral volumes.
	StorageClassName string

	// DisableGvisor drops the gVisor sandbox from user workloads, leaving them
	// on the node's default runtime.
	DisableGvisor bool
}

// New creates a [Controller] ready to be run with [Controller.Run].
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
		cellID:           cfg.CellID,
		region:           cfg.Region,
		platform:         cfg.Platform,
		fingerprints:     cfg.Fingerprints,
		eventDedup:       cfg.EventDedup,
		reportLocks:      keymutex.KeyMutex{},
		lagRecorder:      podstatus.NewLagRecorder("deployment", cfg.ObservedTransitions),
		storageClassName: cfg.StorageClassName,
		disableGvisor:    cfg.DisableGvisor,
	}
}

// Run runs the background control loops until ctx is cancelled.
//
// Three independent loops run concurrently:
//   - [Controller.runActualStateResyncLoop]: periodic safety net for instance
//     state reporting (complements the real-time pod watch).
//   - [Controller.runDesiredStateResyncLoop]: periodic reconciliation of desired
//     state from the control plane (complements the streaming channel).
//   - [Controller.runPodWatchLoop]: real-time Kubernetes watch for pod events.
//
// The actual-state and desired-state loops are decoupled so that slow control
// plane RPCs cannot delay instance reporting.
//
// Run waits for every loop and its event handlers to stop before returning.
func (c *Controller) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() { c.runActualStateResyncLoop(ctx) })
	wg.Go(func() { c.runDesiredStateResyncLoop(ctx) })
	wg.Go(func() { c.runPodWatchLoop(ctx) })

	wg.Wait()
}

func (c *Controller) clusterKey() *ctrlv1.ClusterKey {
	return &ctrlv1.ClusterKey{CellId: c.cellID, Platform: c.platform, Region: c.region}
}

// reportDeploymentStatus reports actual deployment state to the control plane
// through the circuit breaker. The circuit breaker prevents cascading failures
// during control plane outages by failing fast after repeated errors.
func (c *Controller) reportDeploymentStatus(ctx context.Context, status *ctrlv1.ReportDeploymentStatusRequest) error {
	status.Cluster = c.clusterKey()
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
		fingerprint, err := instanceFingerprint(update.GetInstances())
		if err != nil {
			return err
		}
		c.fingerprints.Set(ctx, update.GetK8SName(), fingerprint)
	}

	return nil
}

func (c *Controller) reportReplicaSet(ctx context.Context, rs *appsv1.ReplicaSet, force bool) (bool, error) {
	unlock := c.reportLocks.Lock(rs.Name)
	defer unlock()

	status, err := c.buildDeploymentStatus(ctx, rs)
	if err != nil {
		return false, err
	}
	update := status.GetUpdate()
	fp, err := instanceFingerprint(update.GetInstances())
	if err != nil {
		return false, err
	}
	prev, hit := c.fingerprints.Get(ctx, update.GetK8SName())
	changed := hit != cache.Hit || prev != fp
	if !changed && !force {
		metrics.ReportDedupedTotal.WithLabelValues("deployment").Inc()
		return false, nil
	}

	return changed, c.reportDeploymentStatus(ctx, status)
}

// instanceFingerprint builds a deterministic string from the instance list so
// we can cheaply detect whether the actual state changed between resync ticks.
func instanceFingerprint(instances []*ctrlv1.ReportDeploymentStatusRequest_Update_Instance) (string, error) {
	parts := make([]string, 0, len(instances))
	for _, inst := range instances {
		snapshot := proto.CloneOf(inst)
		if observation := snapshot.GetContainerObservation(); observation != nil {
			observation.ObservedAtUnixNano = 0
		}
		encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(snapshot)
		if err != nil {
			return "", err
		}
		parts = append(parts, hash.Sha256(string(encoded)))
	}
	sort.Strings(parts)
	return hash.Sha256(strings.Join(parts, ";")), nil
}
