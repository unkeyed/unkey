package cilium

import (
	"context"

	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (c *Controller) listCiliumNetworkPolicies(ctx context.Context, cursor string) (*unstructured.UnstructuredList, error) {
	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumnetworkpolicies",
	}

	selector := labels.New().ManagedByKrane().ToString()
	return c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
		Continue:      cursor,
	})
}
