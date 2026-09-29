package undns

import (
	"fmt"
	"net/netip"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
)

type caller struct {
	workspace  string
	project    string
	kind       string
	deployment string
	namespace  string
}

func (c *catalog) identify(ip netip.Addr) (caller, error) {
	objects, err := c.pods.GetIndexer().ByIndex(podIPIndex, ip.Unmap().String())
	if err != nil {
		return emptyCaller(), err
	}
	switch {
	case len(objects) == 0:
		return emptyCaller(), errUnknownCaller
	case len(objects) > 1:
		return emptyCaller(), errAmbiguousCaller
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
		return emptyCaller(), fmt.Errorf("%w: %w", errIneligibleCaller, err)
	}

	kind := l[environmentKindLabel]
	if kind != "production" && kind != "preview" {
		return emptyCaller(), fmt.Errorf("%w: environment kind %q", errIneligibleCaller, kind)
	}

	return caller{
		workspace:  l[labels.LabelKeyWorkspaceID],
		project:    l[labels.LabelKeyProjectID],
		kind:       kind,
		deployment: l[labels.LabelKeyDeploymentID],
		namespace:  pod.Namespace,
	}, nil
}

func emptyCaller() caller {
	return caller{workspace: "", project: "", kind: "", deployment: "", namespace: ""}
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
