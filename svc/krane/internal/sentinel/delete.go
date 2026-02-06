package sentinel

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteSentinel removes a sentinel's Service and Deployment from the cluster.
//
// Both resources are deleted explicitly rather than relying on owner reference
// cascading, ensuring cleanup completes even if ownership wasn't set correctly.
// Not-found errors are ignored since the desired end state is already achieved.
func (c *Controller) DeleteSentinel(ctx context.Context, req *ctrlv1.DeleteSentinel) error {
	logger.Info("deleting sentinel",
		"namespace", NamespaceSentinel,
		"name", req.GetK8SName(),
	)

	err := c.clientSet.CoreV1().Services(NamespaceSentinel).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = c.clientSet.AppsV1().Deployments(NamespaceSentinel).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = c.reportSentinelStatus(ctx, &ctrlv1.ReportSentinelStatusRequest{
		K8SName:           req.GetK8SName(),
		AvailableReplicas: 0,
		Health:            ctrlv1.Health_HEALTH_UNHEALTHY,
	})
	if err != nil {
		return err
	}

	return client.IgnoreNotFound(err)
}
