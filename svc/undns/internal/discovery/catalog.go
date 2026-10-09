package discovery

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/svc/undns/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	podIPIndex   = "podIP"
	appIndex     = "app"
	serviceIndex = "service"
)

// Catalog is the resolver's read-only view of Krane's published discovery
// objects. Lookups read local informer caches and never call the Kubernetes
// API. A Catalog is safe for concurrent use.
type Catalog struct {
	pods        *trackedInformer
	connections *trackedInformer
	services    *trackedInformer
	slices      *trackedInformer
	topology    *trackedInformer
	clock       clock.Clock
	statusMu    sync.Mutex
	statuses    map[string]connectionStatus
	statusDirty atomic.Bool
}

// New creates a Catalog that watches Krane deployment Pods and private-dns
// objects in every namespace. Watches renew every watchTimeout and count as
// stale after twice that without contact. clk supplies the time for watch
// freshness and Service retirement deadlines. Call [Catalog.Run] to start it.
func New(client kubernetes.Interface, watchTimeout time.Duration, clk clock.Clock) (*Catalog, error) {
	c := new(Catalog)
	c.clock = clk
	var err error

	callers := labels.SelectorFromSet(labels.Set{privatenetwork.ManagedByLabel: "krane", privatenetwork.ComponentLabel: "deployment"}).String()
	discovery := labels.Set{privatenetwork.ManagedByLabel: "krane", privatenetwork.ComponentLabel: privatenetwork.DiscoveryComponent}
	published := labels.SelectorFromSet(discovery).String()

	pods := client.CoreV1().Pods("")
	c.pods, err = newTrackedInformer("pods", &corev1.Pod{}, cache.Indexers{podIPIndex: indexPodIP}, watchTimeout, clk, callers,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return pods.List(ctx, options)
		}, pods.Watch)
	if err != nil {
		return nil, err
	}

	configs := client.CoreV1().ConfigMaps("")
	c.connections, err = newTrackedInformer("connections", &corev1.ConfigMap{}, cache.Indexers{appIndex: indexConnection}, watchTimeout, clk, published,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return configs.List(ctx, options)
		}, configs.Watch)
	if err != nil {
		return nil, err
	}

	services := client.CoreV1().Services("")
	c.services, err = newTrackedInformer("services", &corev1.Service{}, nil, watchTimeout, clk, published,
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
	}, cache.Indexers{serviceIndex: indexSlice}, watchTimeout, clk, published,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return endpoints.List(ctx, options)
		}, endpoints.Watch)
	if err != nil {
		return nil, err
	}

	topologyConfigs := client.CoreV1().ConfigMaps(metav1.NamespaceSystem)
	discovery[privatenetwork.ComponentLabel] = privatenetwork.TopologyComponent
	c.topology, err = newTrackedInformer("topology", &corev1.ConfigMap{}, nil, watchTimeout, clk, labels.SelectorFromSet(discovery).String(),
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = "metadata.name=" + privatenetwork.TopologyConfigMap
			return topologyConfigs.List(ctx, options)
		}, func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = "metadata.name=" + privatenetwork.TopologyConfigMap
			return topologyConfigs.Watch(ctx, options)
		})
	if err != nil {
		return nil, err
	}

	for _, informer := range []*trackedInformer{c.connections, c.services, c.slices} {
		_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    func(any) { c.statusDirty.Store(true) },
			UpdateFunc: func(any, any) { c.statusDirty.Store(true) },
			DeleteFunc: func(any) { c.statusDirty.Store(true) },
		})
		if err != nil {
			return nil, fmt.Errorf("watch connection status changes: %w", err)
		}
	}

	return c, nil
}

// Run runs the watches until ctx is canceled and returns nil. Once per second
// it reports watch health metrics and, after the caches synchronize, refreshes
// connection states when discovery changed or a Service retirement deadline
// passed. Health metrics are reported as 0 from the start of Run, before the
// caches synchronize.
func (c *Catalog) Run(ctx context.Context) error {
	reportConnections(nil)
	c.statusDirty.Store(true)

	var group sync.WaitGroup
	for _, informer := range c.informers() {
		group.Go(func() { informer.RunWithContext(ctx) })
	}
	group.Go(func() { c.refresh(ctx) })
	group.Wait()
	return nil
}

// Ready reports whether every watch that private answers depend on is
// synchronized and fresh. The topology watch is excluded because answers fall
// back to known endpoints without it.
func (c *Catalog) Ready() bool {
	return c.pods.healthy() && c.connections.healthy() && c.services.healthy() && c.slices.healthy()
}

// IdentityReady reports whether the Pod watch is synchronized and fresh, so
// [Catalog.Identify] reflects current Pods.
func (c *Catalog) IdentityReady() bool {
	return c.pods.healthy()
}

func (c *Catalog) refresh(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var retirement time.Time
	for {
		c.reportHealth()
		if c.synced() && (c.statusDirty.Swap(false) || !retirement.IsZero() && !c.clock.Now().Before(retirement)) {
			retirement = c.updateStatuses()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Catalog) synced() bool {
	return c.pods.HasSynced() && c.connections.HasSynced() && c.services.HasSynced() && c.slices.HasSynced()
}

func (c *Catalog) reportHealth() {
	metrics.DiscoveryReady.Set(gaugeValue(c.Ready()))
	for _, informer := range c.informers() {
		metrics.DiscoveryWatchHealthy.WithLabelValues(informer.resource).Set(gaugeValue(informer.healthy()))
	}
}

func (c *Catalog) informers() []*trackedInformer {
	return []*trackedInformer{c.pods, c.connections, c.services, c.slices, c.topology}
}

func gaugeValue(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
