package deployment

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// runActualStateReportLoop starts a Kubernetes watch for deployment ReplicaSets
// and reports actual state changes back to the control plane in real-time.
//
// The watch filters for resources with the "managed-by: krane" and "component: deployment"
// labels, ignoring resources created by other controllers. When a ReplicaSet is added,
// modified, or deleted, the method queries pod status and reports the actual state to
// the control plane so routing tables stay synchronized with what's running in the cluster.
func (c *Controller) runActualStateReportLoop(ctx context.Context) error {
	w, err := c.clientSet.AppsV1().ReplicaSets("").Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentDeployment().
			ToString(),
	})
	if err != nil {
		return err
	}

	go func() {
		for event := range w.ResultChan() {
			switch event.Type {
			case watch.Error:
				c.logger.Error("error watching deployment", "event", event.Object)
			case watch.Bookmark:
			case watch.Added, watch.Modified:
				replicaset, ok := event.Object.(*appsv1.ReplicaSet)
				if !ok {
					c.logger.Error("unable to cast object to replicaset")
					continue
				}
				status, err := c.buildDeploymentStatus(ctx, replicaset)
				if err != nil {
					c.logger.Error("unable to build status", "error", err.Error())
					continue
				}
				err = c.reportDeploymentStatus(ctx, status)
				if err != nil {
					c.logger.Error("unable to report status", "error", err.Error())
					continue
				}
			case watch.Deleted:
				replicaset, ok := event.Object.(*appsv1.ReplicaSet)
				if !ok {
					c.logger.Error("unable to cast object to replicaset")
					continue
				}
				err := c.reportDeploymentStatus(ctx, &ctrlv1.ReportDeploymentStatusRequest{
					Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{
						Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{
							K8SName: replicaset.Name,
						},
					},
				})
				if err != nil {
					c.logger.Error("unable to report status", "error", err.Error())
					continue
				}
			}
		}
	}()

	return nil
}
