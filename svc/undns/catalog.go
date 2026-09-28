package undns

import (
	"context"
	"fmt"
	"maps"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
	privatecontract "github.com/unkeyed/unkey/pkg/privatenetwork"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	environmentKindLabel = "unkey.com/environment.kind"
	bindingComponent     = "private-dns"
	podIPIndex           = "podIP"
	appIndex             = "app"
	serviceIndex         = "service"
)

type caller struct {
	workspace  string
	project    string
	kind       string
	deployment string
	namespace  string
}

type binding struct {
	appID      string
	deployment string
	service    string
	namespace  string
	revision   uint64
	resolved   bool
}

type catalog struct {
	pods      *trackedInformer
	bindings  *trackedInformer
	services  *trackedInformer
	slices    *trackedInformer
	activeMu  sync.RWMutex
	active    map[string]*corev1.ConfigMap
	activated bool
	now       func() time.Time
}

func newCatalog(client kubernetes.Interface, timeout time.Duration) (*catalog, error) {
	c := new(catalog)
	c.active = make(map[string]*corev1.ConfigMap)
	c.now = time.Now
	var err error

	pods := client.CoreV1().Pods("")
	c.pods, err = newTrackedInformer("pods", &corev1.Pod{}, cache.Indexers{podIPIndex: indexPodIP}, timeout,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return pods.List(ctx, options)
		}, pods.Watch)
	if err != nil {
		return nil, err
	}

	configs := client.CoreV1().ConfigMaps("")
	c.bindings, err = newTrackedInformer("bindings", &corev1.ConfigMap{}, cache.Indexers{appIndex: indexBinding}, timeout,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return configs.List(ctx, options)
		}, configs.Watch)
	if err != nil {
		return nil, err
	}

	services := client.CoreV1().Services("")
	c.services, err = newTrackedInformer("services", &corev1.Service{}, nil, timeout,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return services.List(ctx, options)
		}, services.Watch)
	if err != nil {
		return nil, err
	}

	endpoints := client.DiscoveryV1().EndpointSlices("")
	c.slices, err = newTrackedInformer("endpointslices", &discoveryv1.EndpointSlice{
		TypeMeta:    metav1.TypeMeta{},
		ObjectMeta:  metav1.ObjectMeta{},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   nil,
		Ports:       nil,
	}, cache.Indexers{serviceIndex: indexSlice}, timeout,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return endpoints.List(ctx, options)
		}, endpoints.Watch)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *catalog) run(ctx context.Context) error {
	var group sync.WaitGroup
	for _, informer := range []*trackedInformer{c.pods, c.bindings, c.services, c.slices} {
		group.Go(func() { informer.RunWithContext(ctx) })
	}

	group.Go(func() {
		if !cache.WaitForCacheSync(ctx.Done(), c.pods.HasSynced, c.bindings.HasSynced, c.services.HasSynced, c.slices.HasSynced) {
			return
		}
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			if c.readyDiscovery() {
				c.activate()
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	})
	group.Wait()
	return nil
}

func (c *catalog) ready() bool {
	c.activeMu.RLock()
	defer c.activeMu.RUnlock()
	return c.activated && c.readyDiscovery()
}

func (c *catalog) readyDiscovery() bool {
	return c.pods.healthy() && c.bindings.healthy() && c.services.healthy() && c.slices.healthy()
}

func (c *catalog) identify(ip netip.Addr) (caller, bool) {
	objects, err := c.pods.GetIndexer().ByIndex(podIPIndex, ip.Unmap().String())
	if err != nil || len(objects) != 1 {
		return emptyCaller(), false
	}

	pod := objects[0].(*corev1.Pod)
	l := pod.Labels
	err = assert.All(
		assert.False(pod.Spec.HostNetwork, "caller must use pod networking"),
		assert.True(pod.DeletionTimestamp == nil, "caller is terminating"),
		assert.Equal(pod.Status.Phase, corev1.PodRunning),
		assert.NotEmpty(pod.UID),
		assert.Equal(l[labels.LabelKeyManagedBy], "krane"),
		assert.Equal(l[labels.LabelKeyComponent], "deployment"),
		assert.NotEmpty(l[labels.LabelKeyWorkspaceID]),
		assert.NotEmpty(l[labels.LabelKeyProjectID]),
		assert.NotEmpty(l[labels.LabelKeyAppID]),
		assert.NotEmpty(l[labels.LabelKeyEnvironmentID]),
		assert.NotEmpty(l[labels.LabelKeyDeploymentID]),
	)
	if err != nil {
		return emptyCaller(), false
	}

	kind := l[environmentKindLabel]
	if kind != "production" && kind != "preview" {
		return emptyCaller(), false
	}

	return caller{
		workspace:  l[labels.LabelKeyWorkspaceID],
		project:    l[labels.LabelKeyProjectID],
		kind:       kind,
		deployment: l[labels.LabelKeyDeploymentID],
		namespace:  pod.Namespace,
	}, true
}

func emptyCaller() caller {
	return caller{workspace: "", project: "", kind: "", deployment: "", namespace: ""}
}

func (c *catalog) resolve(identity caller, app string) ([]netip.Addr, bool, error) {
	if identity.deployment == "" {
		return nil, false, nil
	}
	c.activeMu.Lock()
	defer c.activeMu.Unlock()
	key := appKey(identity.workspace, identity.project, identity.deployment, app)
	objects, err := c.bindings.GetIndexer().ByIndex(appIndex, key)
	if err != nil {
		return nil, false, err
	}

	if len(objects) == 0 {
		delete(c.active, key)
		return nil, false, nil
	}

	if len(objects) != 1 {
		delete(c.active, key)
		return nil, true, fmt.Errorf("ambiguous app binding")
	}

	config := objects[0].(*corev1.ConfigMap)
	active, err := c.activateBinding(key, config)
	if err != nil {
		return nil, true, err
	}
	addresses, err := c.resolveBinding(identity, active)
	return addresses, true, err
}

func (c *catalog) activate() {
	c.activeMu.Lock()
	defer c.activeMu.Unlock()
	desired := make(map[string]*corev1.ConfigMap)
	ambiguous := make(map[string]bool)
	for _, object := range c.bindings.GetStore().List() {
		config := object.(*corev1.ConfigMap)
		keys, err := indexBinding(config)
		if err != nil || len(keys) == 0 {
			continue
		}
		if len(keys) != 1 {
			continue
		}
		if _, exists := desired[keys[0]]; exists {
			ambiguous[keys[0]] = true
		}
		desired[keys[0]] = config
	}

	for key, active := range c.active {
		config := desired[key]
		if ambiguous[key] || !sameBinding(active, config) {
			delete(c.active, key)
		}
	}
	for key, config := range desired {
		if ambiguous[key] {
			continue
		}
		if _, err := c.activateBinding(key, config); err != nil {
			continue
		}
	}
	c.activated = true
}

func sameBinding(a, b *corev1.ConfigMap) bool {
	return a != nil && b != nil && a.UID == b.UID && a.Namespace == b.Namespace && a.Name == b.Name &&
		a.Labels[labels.LabelKeyAppID] == b.Labels[labels.LabelKeyAppID] &&
		a.Labels[labels.LabelKeyWorkspaceID] == b.Labels[labels.LabelKeyWorkspaceID] &&
		a.Labels[labels.LabelKeyProjectID] == b.Labels[labels.LabelKeyProjectID] &&
		a.Labels[labels.LabelKeyCallerDeploymentID] == b.Labels[labels.LabelKeyCallerDeploymentID] &&
		a.Labels[labels.LabelKeyBindingID] == b.Labels[labels.LabelKeyBindingID] &&
		a.Data["appSlug"] == b.Data["appSlug"]
}

func (c *catalog) activateBinding(key string, config *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	candidate, err := parseBinding(config)
	if err != nil {
		delete(c.active, key)
		return nil, err
	}
	active := c.active[key]
	if !sameBinding(active, config) {
		delete(c.active, key)
		active = nil
	}
	if active != nil {
		current, err := parseBinding(active)
		if err != nil {
			return nil, err
		}
		if candidate.revision == current.revision && !maps.Equal(active.Data, config.Data) {
			return nil, fmt.Errorf("binding changed without advancing revision")
		}
		if candidate.revision <= current.revision {
			return active, nil
		}
	}
	if !candidate.resolved {
		delete(c.active, key)
		return nil, fmt.Errorf("binding target is unresolved")
	}
	identity := caller{
		workspace:  config.Labels[labels.LabelKeyWorkspaceID],
		project:    config.Labels[labels.LabelKeyProjectID],
		kind:       "",
		deployment: config.Labels[labels.LabelKeyCallerDeploymentID],
		namespace:  config.Namespace,
	}
	if _, err := c.resolveBinding(identity, config); err != nil {
		if active != nil {
			return active, nil
		}
		return nil, err
	}
	if c.active == nil {
		c.active = make(map[string]*corev1.ConfigMap)
	}
	c.active[key] = config.DeepCopy()
	return c.active[key], nil
}

func (c *catalog) resolveBinding(identity caller, config *corev1.ConfigMap) ([]netip.Addr, error) {
	b, err := parseBinding(config)
	if err != nil {
		return nil, err
	}

	object, exists, err := c.services.GetStore().GetByKey(b.namespace + "/" + b.service)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("selected discovery service is unavailable")
	}

	service := object.(*corev1.Service)
	if expiry := service.Annotations[privatecontract.RetireAfterAnnotation]; expiry != "" {
		deadline, err := time.Parse(time.RFC3339Nano, expiry)
		if err != nil {
			return nil, fmt.Errorf("invalid discovery retirement deadline: %w", err)
		}
		if !c.now().Before(deadline) {
			return nil, fmt.Errorf("discovery replacement overlap expired")
		}
	}
	l := service.Labels
	err = assert.All(
		assert.True(service.DeletionTimestamp == nil, "service is terminating"),
		assert.NotEmpty(service.UID),
		assert.Equal(service.Spec.ClusterIP, corev1.ClusterIPNone),
		assert.False(service.Spec.PublishNotReadyAddresses),
		assert.Equal(l[labels.LabelKeyWorkspaceID], identity.workspace),
		assert.Equal(l[labels.LabelKeyProjectID], identity.project),
		assert.Equal(l[labels.LabelKeyAppID], b.appID),
		assert.Equal(l[labels.LabelKeyDeploymentID], b.deployment),
		assert.Equal(l[labels.LabelKeyManagedBy], "krane"),
		assert.Equal(l[labels.LabelKeyComponent], bindingComponent),
		assert.Empty(l[labels.LabelKeyCallerDeploymentID]),
		assert.Empty(l[labels.LabelKeyBindingID]),
		assert.Equal(service.Namespace, identity.namespace),
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("selected discovery service does not match binding"))
	}

	return c.endpoints(service)
}

func (c *catalog) endpoints(service *corev1.Service) ([]netip.Addr, error) {
	objects, err := c.slices.GetIndexer().ByIndex(serviceIndex, service.Namespace+"/"+service.Name)
	if err != nil {
		return nil, err
	}

	var addresses []netip.Addr
	for _, object := range objects {
		addresses = privatecontract.AppendReadyAddresses(addresses, service, object.(*discoveryv1.EndpointSlice))
	}

	slices.SortFunc(addresses, func(a, b netip.Addr) int { return a.Compare(b) })
	addresses = slices.Compact(addresses)
	if len(addresses) == 0 {
		return nil, fmt.Errorf("selected deployment has no ready endpoints")
	}

	return addresses, nil
}

func parseBinding(config *corev1.ConfigMap) (binding, error) {
	revision, err := strconv.ParseUint(config.Data["revision"], 10, 64)
	if err != nil {
		return binding{}, fault.Wrap(err, fault.Internal("invalid binding revision"))
	}

	err = assert.All(
		assert.True(revision > 0, "binding revision must be positive"),
		assert.NotEmpty(config.UID),
		assert.True(config.DeletionTimestamp == nil, "binding is terminating"),
		assert.NotEmpty(config.Labels[labels.LabelKeyCallerDeploymentID], "caller deployment ID is required"),
		assert.NotEmpty(config.Labels[labels.LabelKeyBindingID], "binding ID is required"),
		assert.NotEmpty(config.Labels[labels.LabelKeyAppID]),
		assert.Equal(config.Data["deploymentId"] == "", config.Data["serviceName"] == "", "binding target must be entirely resolved or unresolved"),
	)
	if config.Data["serviceName"] != "" {
		err = assert.All(err, assert.Equal(len(validation.IsDNS1035Label(config.Data["serviceName"])), 0, "invalid service name"))
	}
	if err != nil {
		return binding{}, fault.Wrap(err, fault.Internal("invalid app binding"))
	}

	return binding{
		appID:      config.Labels[labels.LabelKeyAppID],
		deployment: config.Data["deploymentId"],
		service:    config.Data["serviceName"],
		namespace:  config.Namespace,
		revision:   revision,
		resolved:   config.Data["deploymentId"] != "",
	}, nil
}

func appKey(workspace, project, callerDeployment, app string) string {
	return workspace + "/" + project + "/" + callerDeployment + "/" + app
}

func indexPodIP(object any) ([]string, error) {
	pod := object.(*corev1.Pod)
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return nil, nil
	}

	var keys []string
	for _, ip := range pod.Status.PodIPs {
		if address, err := netip.ParseAddr(ip.IP); err == nil {
			keys = append(keys, address.Unmap().String())
		}
	}

	if address, err := netip.ParseAddr(pod.Status.PodIP); err == nil {
		keys = append(keys, address.Unmap().String())
	}

	slices.Sort(keys)
	return slices.Compact(keys), nil
}

func indexBinding(object any) ([]string, error) {
	config := object.(*corev1.ConfigMap)
	l := config.Labels
	app := config.Data["appSlug"]
	if l[labels.LabelKeyManagedBy] != "krane" || l[labels.LabelKeyComponent] != bindingComponent ||
		l[labels.LabelKeyWorkspaceID] == "" || l[labels.LabelKeyProjectID] == "" ||
		l[labels.LabelKeyCallerDeploymentID] == "" || l[labels.LabelKeyBindingID] == "" ||
		l[labels.LabelKeyAppID] == "" || app == "" || strings.ToLower(app) != app || len(validation.IsDNS1123Label(app)) != 0 {
		return nil, nil
	}

	return []string{appKey(l[labels.LabelKeyWorkspaceID], l[labels.LabelKeyProjectID], l[labels.LabelKeyCallerDeploymentID], app)}, nil
}

func indexSlice(object any) ([]string, error) {
	slice := object.(*discoveryv1.EndpointSlice)
	return []string{slice.Namespace + "/" + slice.Labels[discoveryv1.LabelServiceName]}, nil
}
