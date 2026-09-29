package undns

import (
	"context"
	"sync"
	"time"

	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

type catalog struct {
	pods      *trackedInformer
	bindings  *trackedInformer
	services  *trackedInformer
	slices    *trackedInformer
	activeMu  sync.RWMutex
	active    map[string]*corev1.ConfigMap
	activated bool
	statusMu  sync.Mutex
	statuses  map[string]bindingStatus
	now       func() time.Time
}

func newCatalog(client kubernetes.Interface, timeout time.Duration) (*catalog, error) {
	c := new(catalog)
	c.active = make(map[string]*corev1.ConfigMap)
	c.now = time.Now
	var err error

	callers := labels.New().ManagedByKrane().ComponentDeployment().ToString()
	discovery := labels.New().ManagedByKrane()
	discovery[labels.LabelKeyComponent] = bindingComponent
	published := discovery.ToString()

	pods := client.CoreV1().Pods("")
	c.pods, err = newTrackedInformer("pods", &corev1.Pod{}, cache.Indexers{podIPIndex: indexPodIP}, timeout, callers,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return pods.List(ctx, options)
		}, pods.Watch)
	if err != nil {
		return nil, err
	}

	configs := client.CoreV1().ConfigMaps("")
	c.bindings, err = newTrackedInformer("bindings", &corev1.ConfigMap{}, cache.Indexers{appIndex: indexBinding}, timeout, published,
		func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return configs.List(ctx, options)
		}, configs.Watch)
	if err != nil {
		return nil, err
	}

	services := client.CoreV1().Services("")
	c.services, err = newTrackedInformer("services", &corev1.Service{}, nil, timeout, published,
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
	}, cache.Indexers{serviceIndex: indexSlice}, timeout, published,
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
