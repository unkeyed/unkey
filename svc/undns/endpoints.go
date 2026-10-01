package undns

import (
	"fmt"
	"net/netip"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/deploy/appconnection"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
)

func (c *catalog) resolveConnection(identity caller, config *corev1.ConfigMap, selector string) ([]netip.Addr, error) {
	b, err := parseConnection(config)
	if err != nil {
		return nil, err
	}

	object, exists, err := c.services.GetStore().GetByKey(b.namespace + "/" + b.service)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errServiceMissing
	}

	service := object.(*corev1.Service)
	if expiry := service.Annotations[appconnection.RetireAfterAnnotation]; expiry != "" {
		deadline, err := time.Parse(time.RFC3339Nano, expiry)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid retirement deadline: %w", errServiceRejected, err)
		}
		if !c.now().Before(deadline) {
			return nil, errServiceRetired
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
		assert.Equal(l[labels.LabelKeyComponent], connectionComponent),
		assert.Empty(l[labels.LabelKeyCallerDeploymentID]),
		assert.Empty(l[labels.LabelKeyConnectionID]),
		assert.Equal(service.Namespace, identity.namespace),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errServiceRejected, err)
	}

	return c.endpoints(service, selector)
}

func (c *catalog) endpoints(service *corev1.Service, selector string) ([]netip.Addr, error) {
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
		addresses = appconnection.AppendReadyAddresses(addresses, service, slice)
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
		return nil, errNoReadyEndpoints
	}

	return addresses, nil
}

func indexSlice(object any) ([]string, error) {
	slice := object.(*discoveryv1.EndpointSlice)
	return []string{slice.Namespace + "/" + slice.Labels[discoveryv1.LabelServiceName]}, nil
}
