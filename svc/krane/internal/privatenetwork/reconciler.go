// Package privatenetwork publishes private network discovery objects for
// undns.
package privatenetwork

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"sync"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	privatecontract "github.com/unkeyed/unkey/pkg/privatenetwork"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
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

func (r *Reconciler) snapshot(ctx context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
	stream, err := r.cluster.StreamPrivateNetworkState(ctx, &ctrlv1.StreamPrivateNetworkStateRequest{Cluster: r.clusterKey})
	if err != nil {
		return nil, fmt.Errorf("get complete private network snapshot: %w", err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Warn("close private network snapshot stream", "error", err)
		}
	}()

	var apps []*ctrlv1.PrivateNetworkApp
	for stream.Receive() {
		chunk := stream.Msg()
		if chunk.GetComplete() {
			if chunk.GetTotal() != uint64(len(apps)) || len(chunk.GetApps()) != 0 {
				return nil, fmt.Errorf("private network snapshot has %d apps, complete chunk reports %d", len(apps), chunk.GetTotal())
			}
			return apps, nil
		}
		apps = append(apps, chunk.GetApps()...)
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("get complete private network snapshot: %w", err)
	}
	return nil, fmt.Errorf("private network snapshot ended before it was complete")
}

type snapshotRejection struct {
	retainedBindingKey string
	err                error
}

func validateSnapshot(apps []*ctrlv1.PrivateNetworkApp) ([]*ctrlv1.PrivateNetworkApp, []snapshotRejection) {
	identities := make(map[string]int, len(apps))
	for _, app := range apps {
		if app != nil {
			identities[appIdentity(app)]++
		}
	}
	eligible := make([]*ctrlv1.PrivateNetworkApp, 0, len(apps))
	var rejected []snapshotRejection
	for i, app := range apps {
		if app == nil {
			rejected = append(rejected, snapshotRejection{retainedBindingKey: "", err: fmt.Errorf("invalid private network snapshot app %d: app is nil", i)})
			continue
		}
		err := assert.All(
			assert.NotEmpty(app.GetWorkspaceId(), "workspace ID is required"),
			assert.NotEmpty(app.GetProjectId(), "project ID is required"),
			assert.NotEmpty(app.GetAppId(), "app ID is required"),
			assert.NotEmpty(app.GetCallerDeploymentId(), "caller deployment ID is required"),
			assert.NotEmpty(app.GetBindingId(), "binding ID is required"),
			assert.Equal(len(validation.IsValidLabelValue(app.GetCallerDeploymentId())), 0, "invalid caller deployment ID"),
			assert.Equal(len(validation.IsValidLabelValue(app.GetBindingId())), 0, "invalid binding ID"),
			assert.Equal(len(validation.IsDNS1123Label(app.GetBindingName())), 0, "invalid binding name"),
			assert.Equal(len(validation.IsDNS1123Label(app.GetK8SNamespace())), 0, "invalid Kubernetes namespace"),
			assert.True(app.GetDeploymentId() == "" || app.GetPort() > 0, "resolved target port must be positive"),
			assert.GreaterOrEqual(app.GetPort(), int32(0), "port must not be negative"),
			assert.LessOrEqual(app.GetPort(), int32(65535), "port must be at most 65535"),
			assert.Equal(identities[appIdentity(app)], 1, "duplicate app identity"),
		)
		if err != nil {
			rejected = append(rejected, snapshotRejection{
				retainedBindingKey: publishedBindingKey(app),
				err:                fmt.Errorf("invalid private network snapshot app %d: %w", i, err),
			})
			continue
		}
		eligible = append(eligible, app)
	}
	return eligible, rejected
}

func appIdentity(app *ctrlv1.PrivateNetworkApp) string {
	return app.GetWorkspaceId() + "/" + app.GetProjectId() + "/" + app.GetCallerDeploymentId() + "/" + app.GetBindingName()
}

func publishedBindingKey(app *ctrlv1.PrivateNetworkApp) string {
	if len(validation.IsDNS1123Label(app.GetK8SNamespace())) != 0 || app.GetBindingId() == "" || app.GetCallerDeploymentId() == "" {
		return ""
	}
	return app.GetK8SNamespace() + "/" + bindingResourceName(app)
}

func bindingResourceName(app *ctrlv1.PrivateNetworkApp) string {
	return resourceName("unkey-pn-binding", app.GetBindingId()+"/"+app.GetCallerDeploymentId())
}

func (r *Reconciler) namespaces(ctx context.Context) (map[string]struct{}, error) {
	list, err := r.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	namespaces := make(map[string]struct{}, len(list.Items))
	for i := range list.Items {
		namespaces[list.Items[i].Name] = struct{}{}
	}
	return namespaces, nil
}

func appLabels(app *ctrlv1.PrivateNetworkApp) labels.Labels {
	return labels.New().
		ManagedByKrane().
		WorkspaceID(app.GetWorkspaceId()).
		ProjectID(app.GetProjectId()).
		AppID(app.GetAppId())
}

func (r *Reconciler) ensureService(ctx context.Context, app *ctrlv1.PrivateNetworkApp, name string, existing *corev1.Service) (*corev1.Service, error) {
	client := r.client.CoreV1().Services(app.GetK8SNamespace())
	desiredLabels := appLabels(app).DeploymentID(app.GetDeploymentId())
	desiredLabels[labels.LabelKeyComponent] = component
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   app.GetK8SNamespace(),
			Labels:      desiredLabels,
			Annotations: map[string]string{ciliumGlobal: "true", ciliumGlobalSlices: "true"},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{{
				Name:       "app",
				Port:       app.GetPort(),
				TargetPort: intstr.FromInt32(app.GetPort()),
				Protocol:   corev1.ProtocolTCP,
			}},
			PublishNotReadyAddresses: false,
		},
	}
	if existing != nil {
		if !ownedByApp(existing.Labels, app) {
			return nil, fmt.Errorf("refuse to replace foreign Service %s/%s", app.GetK8SNamespace(), name)
		}

		if maps.Equal(existing.Labels, desired.Labels) && maps.Equal(existing.Annotations, desired.Annotations) &&
			maps.Equal(existing.Spec.Selector, desired.Spec.Selector) && existing.Spec.ClusterIP == corev1.ClusterIPNone &&
			!existing.Spec.PublishNotReadyAddresses && len(existing.Spec.Ports) == 1 &&
			existing.Spec.Ports[0] == desired.Spec.Ports[0] {
			return existing, nil
		}

		desired.ResourceVersion = existing.ResourceVersion
		desired.UID = existing.UID
		desired.Spec.ClusterIPs = existing.Spec.ClusterIPs
		desired.Spec.IPFamilies = existing.Spec.IPFamilies
		desired.Spec.IPFamilyPolicy = existing.Spec.IPFamilyPolicy
		updated, err := client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager})
		if err != nil {
			return nil, fmt.Errorf("update private network Service %s/%s: %w", app.GetK8SNamespace(), name, err)
		}
		return updated, nil
	}

	created, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	if err != nil {
		return nil, fmt.Errorf("create private network Service %s/%s: %w", app.GetK8SNamespace(), name, err)
	}
	return created, nil
}

func (r *Reconciler) ensureBinding(ctx context.Context, app *ctrlv1.PrivateNetworkApp, name string, service *corev1.Service, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	client := r.client.CoreV1().ConfigMaps(app.GetK8SNamespace())
	key := app.GetK8SNamespace() + "/" + name
	desiredData := map[string]string{
		"appSlug":      app.GetBindingName(),
		"deploymentId": app.GetDeploymentId(),
		"serviceName":  "",
	}
	if service != nil {
		desiredData["serviceName"] = service.Name
	}
	if existing != nil {
		if !owned(existing.Labels) || existing.Labels[labels.LabelKeyWorkspaceID] != app.GetWorkspaceId() ||
			existing.Labels[labels.LabelKeyProjectID] != app.GetProjectId() ||
			existing.Labels[labels.LabelKeyBindingID] != app.GetBindingId() ||
			existing.Labels[labels.LabelKeyCallerDeploymentID] != app.GetCallerDeploymentId() {
			return nil, fmt.Errorf("refuse to replace foreign ConfigMap %s", key)
		}
		revision, parseErr := strconv.ParseUint(existing.Data["revision"], 10, 64)
		if parseErr != nil || revision == 0 {
			return nil, fmt.Errorf("invalid existing binding revision for %s", key)
		}

		if existing.Data["appSlug"] == desiredData["appSlug"] && existing.Data["deploymentId"] == desiredData["deploymentId"] && existing.Data["serviceName"] == desiredData["serviceName"] && maps.Equal(existing.Labels, bindingLabels(app)) {
			return existing, nil
		}

		if service != nil && existing.Data["appSlug"] == desiredData["appSlug"] && maps.Equal(existing.Labels, bindingLabels(app)) {
			ready, err := r.hasReadyEndpoints(ctx, service)
			if err != nil {
				return nil, err
			}
			if !ready {
				return existing, nil
			}
		}

		if revision == ^uint64(0) {
			return nil, fmt.Errorf("binding revision exhausted for %s", key)
		}
		desiredData["revision"] = strconv.FormatUint(revision+1, 10)
		desired := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: app.GetK8SNamespace(), Labels: bindingLabels(app),
			ResourceVersion: existing.ResourceVersion, UID: existing.UID,
		}, Data: desiredData}
		updated, err := client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager})
		if err != nil {
			return nil, fmt.Errorf("update private network binding %s: %w", key, err)
		}
		return updated, nil
	}

	desiredData["revision"] = "1"
	desired := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: app.GetK8SNamespace(), Labels: bindingLabels(app),
	}, Data: desiredData}
	created, err := client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	if err != nil {
		return nil, fmt.Errorf("create private network binding %s: %w", key, err)
	}
	return created, nil
}

func (r *Reconciler) hasReadyEndpoints(ctx context.Context, service *corev1.Service) (bool, error) {
	slices, err := r.client.DiscoveryV1().EndpointSlices(service.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: discoveryv1.LabelServiceName + "=" + service.Name,
	})
	if err != nil {
		return false, fmt.Errorf("check discovery readiness for %s/%s: %w", service.Namespace, service.Name, err)
	}
	for i := range slices.Items {
		if len(privatecontract.AppendReadyAddresses(nil, service, &slices.Items[i])) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func bindingLabels(app *ctrlv1.PrivateNetworkApp) labels.Labels {
	l := appLabels(app)
	l[labels.LabelKeyComponent] = component
	l[labels.LabelKeyCallerDeploymentID] = app.GetCallerDeploymentId()
	l[labels.LabelKeyBindingID] = app.GetBindingId()
	return l
}

func (r *Reconciler) cleanup(ctx context.Context, services *corev1.ServiceList, bindings *corev1.ConfigMapList, desiredServices, desiredBindings map[string]struct{}) error {
	for i := range bindings.Items {
		item := &bindings.Items[i]
		if _, ok := desiredBindings[item.Namespace+"/"+item.Name]; ok || !owned(item.Labels) {
			continue
		}
		if err := r.client.CoreV1().ConfigMaps(item.Namespace).Delete(ctx, item.Name, deleteOptions(item)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete private network binding %s/%s: %w", item.Namespace, item.Name, err)
		}
	}
	for i := range services.Items {
		item := &services.Items[i]
		if _, ok := desiredServices[item.Namespace+"/"+item.Name]; ok || !owned(item.Labels) {
			continue
		}
		deadline, ok := retirementDeadline(item)
		if !ok {
			updated := item.DeepCopy()
			updated.Annotations = maps.Clone(item.Annotations)
			if updated.Annotations == nil {
				updated.Annotations = make(map[string]string)
			}
			updated.Annotations[privatecontract.RetireAfterAnnotation] = r.clock().Add(privatecontract.ReplacementOverlap).UTC().Format(time.RFC3339Nano)
			if _, err := r.client.CoreV1().Services(item.Namespace).Update(ctx, updated, metav1.UpdateOptions{FieldManager: fieldManager}); err != nil {
				return fmt.Errorf("schedule obsolete private network Service %s/%s retirement: %w", item.Namespace, item.Name, err)
			}
			continue
		}
		if r.clock().Before(deadline) {
			continue
		}
		if err := r.client.CoreV1().Services(item.Namespace).Delete(ctx, item.Name, deleteOptions(item)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete private network Service %s/%s: %w", item.Namespace, item.Name, err)
		}
	}
	return nil
}

func (r *Reconciler) clock() time.Time {
	if r.now != nil {
		return r.now()
	}
	return time.Now()
}

func retirementDeadline(service *corev1.Service) (time.Time, bool) {
	raw := service.Annotations[privatecontract.RetireAfterAnnotation]
	if raw == "" {
		return time.Time{}, false
	}
	deadline, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, true
	}
	return deadline, true
}

func owned(l map[string]string) bool {
	return l[labels.LabelKeyManagedBy] == "krane" &&
		l[labels.LabelKeyComponent] == component &&
		l[labels.LabelKeyWorkspaceID] != "" &&
		l[labels.LabelKeyProjectID] != "" &&
		l[labels.LabelKeyAppID] != ""
}

func ownedByApp(l map[string]string, app *ctrlv1.PrivateNetworkApp) bool {
	return owned(l) &&
		l[labels.LabelKeyWorkspaceID] == app.GetWorkspaceId() &&
		l[labels.LabelKeyProjectID] == app.GetProjectId() &&
		l[labels.LabelKeyAppID] == app.GetAppId()
}

func discoveryName(deployment string, port int32) string {
	return resourceName("unkey-pn-v3", deployment+"/"+strconv.Itoa(int(port)))
}

func deleteOptions(object metav1.Object) metav1.DeleteOptions {
	uid := object.GetUID()
	version := object.GetResourceVersion()
	return metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid, ResourceVersion: &version}}
}

func resourceName(prefix, id string) string {
	sum := sha256.Sum256([]byte(id))
	return prefix + "-" + hex.EncodeToString(sum[:16])
}
