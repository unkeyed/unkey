package cilium

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteCiliumNetworkPolicy removes a CiliumNetworkPolicy from the cluster.
//
// Not-found errors are ignored since the desired end state (resource gone) is
// already achieved. The method is idempotent: calling it multiple times for the
// same policy succeeds without error.
func (c *Controller) DeleteCiliumNetworkPolicy(ctx context.Context, req *ctrlv1.DeleteCiliumNetworkPolicy) error {
	logger.Info("deleting cilium network policy",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
	)

	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumnetworkpolicies",
	}

	err := c.dynamicClient.Resource(gvr).Namespace(req.GetK8SNamespace()).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete cilium network policy: %w", err)
	}

	return client.IgnoreNotFound(err)
}
