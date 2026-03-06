package deployment

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteDeployment removes a user workload's ReplicaSet and associated resources
// (Secret, ServiceAccount, Role, RoleBinding) from the cluster.
//
// Not-found errors are ignored since the desired end state (resource gone) is
// already achieved. After deletion, the method reports the deletion to the control
// plane so it can update routing tables and stop sending traffic to this deployment.
//
// The method is idempotent: calling it multiple times for the same deployment
// succeeds without error.
func (c *Controller) DeleteDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) error {
	logger.Info("deleting deployment",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
	)

	// Get deployment ID from ReplicaSet labels before deleting it
	var deploymentID string
	rs, err := c.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace()).Get(ctx, req.GetK8SName(), metav1.GetOptions{})
	if err == nil {
		deploymentID, _ = labels.GetDeploymentID(rs.Labels)
	}

	err = c.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace()).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Clean up associated Secret, ServiceAccount, Role, RoleBinding
	if deploymentID != "" {
		c.cleanupDeploymentResources(ctx, req.GetK8SNamespace(), deploymentID)
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

	return nil
}
