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
	}

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
			OnStartedLeading: r.run,
			OnStoppedLeading: func() { logger.Warn("private network leadership lost") },
			OnNewLeader:      func(string) {},
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

func (r *Reconciler) reconcile(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	selector := labels.New().ManagedByKrane()
	selector[labels.LabelKeyComponent] = component
	services, err := r.client.CoreV1().Services("").List(ctx, metav1.ListOptions{LabelSelector: selector.ToString()})
	if err != nil {
		return fmt.Errorf("list private network Services: %w", err)
	}
	bindings, err := r.client.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{LabelSelector: selector.ToString()})
	if err != nil {
		return fmt.Errorf("list private network bindings: %w", err)
	}

	servicesByKey := make(map[string]*corev1.Service, len(services.Items))
	for i := range services.Items {
		service := &services.Items[i]
		servicesByKey[service.Namespace+"/"+service.Name] = service
	}
	bindingsByKey := make(map[string]*corev1.ConfigMap, len(bindings.Items))
	for i := range bindings.Items {
		binding := &bindings.Items[i]
		bindingsByKey[binding.Namespace+"/"+binding.Name] = binding
	}

	snapshot, err := r.snapshot(ctx)
	if err != nil {
		return err
	}
	apps, rejected := validateSnapshot(snapshot)

	r.endpointMu.Lock()
	defer r.endpointMu.Unlock()
	pods, err := r.localPods(ctx)
	if err != nil {
		return err
	}
	sourceSlices, err := r.sourceSlices(ctx)
	if err != nil {
		return err
	}
	namespaces, err := r.namespaces(ctx)
	if err != nil {
		return err
	}

	desiredServices := make(map[string]struct{}, len(apps))
	desiredBindings := make(map[string]struct{}, len(apps))
	desiredPolicies := make(map[string]struct{}, len(apps))
	ensuredServices := make(map[string]struct{}, len(apps))
	retain := func(bindingKey string) {
		desiredBindings[bindingKey] = struct{}{}
		desiredPolicies[bindingKey] = struct{}{}
		if existing := bindingsByKey[bindingKey]; existing != nil && existing.Data["serviceName"] != "" {
			desiredServices[existing.Namespace+"/"+existing.Data["serviceName"]] = struct{}{}
		}
	}

	appErrs := make([]error, 0, len(rejected))
	for _, rejection := range rejected {
		appErrs = append(appErrs, rejection.err)
		if rejection.retainedBindingKey != "" {
			retain(rejection.retainedBindingKey)
		}
	}

	for _, app := range apps {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(appErrs, err)...)
		}

		bindingName := bindingResourceName(app)
		bindingKey := app.GetK8SNamespace() + "/" + bindingName

		if _, exists := namespaces[app.GetK8SNamespace()]; !exists {
			_, err := r.client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: app.GetK8SNamespace()},
			}, metav1.CreateOptions{})
			if err != nil && !apierrors.IsAlreadyExists(err) {
				appErrs = append(appErrs, fmt.Errorf("ensure private network namespace %s: %w", app.GetK8SNamespace(), err))
				retain(bindingKey)
				continue
			}
			namespaces[app.GetK8SNamespace()] = struct{}{}
		}

		var service *corev1.Service
		if app.GetDeploymentId() != "" {
			name := discoveryName(app.GetDeploymentId(), app.GetPort())
			serviceKey := app.GetK8SNamespace() + "/" + name
			if _, ensured := ensuredServices[serviceKey]; !ensured {
				ensuredService, err := r.ensureService(ctx, app, name, servicesByKey[serviceKey])
				if err != nil {
					appErrs = append(appErrs, err)
					retain(bindingKey)
					continue
				}
				servicesByKey[serviceKey] = ensuredService
				if err := r.ensureEndpoints(ctx, ensuredService, pods, sourceSlices[serviceKey]); err != nil {
					appErrs = append(appErrs, err)
					desiredServices[serviceKey] = struct{}{}
					retain(bindingKey)
					continue
				}
				ensuredServices[serviceKey] = struct{}{}
				desiredServices[serviceKey] = struct{}{}
			}
			service = servicesByKey[serviceKey]
		}

		if err := r.ensurePolicy(ctx, app, bindingName, bindingsByKey[bindingKey]); err != nil {
			appErrs = append(appErrs, err)
			retain(bindingKey)
			continue
		}
		binding, err := r.ensureBinding(ctx, app, bindingName, service, bindingsByKey[bindingKey])
		if err != nil {
			appErrs = append(appErrs, err)
			retain(bindingKey)
			continue
		}

		if binding.Data["serviceName"] != "" {
			desiredServices[binding.Namespace+"/"+binding.Data["serviceName"]] = struct{}{}
		}
		desiredBindings[bindingKey] = struct{}{}
		desiredPolicies[bindingKey] = struct{}{}
	}

	if err := r.cleanupPolicies(ctx, desiredPolicies); err != nil {
		appErrs = append(appErrs, err)
	} else if err := r.cleanup(ctx, services, bindings, desiredServices, desiredBindings); err != nil {
		appErrs = append(appErrs, err)
	}

	if len(appErrs) > 0 {
		return fmt.Errorf("reconcile private network apps, failed apps kept their published objects: %w", errors.Join(appErrs...))
	}
	return nil
}
