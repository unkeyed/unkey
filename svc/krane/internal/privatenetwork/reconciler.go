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
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	discoverylisters "k8s.io/client-go/listers/discovery/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	component       = "private-dns"
	fieldManager    = "krane-private-network"
	indexDeployment = "deployment"
	intervalPoll    = 5 * time.Second
	timeoutPass     = 30 * time.Second
)

// Config holds the dependencies of a [Reconciler].
type Config struct {
	Client     kubernetes.Interface
	Dynamic    dynamic.Interface
	Cluster    ctrl.ClusterServiceClient
	ClusterKey *ctrlv1.ClusterKey
	Clock      clock.Clock
}

// Reconciler publishes discovery objects. Only the Krane leader may run it.
type Reconciler struct {
	client     kubernetes.Interface
	dynamic    dynamic.Interface
	cluster    ctrl.ClusterServiceClient
	clusterKey *ctrlv1.ClusterKey
	clock      clock.Clock

	services       corelisters.ServiceLister
	endpointSlices discoverylisters.EndpointSliceLister
	pods           cache.Indexer
	policies       cache.GenericLister

	entries      map[string]entryStatus
	absences     map[string]absence
	snapshotLast *ctrlv1.PrivateNetworkStateChunk
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

	initMetrics()
	return &Reconciler{
		client:         cfg.Client,
		dynamic:        cfg.Dynamic,
		cluster:        cfg.Cluster,
		clusterKey:     cfg.ClusterKey,
		clock:          cfg.Clock,
		services:       nil,
		endpointSlices: nil,
		pods:           nil,
		policies:       nil,
		entries:        nil,
		absences:       nil,
		snapshotLast:   nil,
	}, nil
}

// Run reconciles until ctx is cancelled. Callers must hold Krane leadership
// for the lifetime of ctx.
func (r *Reconciler) Run(ctx context.Context) {
	r.entries, r.absences = nil, nil
	startLeading()

	inputsChanged, slicesChanged := make(chan struct{}, 1), make(chan struct{}, 1)
	stop, err := r.startCaches(ctx, inputsChanged, slicesChanged)
	if err != nil {
		logger.Error("private network caches did not sync", "error", err)
		return
	}
	defer stop()

	var loops sync.WaitGroup
	defer loops.Wait()
	loops.Go(func() { r.runEndpoints(ctx, inputsChanged) })

	for {
		if err := r.reconcile(ctx); err != nil {
			logger.Warn("private network reconciliation failed", "error", err)
		}
		if !r.sleep(ctx, intervalPoll, slicesChanged) {
			return
		}
	}
}

func (r *Reconciler) startCaches(ctx context.Context, inputsChanged, slicesChanged chan<- struct{}) (func(), error) {
	discovery := informers.NewSharedInformerFactoryWithOptions(r.client, 0, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.LabelSelector = discoveryLabels.ToString()
	}))
	workloads := informers.NewSharedInformerFactoryWithOptions(r.client, 0, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.LabelSelector = labels.New().ManagedByKrane().ComponentDeployment().ToString()
	}), informers.WithTransform(stripManagedFields))
	policies := dynamicinformer.NewFilteredDynamicSharedInformerFactory(r.dynamic, 0, metav1.NamespaceAll, func(options *metav1.ListOptions) {
		options.LabelSelector = discoveryLabels.ToString()
	})
	stop := func() {
		discovery.Shutdown()
		workloads.Shutdown()
		policies.Shutdown()
	}

	services := discovery.Core().V1().Services()
	endpointSlices := discovery.Discovery().V1().EndpointSlices()
	pods := workloads.Core().V1().Pods().Informer()
	policy := policies.ForResource(cilium.NetworkPolicyResource)
	if err := pods.AddIndexers(cache.Indexers{indexDeployment: deploymentIndex}); err != nil {
		return nil, fmt.Errorf("index private network Pods: %w", err)
	}
	inputs := notifier(inputsChanged)
	watches := map[cache.SharedIndexInformer]cache.ResourceEventHandler{
		services.Informer():       inputs,
		pods:                      inputs,
		endpointSlices.Informer(): notifier(inputsChanged, slicesChanged),
	}
	for informer, handler := range watches {
		if _, err := informer.AddEventHandler(handler); err != nil {
			return nil, fmt.Errorf("watch private network inputs: %w", err)
		}
	}
	r.services, r.endpointSlices, r.pods, r.policies = services.Lister(), endpointSlices.Lister(), pods.GetIndexer(), policy.Lister()

	discovery.Start(ctx.Done())
	workloads.Start(ctx.Done())
	policies.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), services.Informer().HasSynced, endpointSlices.Informer().HasSynced, pods.HasSynced, policy.Informer().HasSynced) {
		stop()
		return nil, ctx.Err()
	}
	return stop, nil
}

func notifier(wakes ...chan<- struct{}) cache.ResourceEventHandler {
	notify := func() {
		for _, wake := range wakes {
			select {
			case wake <- struct{}{}:
			default:
			}
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(any) { notify() },
		UpdateFunc: func(any, any) { notify() },
		DeleteFunc: func(any) { notify() },
	}
}

func stripManagedFields(object any) (any, error) {
	if pod, ok := object.(*corev1.Pod); ok {
		pod.ManagedFields = nil
	}
	return object, nil
}

func deploymentIndex(object any) ([]string, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("unexpected Pod cache entry %T", object)
	}
	return []string{pod.Namespace + "/" + pod.Labels[labels.LabelKeyDeploymentID]}, nil
}

func (r *Reconciler) sleep(ctx context.Context, d time.Duration, wake <-chan struct{}) bool {
	timer := r.clock.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C():
	case <-wake:
	}
	return ctx.Err() == nil
}

type discoveryPass struct {
	services        []*corev1.Service
	servicesByKey   map[string]*corev1.Service
	connections     map[string][]*corev1.Service
	policies        map[string]*unstructured.Unstructured
	regions         []string
	pruneRegions    bool
	servicesDesired map[string]struct{}
	servicesEnsured map[string]struct{}
}

func (r *Reconciler) reconcile(ctx context.Context) error {
	started, completed := r.clock.Now(), false
	defer func() { r.observePass(loopDiscovery, started, completed) }()
	ctx, cancel := context.WithTimeout(ctx, timeoutPass)
	defer cancel()

	p, err := r.load()
	if err != nil {
		return countError(loopDiscovery, stageList, err)
	}
	snapshot, err := r.snapshot(ctx)
	if err != nil {
		r.absences = nil
		return countError(loopDiscovery, stageSnapshot, err)
	}

	var errs []error
	regions, err := r.topologyRegions(snapshot.GetTopology())
	if err != nil {
		errs = append(errs, countError(loopDiscovery, stageSnapshot, err))
	}
	p.regions = regions
	p.pruneRegions = snapshot.GetTopology() != nil && err == nil && snapshot.GetCertified()

	connectionSpecs, rejected := validateSnapshot(snapshot.GetConnections())
	connectionSpecs = slices.DeleteFunc(connectionSpecs, func(connectionSpec *ctrlv1.PrivateNetworkConnection) bool {
		policy := p.policies[publishedConnectionKey(connectionSpec)]
		return !r.runsHere(connectionSpec) && (policy == nil || !owned(policy.GetLabels()))
	})
	if err := r.cleanupGrants(ctx, p, snapshot, connectionSpecs, len(rejected) == 0); err != nil {
		return errors.Join(append(errs, countError(loopDiscovery, stageCleanup, err))...)
	}
	for _, connections := range p.connections {
		for _, connection := range connections {
			if name := targetServiceName(connection); name != "" {
				p.servicesDesired[connection.Namespace+"/"+name] = struct{}{}
			}
		}
	}

	entries := make(map[string]entryStatus, len(snapshot.GetConnections()))
	var untracked []entryStatus
	for _, rejection := range rejected {
		errs = append(errs, countError(loopDiscovery, stageInvalidEntry, rejection.err))
		entry := p.failedEntry(rejection.connectionSpec, rejection.connectionKeyRetained, stageInvalidEntry, rejection.err)
		if rejection.connectionKeyRetained == "" {
			untracked = append(untracked, entry)
			continue
		}
		entries[rejection.connectionKeyRetained] = entry
	}

	for _, connectionSpec := range connectionSpecs {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(errs, err)...)
		}
		key := publishedConnectionKey(connectionSpec)
		connection, stage, err := r.reconcileConnection(ctx, p, connectionSpec)
		switch {
		case err != nil:
			errs = append(errs, countError(loopDiscovery, stage, err))
			entries[key] = p.failedEntry(connectionSpec, key, stage, err)
		case r.callerRunsHere(connectionSpec):
			entries[key] = publishedEntry(connectionSpec, connection)
		}
	}

	if err := r.cleanupServices(ctx, p.services, p.servicesDesired); err != nil {
		errs = append(errs, countError(loopDiscovery, stageCleanup, err))
	}

	completed = true
	recordEntries(entries, untracked)
	logEntryChanges(r.entries, entries)
	r.entries = entries

	if len(errs) > 0 {
		return fmt.Errorf("reconcile private network connections, failed connections kept their published objects: %w", errors.Join(errs...))
	}
	return nil
}

func (r *Reconciler) load() (*discoveryPass, error) {
	services, err := r.services.List(discoverySelector)
	if err != nil {
		return nil, fmt.Errorf("list private network Services: %w", err)
	}
	policies, err := r.policies.List(discoverySelector)
	if err != nil {
		return nil, fmt.Errorf("list connection policies: %w", err)
	}

	p := &discoveryPass{
		services:        services,
		servicesByKey:   make(map[string]*corev1.Service, len(services)),
		connections:     make(map[string][]*corev1.Service),
		policies:        make(map[string]*unstructured.Unstructured, len(policies)),
		regions:         nil,
		pruneRegions:    false,
		servicesDesired: make(map[string]struct{}),
		servicesEnsured: make(map[string]struct{}),
	}
	for _, service := range services {
		p.servicesByKey[service.Namespace+"/"+service.Name] = service
		if connectionService(service) {
			key := grantKey(service)
			p.connections[key] = append(p.connections[key], service)
		}
	}
	for _, object := range policies {
		policy, ok := object.(*unstructured.Unstructured)
		if !ok {
			return nil, fmt.Errorf("unexpected connection policy type %T", object)
		}
		p.policies[policy.GetNamespace()+"/"+policy.GetName()] = policy
	}
	return p, nil
}

func (r *Reconciler) reconcileConnection(ctx context.Context, p *discoveryPass, connectionSpec *ctrlv1.PrivateNetworkConnection) (*corev1.Service, string, error) {
	name := grantName(connectionSpec.GetConnectionId(), connectionSpec.GetCallerDeploymentId())
	namespace := connectionSpec.GetK8SNamespace()
	key := namespace + "/" + name
	existing := p.connections[key]

	if !r.runsHere(connectionSpec) {
		if err := r.ensurePolicy(ctx, connectionSpec, name, nil, p.policies[key]); err != nil {
			return nil, stagePolicy, err
		}
		if err := r.withdrawStaleConnections(ctx, existing, nil); err != nil {
			return nil, stageConnection, err
		}
		return nil, "", nil
	}

	var target *corev1.Service
	if connectionSpec.GetTargetDeploymentId() != "" {
		serviceKey := namespace + "/" + privatenetwork.ServiceName(connectionSpec.GetTargetDeploymentId(), connectionSpec.GetTargetPort())
		if _, ensured := p.servicesEnsured[serviceKey]; !ensured {
			service, err := r.ensureService(ctx, connectionSpec, p.servicesByKey[serviceKey])
			if err != nil {
				return nil, stageService, err
			}
			p.servicesByKey[serviceKey] = service
			p.servicesEnsured[serviceKey] = struct{}{}
			p.servicesDesired[serviceKey] = struct{}{}
		}
		target = p.servicesByKey[serviceKey]
	}

	aliasName := privatenetwork.ConnectionServiceName(connectionSpec.GetCallerDeploymentId(), connectionSpec.GetConnectionName())
	labelsDesired := connectionLabels(connectionSpec)
	var published []string
	previous := ""
	for _, connection := range existing {
		deployment, err := publishedDeployment(connection, p.servicesByKey)
		if err != nil {
			return nil, stageConnection, err
		}
		if maps.Equal(connection.Labels, labelsDesired) {
			published = append(published, deployment)
		}
		if connection.Name == aliasName {
			previous = deployment
		}
	}

	if !replicaConnection(connectionSpec) {
		if err := r.ensurePolicy(ctx, connectionSpec, name, append(published, connectionSpec.GetTargetDeploymentId()), p.policies[key]); err != nil {
			return nil, stagePolicy, err
		}
	}
	if !r.callerRunsHere(connectionSpec) {
		if err := r.withdrawStaleConnections(ctx, existing, nil); err != nil {
			return nil, stageConnection, err
		}
		return nil, "", nil
	}

	connection, err := r.ensureConnection(ctx, connectionSpec, aliasName, "", target, p.servicesByKey[namespace+"/"+aliasName], previous)
	if err != nil {
		return nil, stageConnection, err
	}
	current := map[string]struct{}{aliasName: {}}
	if connection != nil {
		resolved := p.servicesByKey[namespace+"/"+targetServiceName(connection)]
		for _, region := range p.regions {
			nameRegional := privatenetwork.RegionalConnectionServiceName(connectionSpec.GetCallerDeploymentId(), connectionSpec.GetConnectionName(), region)
			current[nameRegional] = struct{}{}
			if resolved == nil {
				continue
			}
			if _, err := r.ensureConnection(ctx, connectionSpec, nameRegional, region, resolved, p.servicesByKey[namespace+"/"+nameRegional], ""); err != nil {
				return nil, stageConnection, err
			}
		}
		if !p.pruneRegions {
			for _, service := range existing {
				region := service.Annotations[privatenetwork.RegionAnnotation]
				if region != "" && service.Name == privatenetwork.RegionalConnectionServiceName(connectionSpec.GetCallerDeploymentId(), connectionSpec.GetConnectionName(), region) {
					current[service.Name] = struct{}{}
				}
			}
		}
	}
	if err := r.withdrawStaleConnections(ctx, existing, current); err != nil {
		return nil, stageConnection, err
	}
	return connection, "", nil
}

func (r *Reconciler) runsHere(connectionSpec *ctrlv1.PrivateNetworkConnection) bool {
	return r.callerRunsHere(connectionSpec) || slices.Contains(connectionSpec.GetTargetRegions(), r.clusterKey.GetRegion())
}

func (r *Reconciler) callerRunsHere(connectionSpec *ctrlv1.PrivateNetworkConnection) bool {
	return slices.Contains(connectionSpec.GetCallerRegions(), r.clusterKey.GetRegion())
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

func (p *discoveryPass) failedEntry(connectionSpec *ctrlv1.PrivateNetworkConnection, key, stage string, err error) entryStatus {
	entry := entryStatus{
		kind:                entryKind(connectionSpec),
		state:               stateFailed,
		stage:               stage,
		err:                 err,
		connectionSpec:      connectionSpec,
		publishedDeployment: "",
	}
	for _, connection := range p.connections[key] {
		if deployment := connection.Annotations[privatenetwork.TargetDeploymentAnnotation]; deployment != "" {
			entry.publishedDeployment = deployment
			break
		}
	}
	return entry
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
