package deployment

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteDeployment removes a user workload from the cluster: its Deployment
// and any legacy ReplicaSet with the same name, plus its environment Secret.
// Other owned resources (ReplicaSets, pods, ServiceAccount, HPA, PDB, CiliumNetworkPolicy) are
// garbage-collected automatically by K8s via ownerReferences.
//
// Not-found errors are ignored since the desired end state (resource gone) is
// already achieved. After deletion, the method reports the deletion to the control
// plane so it can update routing tables and stop sending traffic to this deployment.
func (c *Controller) DeleteDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) (retErr error) {
	defer func() { metrics.RecordReconcile("deployment", "delete", retErr) }()
	logger.Info("deleting deployment",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
	)

	apps := c.clientSet.AppsV1()
	d, err := apps.Deployments(req.GetK8SNamespace()).Get(ctx, req.GetK8SName(), metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if d != nil {
		if id, ok := labels.GetDeploymentID(d.Labels); ok && id != "" {
			if err := c.deleteDeploymentSecret(ctx, req.GetK8SNamespace(), id); err != nil {
				return err
			}
		}
	}
	rs, err := apps.ReplicaSets(req.GetK8SNamespace()).Get(ctx, req.GetK8SName(), metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if rs != nil {
		if id, ok := labels.GetDeploymentID(rs.Labels); ok && id != "" {
			if err := c.deleteDeploymentSecret(ctx, req.GetK8SNamespace(), id); err != nil {
				return err
			}
		}
	}
	err = apps.Deployments(req.GetK8SNamespace()).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = apps.ReplicaSets(req.GetK8SNamespace()).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	c.fingerprints.Remove(ctx, req.GetK8SName())

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
