// Package privatenetwork publishes private network discovery objects for
// undns.
package privatenetwork

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	component          = "private-dns"
	fieldManager       = "krane-private-network"
	leaseName          = "krane-private-network"
	pollInterval       = 5 * time.Second
	ciliumGlobal       = "service.cilium.io/global"
	ciliumGlobalSlices = "service.cilium.io/global-sync-endpoint-slices"
)

// Reconciler publishes discovery objects while it holds the cluster Lease.
type Reconciler struct {
	client     kubernetes.Interface
	dynamic    dynamic.Interface
	cluster    ctrl.ClusterServiceClient
	clusterKey *ctrlv1.ClusterKey
	elector    *leaderelection.LeaderElector
	endpointMu sync.Mutex
	now        func() time.Time
	entries    map[string]entryStatus
}

// Config holds the dependencies of a [Reconciler]. Identity must be unique
// among the Krane instances that share LeaseNamespace.
type Config struct {
	Client         kubernetes.Interface
	Dynamic        dynamic.Interface
	Cluster        ctrl.ClusterServiceClient
	ClusterKey     *ctrlv1.ClusterKey
	Identity       string
	LeaseNamespace string
}

// New validates cfg and prepares leader election. It does not contact the
// cluster.
func New(cfg Config) (*Reconciler, error) {
	err := assert.All(
		assert.NotNil(cfg.Client, "Kubernetes client is required"),
		assert.NotNil(cfg.Dynamic, "dynamic Kubernetes client is required"),
		assert.NotNil(cfg.Cluster, "control plane client is required"),
		assert.NotNil(cfg.ClusterKey, "cluster key is required"),
		assert.NotEmpty(cfg.Identity, "leader identity is required"),
		assert.NotEmpty(cfg.LeaseNamespace, "lease namespace is required"),
	)
	if err != nil {
		return nil, err
	}

	r := &Reconciler{
		client:     cfg.Client,
		dynamic:    cfg.Dynamic,
		cluster:    cfg.Cluster,
		clusterKey: cfg.ClusterKey,
		elector:    nil,
		endpointMu: sync.Mutex{},
		now:        time.Now,
		entries:    nil,
	}
	initMetrics()

	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: leaseName, Namespace: cfg.LeaseNamespace},
		Labels:     nil,
		Client:     cfg.Client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: cfg.Identity, EventRecorder: nil},
	}
	r.elector, err = leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: false,
		Name:            leaseName,
		WatchDog:        nil,
		Coordinated:     false,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Info("private network leadership acquired", "identity", cfg.Identity)
				startLeading()
				r.run(ctx)
			},
			OnStoppedLeading: func() {
				metrics.PrivateNetworkLeader.Set(0)
				logger.Warn("private network leadership lost", "identity", cfg.Identity)
			},
			OnNewLeader: func(string) {},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create private network leader elector: %w", err)
	}

	return r, nil
}

// Run campaigns for the Lease and reconciles while it leads. It returns an
// error when leadership is lost instead of campaigning again.
func (r *Reconciler) Run(ctx context.Context) error {
	r.elector.Run(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return fmt.Errorf("private network leadership lost")
}

func (r *Reconciler) run(ctx context.Context) {
	endpointCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.runEndpoints(endpointCtx)
	}()
	defer func() {
		cancel()
		<-done
	}()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if err := r.reconcile(ctx); err != nil {
			logger.Warn("private network reconciliation failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Reconciler) reconcile(ctx context.Context) (err error) {
	started, completed := time.Now(), false
	defer func() { observePass(loopDiscovery, started, completed) }()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	selector := labels.New().ManagedByKrane()
	selector[labels.LabelKeyComponent] = component
	services, err := r.client.CoreV1().Services("").List(ctx, metav1.ListOptions{LabelSelector: selector.ToString()})
	if err != nil {
		return countError(loopDiscovery, stageList, fmt.Errorf("list private network Services: %w", err))
	}
	connections, err := r.client.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{LabelSelector: selector.ToString()})
	if err != nil {
		return countError(loopDiscovery, stageList, fmt.Errorf("list private network connections: %w", err))
	}

	servicesByKey := make(map[string]*corev1.Service, len(services.Items))
	for i := range services.Items {
		service := &services.Items[i]
		servicesByKey[service.Namespace+"/"+service.Name] = service
	}
	connectionsByKey := make(map[string]*corev1.ConfigMap, len(connections.Items))
	for i := range connections.Items {
		connection := &connections.Items[i]
		connectionsByKey[connection.Namespace+"/"+connection.Name] = connection
	}

	snapshot, err := r.snapshot(ctx)
	if err != nil {
		return countError(loopDiscovery, stageSnapshot, err)
	}
	defer func() {
		topologyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if topologyErr := r.ensureTopology(topologyCtx, snapshot.GetTopology()); topologyErr != nil {
			err = errors.Join(err, countError(loopDiscovery, stageSnapshot, topologyErr))
		}
	}()
	snapshotConnections, rejected := validateSnapshot(snapshot.GetConnections())

	r.endpointMu.Lock()
	defer r.endpointMu.Unlock()
	pods, err := r.localPods(ctx)
	if err != nil {
		return countError(loopDiscovery, stageList, err)
	}
	sourceSlices, err := r.sourceSlices(ctx)
	if err != nil {
		return countError(loopDiscovery, stageList, err)
	}
	namespaces, err := r.namespaces(ctx)
	if err != nil {
		return countError(loopDiscovery, stageList, err)
	}

	desiredServices := make(map[string]struct{}, len(snapshotConnections))
	desiredConnections := make(map[string]struct{}, len(snapshotConnections))
	desiredPolicies := make(map[string]struct{}, len(snapshotConnections))
	ensuredServices := make(map[string]struct{}, len(snapshotConnections))
	entries := make(map[string]entryStatus, len(snapshot.GetConnections()))
	var untracked []entryStatus
	retain := func(connectionKey string) {
		desiredConnections[connectionKey] = struct{}{}
		desiredPolicies[connectionKey] = struct{}{}
		if existing := connectionsByKey[connectionKey]; existing != nil && existing.Data["serviceName"] != "" {
			desiredServices[existing.Namespace+"/"+existing.Data["serviceName"]] = struct{}{}
		}
	}

	connectionErrs := make([]error, 0, len(rejected))
	fail := func(connectionSpec *ctrlv1.PrivateNetworkConnection, connectionKey, stage string, err error) {
		connectionErrs = append(connectionErrs, countError(loopDiscovery, stage, err))
		entry := entryStatus{kind: entryKind(connectionSpec), state: stateFailed, stage: stage, err: err, connectionSpec: connectionSpec, publishedDeployment: ""}
		if connectionKey == "" {
			untracked = append(untracked, entry)
			return
		}
		if existing := connectionsByKey[connectionKey]; existing != nil {
			entry.publishedDeployment = existing.Data["deploymentId"]
		}
		entries[connectionKey] = entry
		retain(connectionKey)
	}

	for _, rejection := range rejected {
		fail(rejection.connectionSpec, rejection.retainedConnectionKey, stageInvalidEntry, rejection.err)
	}

	for _, connectionSpec := range snapshotConnections {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(connectionErrs, err)...)
		}

		connectionName := connectionResourceName(connectionSpec)
		connectionKey := connectionSpec.GetK8SNamespace() + "/" + connectionName

		if _, exists := namespaces[connectionSpec.GetK8SNamespace()]; !exists {
			_, err := r.client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: connectionSpec.GetK8SNamespace()},
			}, metav1.CreateOptions{})
			if err != nil && !apierrors.IsAlreadyExists(err) {
				fail(connectionSpec, connectionKey, stageNamespace, fmt.Errorf("ensure private network namespace %s: %w", connectionSpec.GetK8SNamespace(), err))
				continue
			}
			namespaces[connectionSpec.GetK8SNamespace()] = struct{}{}
		}

		var service *corev1.Service
		if connectionSpec.GetTargetDeploymentId() != "" {
			name := discoveryName(connectionSpec.GetTargetDeploymentId(), connectionSpec.GetTargetPort())
			serviceKey := connectionSpec.GetK8SNamespace() + "/" + name
			if _, ensured := ensuredServices[serviceKey]; !ensured {
				ensuredService, err := r.ensureService(ctx, connectionSpec, name, servicesByKey[serviceKey])
				if err != nil {
					fail(connectionSpec, connectionKey, stageService, err)
					continue
				}
				servicesByKey[serviceKey] = ensuredService
				if err := r.ensureEndpoints(ctx, ensuredService, pods, sourceSlices[serviceKey]); err != nil {
					desiredServices[serviceKey] = struct{}{}
					fail(connectionSpec, connectionKey, stageEndpointSlice, err)
					continue
				}
				ensuredServices[serviceKey] = struct{}{}
				desiredServices[serviceKey] = struct{}{}
			}
			service = servicesByKey[serviceKey]
		}

		if err := r.ensurePolicy(ctx, connectionSpec, connectionName, connectionsByKey[connectionKey]); err != nil {
			fail(connectionSpec, connectionKey, stagePolicy, err)
			continue
		}
		connection, err := r.ensureConnection(ctx, connectionSpec, connectionName, service, connectionsByKey[connectionKey])
		if err != nil {
			fail(connectionSpec, connectionKey, stageConnection, err)
			continue
		}

		if connection.Data["serviceName"] != "" {
			desiredServices[connection.Namespace+"/"+connection.Data["serviceName"]] = struct{}{}
		}
		desiredConnections[connectionKey] = struct{}{}
		desiredPolicies[connectionKey] = struct{}{}
		entries[connectionKey] = publishedEntry(connectionSpec, connection)
	}

	if err := r.cleanupPolicies(ctx, desiredPolicies); err != nil {
		connectionErrs = append(connectionErrs, countError(loopDiscovery, stageCleanup, err))
	} else if err := r.cleanup(ctx, services, connections, desiredServices, desiredConnections); err != nil {
		connectionErrs = append(connectionErrs, countError(loopDiscovery, stageCleanup, err))
	}

	completed = true
	recordEntries(entries, untracked)
	logEntryChanges(r.entries, entries)
	r.entries = entries

	if len(connectionErrs) > 0 {
		return fmt.Errorf("reconcile private network connections, failed connections kept their published objects: %w", errors.Join(connectionErrs...))
	}
	return nil
}

func publishedEntry(connectionSpec *ctrlv1.PrivateNetworkConnection, connection *corev1.ConfigMap) entryStatus {
	entry := entryStatus{
		kind: entryKind(connectionSpec), state: stateCurrent, stage: "", err: nil, connectionSpec: connectionSpec,
		publishedDeployment: connection.Data["deploymentId"],
	}
	switch {
	case connectionSpec.GetTargetDeploymentId() == "":
		entry.state = stateUnresolved
	case entry.publishedDeployment != connectionSpec.GetTargetDeploymentId():
		entry.state = stateWaitingForEndpoints
	}
	return entry
}
