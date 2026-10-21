package privatenetwork

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	component          = privatenetwork.DiscoveryComponent
	fieldManager       = "krane-private-network"
	pollInterval       = 5 * time.Second
	ciliumGlobal       = privatenetwork.CiliumGlobalAnnotation
	ciliumGlobalSlices = privatenetwork.CiliumGlobalSlicesAnnotation
)

// Reconciler publishes discovery objects. Only the Krane leader may run it.
type Reconciler struct {
	client       kubernetes.Interface
	dynamic      dynamic.Interface
	cluster      ctrl.ClusterServiceClient
	clusterKey   *ctrlv1.ClusterKey
	clock        clock.Clock
	endpointMu   sync.Mutex
	entries      map[string]entryStatus
	lastSnapshot *ctrlv1.PrivateNetworkStateChunk
	absences     map[string]absence
}

// New validates cfg. It does not contact the cluster.
func New(cfg Config) (*Reconciler, error) {
	err := assert.All(
		assert.NotNil(cfg.Client, "Kubernetes client is required"),
		assert.NotNil(cfg.Dynamic, "dynamic Kubernetes client is required"),
		assert.NotNil(cfg.Cluster, "control plane client is required"),
		assert.NotNil(cfg.ClusterKey, "cluster key is required"),
		assert.NotNil(cfg.Clock, "clock is required"),
		assert.True(validRegion(cfg.ClusterKey.GetRegion()), "cluster region must be a DNS label of at most 22 characters"),
	)
	if err != nil {
		return nil, err
	}

	r := &Reconciler{
		client:       cfg.Client,
		dynamic:      cfg.Dynamic,
		cluster:      cfg.Cluster,
		clusterKey:   cfg.ClusterKey,
		clock:        cfg.Clock,
		endpointMu:   sync.Mutex{},
		entries:      nil,
		lastSnapshot: nil,
		absences:     nil,
	}
	initMetrics()

	return r, nil
}

// Run reconciles until ctx is cancelled. Callers must hold Krane leadership
// for the lifetime of ctx.
func (r *Reconciler) Run(ctx context.Context) {
	startLeading()
	defer metrics.PrivateNetworkLeader.Set(0)

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

	selector := labels.New().ManagedByKrane().Component(component)
	serviceList, err := r.client.CoreV1().Services("").List(ctx, metav1.ListOptions{LabelSelector: selector.ToString()})
	if err != nil {
		return countError(loopDiscovery, stageList, fmt.Errorf("list private network Services: %w", err))
	}
	policies, err := r.listPolicies(ctx)
	if err != nil {
		return countError(loopDiscovery, stageList, err)
	}

	services := make([]*corev1.Service, 0, len(serviceList.Items))
	servicesByKey := make(map[string]*corev1.Service, len(serviceList.Items))
	connectionsByKey := make(map[string][]*corev1.Service)
	for i := range serviceList.Items {
		service := &serviceList.Items[i]
		services = append(services, service)
		servicesByKey[service.Namespace+"/"+service.Name] = service
		if connectionService(service) {
			connectionsByKey[grantKey(service)] = append(connectionsByKey[grantKey(service)], service)
		}
	}

	snapshot, err := r.snapshot(ctx)
	if err != nil {
		r.absences = nil
		return countError(loopDiscovery, stageSnapshot, err)
	}
	regions, topologyErr := r.topologyRegions(snapshot.GetTopology())
	defer func() {
		if topologyErr != nil {
			err = errors.Join(err, countError(loopDiscovery, stageSnapshot, topologyErr))
		}
	}()
	pruneRegions := snapshot.GetTopology() != nil && topologyErr == nil && snapshot.GetCertified()
	snapshotConnections, rejected := validateSnapshot(snapshot.GetConnections())
	wanted := make(map[string]struct{}, len(snapshotConnections))
	for _, spec := range snapshotConnections {
		wanted[publishedConnectionKey(spec)] = struct{}{}
	}
	if len(rejected) > 0 {
		r.absences = nil
	}
	if !snapshot.GetCertified() {
		maps.DeleteFunc(r.absences, func(key string, _ absence) bool {
			_, present := wanted[key]
			return present
		})
	} else if err := r.cleanupGrants(ctx, snapshot.GetSnapshotId(), wanted, connectionsByKey, policies, len(rejected) == 0); err != nil {
		return countError(loopDiscovery, stageCleanup, err)
	}
	maps.DeleteFunc(servicesByKey, func(_ string, service *corev1.Service) bool {
		return connectionService(service) && !slices.Contains(connectionsByKey[grantKey(service)], service)
	})

	endpointSliceList, err := r.client.DiscoveryV1().EndpointSlices("").List(ctx, metav1.ListOptions{LabelSelector: selector.ToString()})
	if err != nil {
		return countError(loopDiscovery, stageList, fmt.Errorf("list private network EndpointSlices: %w", err))
	}
	slicesByService := make(map[string][]discoveryv1.EndpointSlice)
	for _, slice := range endpointSliceList.Items {
		key := slice.Namespace + "/" + slice.Labels[discoveryv1.LabelServiceName]
		slicesByService[key] = append(slicesByService[key], slice)
	}

	namespaces, err := r.namespaces(ctx)
	if err != nil {
		return countError(loopDiscovery, stageList, err)
	}

	desiredServices := make(map[string]struct{}, len(snapshotConnections))
	ensuredServices := make(map[string]struct{}, len(snapshotConnections))
	var pods []corev1.Pod
	var sourceSlices map[string][]discoveryv1.EndpointSlice
	var endpointInputsErr error
	endpointInputsLoaded := false
	entries := make(map[string]entryStatus, len(snapshot.GetConnections()))
	var untracked []entryStatus
	retain := func(connectionKey string) {
		for _, connection := range connectionsByKey[connectionKey] {
			if name := targetServiceName(connection); name != "" {
				desiredServices[connection.Namespace+"/"+name] = struct{}{}
			}
		}
	}

	connectionErrs := make([]error, 0, len(rejected))
	fail := func(connectionSpec *ctrlv1.PrivateNetworkConnection, connectionKey, stage string, err error) {
		connectionErrs = append(connectionErrs, countError(loopDiscovery, stage, err))
		entry := entryStatus{
			kind:                entryKind(connectionSpec),
			state:               stateFailed,
			stage:               stage,
			err:                 err,
			connectionSpec:      connectionSpec,
			publishedDeployment: "",
		}
		if connectionKey == "" {
			untracked = append(untracked, entry)
			return
		}
		for _, connection := range connectionsByKey[connectionKey] {
			if entry.publishedDeployment == "" {
				entry.publishedDeployment = connection.Annotations[privatenetwork.TargetDeploymentAnnotation]
			}
		}
		entries[connectionKey] = entry
		retain(connectionKey)
	}

	for _, rejection := range rejected {
		fail(rejection.connectionSpec, rejection.retainedConnectionKey, stageInvalidEntry, rejection.err)
	}
	for key := range connectionsByKey {
		retain(key)
	}

entries:
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
				needsEndpointEnsure := servicesByKey[serviceKey] == nil
				if !needsEndpointEnsure {
					needsEndpointEnsure = !slices.ContainsFunc(slicesByService[serviceKey], func(slice discoveryv1.EndpointSlice) bool {
						return slice.Labels[discoveryv1.LabelManagedBy] == sourceSliceManager
					})
				}
				ensuredService, err := r.ensureService(ctx, connectionSpec, name, servicesByKey[serviceKey])
				if err != nil {
					fail(connectionSpec, connectionKey, stageService, err)
					continue
				}
				servicesByKey[serviceKey] = ensuredService
				if needsEndpointEnsure {
					r.endpointMu.Lock()
					if !endpointInputsLoaded {
						pods, endpointInputsErr = r.localPods(ctx)
						if endpointInputsErr == nil {
							sourceSlices, endpointInputsErr = r.sourceSlices(ctx)
						}
						endpointInputsLoaded = true
					}
					err = endpointInputsErr
					if err == nil {
						err = r.ensureEndpoints(ctx, ensuredService, pods, sourceSlices[serviceKey])
					}
					r.endpointMu.Unlock()
				}
				if err != nil {
					fail(connectionSpec, connectionKey, stageEndpointSlice, err)
					continue
				}
				ensuredServices[serviceKey] = struct{}{}
				desiredServices[serviceKey] = struct{}{}
			}
			service = servicesByKey[serviceKey]
		}

		aliasName := privatenetwork.ConnectionServiceName(connectionSpec.GetCallerDeploymentId(), connectionSpec.GetConnectionName())
		var published []string
		previous := ""
		for _, existing := range connectionsByKey[connectionKey] {
			deployment, err := publishedDeployment(existing, servicesByKey)
			if err != nil {
				fail(connectionSpec, connectionKey, stageConnection, err)
				continue entries
			}
			if maps.Equal(existing.Labels, connectionLabels(connectionSpec)) {
				published = append(published, deployment)
			}
			if existing.Name == aliasName {
				previous = deployment
			}
		}

		if !replicaConnection(connectionSpec) {
			if err := r.ensurePolicy(ctx, connectionSpec, connectionName, published, policies[connectionKey]); err != nil {
				fail(connectionSpec, connectionKey, stagePolicy, err)
				continue
			}
		}
		var endpoints []discoveryv1.EndpointSlice
		if service != nil {
			endpoints = slicesByService[service.Namespace+"/"+service.Name]
		}
		connection, err := r.ensureConnection(ctx, connectionSpec, aliasName, "", service, servicesByKey[connectionSpec.GetK8SNamespace()+"/"+aliasName], previous, endpoints)
		if err != nil {
			fail(connectionSpec, connectionKey, stageConnection, err)
			continue
		}
		current := map[string]struct{}{aliasName: {}}
		if connection != nil {
			resolved := servicesByKey[connection.Namespace+"/"+targetServiceName(connection)]
			for _, region := range regions {
				name := privatenetwork.RegionalConnectionServiceName(connectionSpec.GetCallerDeploymentId(), connectionSpec.GetConnectionName(), region)
				current[name] = struct{}{}
				if resolved == nil {
					continue
				}
				if _, err := r.ensureConnection(ctx, connectionSpec, name, region, resolved, servicesByKey[connection.Namespace+"/"+name], "", nil); err != nil {
					fail(connectionSpec, connectionKey, stageConnection, err)
					continue entries
				}
			}
			for _, existing := range connectionsByKey[connectionKey] {
				region := existing.Annotations[privatenetwork.RegionAnnotation]
				if !pruneRegions && region != "" && existing.Name == privatenetwork.RegionalConnectionServiceName(connectionSpec.GetCallerDeploymentId(), connectionSpec.GetConnectionName(), region) {
					current[existing.Name] = struct{}{}
				}
			}
		}
		if err := r.withdrawStaleConnections(ctx, connectionsByKey[connectionKey], current); err != nil {
			fail(connectionSpec, connectionKey, stageConnection, err)
			continue
		}
		entries[connectionKey] = publishedEntry(connectionSpec, connection)
	}

	if err := r.cleanupServices(ctx, services, desiredServices); err != nil {
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

func (r *Reconciler) withdrawStaleConnections(ctx context.Context, connections []*corev1.Service, current map[string]struct{}) error {
	for _, connection := range connections {
		if _, keep := current[connection.Name]; keep {
			continue
		}
		if err := r.client.CoreV1().Services(connection.Namespace).Delete(ctx, connection.Name, precondition.Unchanged(connection)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("withdraw stale private network connection %s/%s: %w", connection.Namespace, connection.Name, err)
		}
	}
	return nil
}

func publishedEntry(connectionSpec *ctrlv1.PrivateNetworkConnection, connection *corev1.Service) entryStatus {
	entry := entryStatus{
		kind:                entryKind(connectionSpec),
		state:               stateCurrent,
		stage:               "",
		err:                 nil,
		connectionSpec:      connectionSpec,
		publishedDeployment: "",
	}
	if connection != nil {
		entry.publishedDeployment = connection.Annotations[privatenetwork.TargetDeploymentAnnotation]
	}
	switch {
	case connectionSpec.GetTargetDeploymentId() == "":
		entry.state = stateUnresolved
	case entry.publishedDeployment != connectionSpec.GetTargetDeploymentId():
		entry.state = stateWaitingForEndpoints
	}
	return entry
}

func replicaConnection(connectionSpec *ctrlv1.PrivateNetworkConnection) bool {
	return connectionSpec.GetCallerDeploymentId() == connectionSpec.GetTargetDeploymentId()
}
