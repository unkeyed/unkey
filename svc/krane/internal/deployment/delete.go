package deployment

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteDeployment removes a user workload's ReplicaSet from the cluster.
// Owned resources (Secret, ServiceAccount, Role, RoleBinding) are garbage-collected
// automatically by K8s via ownerReferences.
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

	if req.GetPermanent() {
		if err := assert.NotEmpty(req.GetDeploymentId(), "deployment ID is required for permanent removal"); err != nil {
			return err
		}
	} else if err := assert.NotEmpty(req.GetK8SName(), "Kubernetes name is required"); err != nil {
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	if req.GetPermanent() {
		c.trackRemoval(req)
		deleteOptions.PropagationPolicy = new(metav1.DeletePropagationForeground)
	}
	if req.GetK8SName() != "" {
		if err := c.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace()).Delete(ctx, req.GetK8SName(), deleteOptions); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		c.fingerprints.Remove(ctx, req.GetK8SName())
	}

	if req.GetPermanent() {
		removed, err := c.removePermanentDeployment(ctx, req)
		if err != nil || !removed {
			return err
		}
	}

	err := c.reportDeploymentStatus(ctx, &ctrlv1.ReportDeploymentStatusRequest{
		Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{
			Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{
				K8SName:          req.GetK8SName(),
				DeploymentId:     req.GetDeploymentId(),
				RemovalConfirmed: req.GetPermanent(),
			},
		},
	})
	if err != nil {
		return err
	}

	c.forgetRemoval(req.GetDeploymentId())
	return nil
}

func (c *Controller) removePermanentDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) (bool, error) {
	selector := labels.New().DeploymentID(req.GetDeploymentId()).ToString()
	replicaSets, err := c.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace()).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return false, fmt.Errorf("failed to list deployment replicasets: %w", err)
	}
	for i := range replicaSets.Items {
		if err := c.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace()).Delete(ctx, replicaSets.Items[i].Name, metav1.DeleteOptions{
			PropagationPolicy: new(metav1.DeletePropagationForeground),
		}); err != nil && !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("failed to delete deployment replicaset: %w", err)
		}
	}
	pods, err := c.clientSet.CoreV1().Pods(req.GetK8SNamespace()).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return false, fmt.Errorf("failed to list deployment pods: %w", err)
	}
	for i := range pods.Items {
		if err := c.clientSet.CoreV1().Pods(req.GetK8SNamespace()).Delete(ctx, pods.Items[i].Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("failed to delete deployment pod %q: %w", pods.Items[i].Name, err)
		}
	}
	if len(pods.Items) > 0 || len(replicaSets.Items) > 0 {
		return false, nil
	}

	if req.GetK8SName() != "" {
		if _, err := c.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace()).Get(ctx, req.GetK8SName(), metav1.GetOptions{}); err == nil {
			return false, nil
		} else if !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("failed to observe replicaset removal: %w", err)
		}
	}

	pods, err = c.clientSet.CoreV1().Pods(req.GetK8SNamespace()).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return false, fmt.Errorf("failed to observe deployment pod removal: %w", err)
	}
	return len(pods.Items) == 0, nil
}
