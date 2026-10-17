package discovery

import (
	"net/netip"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
)

func (c *Catalog) resolveConnection(caller Caller, config *corev1.ConfigMap, selector string) ([]netip.Addr, error) {
	b, err := parseConnection(config)
	if err != nil {
		return nil, err
	}

	object, exists, err := c.services.GetStore().GetByKey(b.namespace + "/" + b.service)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fail(ReasonServiceMissing, "selected discovery service is unavailable")
	}

	service := object.(*corev1.Service)
	if expiry := service.Annotations[privatenetwork.RetireAfterAnnotation]; expiry != "" {
		deadline, err := time.Parse(time.RFC3339Nano, expiry)
		if err != nil {
			return nil, fail(ReasonServiceRejected, "selected discovery service does not match connection: invalid retirement deadline: %w", err)
		}
		if !c.clock.Now().Before(deadline) {
			return nil, fail(ReasonServiceRetired, "discovery replacement overlap expired")
		}
	}

	l := service.Labels
	err = assert.All(
		assert.True(service.DeletionTimestamp == nil, "service is terminating"),
		assert.NotEmpty(service.UID),
		assert.Equal(service.Spec.ClusterIP, corev1.ClusterIPNone),
		assert.False(service.Spec.PublishNotReadyAddresses),
		assert.Equal(l[privatenetwork.WorkspaceLabel], caller.Workspace),
		assert.Equal(l[privatenetwork.ProjectLabel], caller.Project),
		assert.Equal(l[privatenetwork.AppLabel], b.appID),
		assert.Equal(l[privatenetwork.DeploymentLabel], b.deployment),
		assert.Equal(l[privatenetwork.ManagedByLabel], "krane"),
		assert.Equal(l[privatenetwork.ComponentLabel], privatenetwork.DiscoveryComponent),
		assert.Empty(l[privatenetwork.CallerDeploymentLabel]),
		assert.Empty(l[privatenetwork.ConnectionLabel]),
		assert.Equal(service.Namespace, caller.Namespace),
	)
	if err != nil {
		return nil, fail(ReasonServiceRejected, "selected discovery service does not match connection: %w", err)
	}

	return c.endpoints(service, selector)
}

func (c *Catalog) endpoints(service *corev1.Service, selector string) ([]netip.Addr, error) {
	objects, err := c.slices.GetIndexer().ByIndex(serviceIndex, service.Namespace+"/"+service.Name)
	if err != nil {
		return nil, err
	}

	var topology map[string]string
	if selector != "" {
		topology, err = c.clusterRegions()
		if err != nil {
			return nil, err
		}
	}

	var addresses, selected []netip.Addr
	for _, object := range objects {
		slice := object.(*discoveryv1.EndpointSlice)
		start := len(addresses)
		addresses = privatenetwork.AppendReadyAddresses(addresses, service, slice)
		if selector != "" && sliceMatchesRegion(slice, topology, selector) {
			selected = append(selected, addresses[start:]...)
		}
	}
	if selector != "" && (selector != "local-first" || len(selected) > 0) {
		addresses = selected
	}

	slices.SortFunc(addresses, func(a, b netip.Addr) int { return a.Compare(b) })
	addresses = slices.Compact(addresses)
	if len(addresses) == 0 {
		return nil, fail(ReasonNoReadyEndpoints, "selected deployment has no ready endpoints")
	}

	return addresses, nil
}

func indexSlice(object any) ([]string, error) {
	slice := object.(*discoveryv1.EndpointSlice)
	return []string{slice.Namespace + "/" + slice.Labels[discoveryv1.LabelServiceName]}, nil
}
