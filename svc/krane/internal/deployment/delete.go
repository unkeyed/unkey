package deployment

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteDeployment removes a user workload's ReplicaSet from the cluster.
//
// Not-found errors are ignored since the desired end state (resource gone) is
// already achieved. After deletion, the method notifies the control plane so it
// can update routing tables and stop sending traffic to this deployment.
func (c *Controller) DeleteDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) error {
	c.logger.Info("deleting deployment",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
	)

	err := c.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace()).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = c.reportDeploymentStatus(ctx, &ctrlv1.ReportDeploymentStatusRequest{
		Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{
			Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{
				K8SName: req.GetK8SName(),
			},
		},
	})
	if err != nil {
		return err
	}

	return client.IgnoreNotFound(err)
}
