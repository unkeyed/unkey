package discovery

import (
	"net/netip"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	corev1 "k8s.io/api/core/v1"
)

// Caller is the Krane deployment that sent a query. Connections are scoped to
// the caller's workspace, project, and deployment, and a target Service must
// share the caller's namespace.
type Caller struct {
	Workspace  string
	Project    string
	Deployment string
	Namespace  string
}

// Identify returns the caller whose Pod uses ip. The Pod must be the only
// non-terminal Pod with that IP, Running, not terminating, not host-networked,
// and carry every Krane deployment identity label with a production or preview
// environment kind. Failures are an [*Error] with reason unknown_caller,
// ambiguous_caller, or ineligible_caller. The result reflects the Pod cache,
// so check [Catalog.IdentityReady] first.
func (c *Catalog) Identify(ip netip.Addr) (caller Caller, err error) {
	objects, err := c.pods.GetIndexer().ByIndex(podIPIndex, ip.Unmap().String())
	if err != nil {
		return caller, err
	}
	switch {
	case len(objects) == 0:
		return caller, fail(ReasonUnknownCaller, "no pod uses the source IP")
	case len(objects) > 1:
		return caller, fail(ReasonAmbiguousCaller, "several pods use the source IP")
	}

	pod := objects[0].(*corev1.Pod)
	l := pod.Labels
	err = assert.All(
		assert.False(pod.Spec.HostNetwork, "caller must use pod networking"),
		assert.True(pod.DeletionTimestamp == nil, "caller is terminating"),
		assert.Equal(pod.Status.Phase, corev1.PodRunning),
		assert.NotEmpty(pod.UID),
		assert.Equal(l[privatenetwork.ManagedByLabel], "krane"),
		assert.Equal(l[privatenetwork.ComponentLabel], "deployment"),
		assert.NotEmpty(l[privatenetwork.WorkspaceLabel]),
		assert.NotEmpty(l[privatenetwork.ProjectLabel]),
		assert.NotEmpty(l[privatenetwork.AppLabel]),
		assert.NotEmpty(l[privatenetwork.EnvironmentLabel]),
		assert.NotEmpty(l[privatenetwork.DeploymentLabel]),
	)
	if err != nil {
		return caller, fail(ReasonIneligibleCaller, "source pod is not a running Krane deployment pod: %w", err)
	}

	if kind := l[privatenetwork.EnvironmentKindLabel]; kind != "production" && kind != "preview" {
		return caller, fail(ReasonIneligibleCaller, "source pod is not a running Krane deployment pod: environment kind %q", kind)
	}

	return Caller{
		Workspace:  l[privatenetwork.WorkspaceLabel],
		Project:    l[privatenetwork.ProjectLabel],
		Deployment: l[privatenetwork.DeploymentLabel],
		Namespace:  pod.Namespace,
	}, nil
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
