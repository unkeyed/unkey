package peer

import (
	"context"
	"fmt"
	"net"
	"strconv"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Discovery returns relay addresses (host:port) for a peer region.
type Discovery interface {
	Addresses(region string) []string
}

// StaticDiscovery maps a region to fixed relay addresses.
type StaticDiscovery map[string][]string

func (s StaticDiscovery) Addresses(region string) []string { return s[region] }

const sourceClusterLabel = "multicluster.kubernetes.io/source-cluster"

// EndpointSliceDiscovery reads the EndpointSlices that ClusterMesh syncs for
// the global relay Service. Synced slices carry the source cluster label but
// no real readiness, so they are only the candidate set; the pool's own dials
// and PINGs decide health.
type EndpointSliceDiscovery struct {
	indexer cache.Indexer
	cells   map[string]string
	port    int
}

// NewEndpointSliceDiscovery watches EndpointSlices of service in namespace.
// cells maps a ClusterMesh cluster name to a region such as "us-east-1.aws".
func NewEndpointSliceDiscovery(ctx context.Context, client kubernetes.Interface, namespace, service string, port int, cells map[string]string) (*EndpointSliceDiscovery, error) {
	factory := informers.NewSharedInformerFactoryWithOptions(client, 0,
		informers.WithNamespace(namespace),
		informers.WithTweakListOptions(func(o *metav1.ListOptions) {
			o.LabelSelector = discoveryv1.LabelServiceName + "=" + service
		}),
	)
	informer := factory.Discovery().V1().EndpointSlices().Informer()
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return nil, fmt.Errorf("endpointslice informer did not sync")
	}
	return &EndpointSliceDiscovery{indexer: informer.GetIndexer(), cells: cells, port: port}, nil
}

func (d *EndpointSliceDiscovery) Addresses(region string) []string {
	var out []string
	for _, obj := range d.indexer.List() {
		slice, ok := obj.(*discoveryv1.EndpointSlice)
		if !ok {
			continue
		}
		if d.cells[slice.Labels[sourceClusterLabel]] != region {
			continue
		}
		port := strconv.Itoa(d.port)
		for _, ep := range slice.Endpoints {
			if ep.Conditions.Terminating != nil && *ep.Conditions.Terminating {
				continue
			}
			for _, addr := range ep.Addresses {
				out = append(out, net.JoinHostPort(addr, port))
			}
		}
	}
	return out
}
