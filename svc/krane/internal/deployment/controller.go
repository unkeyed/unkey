package deployment

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/hash"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Controller manages deployment ReplicaSets in a Kubernetes cluster by maintaining
// bidirectional state synchronization with the control plane.
//
// The controller receives desired state via the WatchDeployments stream and reports
// actual state via ReportDeploymentStatus. It operates independently from the sentinel
// controller with its own version cursor and circuit breaker, ensuring that failures
// in one controller don't cascade to the other.
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
	versionLastSeen  uint64

	// fingerprintMu guards fingerprints.
	fingerprintMu sync.Mutex
	// fingerprints tracks the most recently reported state per ReplicaSet
	// so we can skip redundant reports during resync.
	fingerprints map[string]string
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

	// Vault provides secrets decryption. Nil disables deploy-time secret decryption.
	Vault vault.VaultServiceClient

	// Registry holds container registry credentials for creating imagePullSecrets.
	// Nil disables pull secret creation.
	Registry *RegistryConfig
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
		versionLastSeen:  0,
		fingerprintMu:    sync.Mutex{},
		fingerprints:     make(map[string]string),
	}
}

// Start launches the three background control loops and blocks until they're initialized.
//
// The method starts [Controller.runResyncLoop] and [Controller.runDesiredStateApplyLoop]
// as background goroutines, and initializes [Controller.runPodWatchLoop]'s Kubernetes
// watch before returning. If watch initialization fails, Start returns the error and
// no goroutines are left running.
//
// All loops continue until the context is cancelled or [Controller.Stop] is called.
func (c *Controller) Start(ctx context.Context) error {
	go c.runResyncLoop(ctx)

	if err := c.runPodWatchLoop(ctx); err != nil {
		return err
	}

	go c.runDesiredStateApplyLoop(ctx)

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
	_, err := c.cb.Do(ctx, func(innerCtx context.Context) (any, error) {
		return c.cluster.ReportDeploymentStatus(innerCtx, status)
	})
	if err != nil {
		return fmt.Errorf("failed to report deployment status: %w", err)
	}

	if update := status.GetUpdate(); update != nil {
		c.fingerprintMu.Lock()
		c.fingerprints[update.GetK8SName()] = instanceFingerprint(update.GetInstances())
		c.fingerprintMu.Unlock()
	}

	return nil
}

// reportIfChanged reports deployment status only when the instance list differs
// from the last successful report. Returns true if a report was sent.
func (c *Controller) reportIfChanged(ctx context.Context, status *ctrlv1.ReportDeploymentStatusRequest) (bool, error) {
	update := status.GetUpdate()
	if update == nil {
		// Deletes are always forwarded.
		return true, c.reportDeploymentStatus(ctx, status)
	}

	fp := instanceFingerprint(update.GetInstances())
	c.fingerprintMu.Lock()
	prev := c.fingerprints[update.GetK8SName()]
	c.fingerprintMu.Unlock()
	if prev == fp {
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
