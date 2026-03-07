package deployment

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// runActualStateReportLoop starts a Kubernetes watch for Deployments
// and reports actual state changes back to the control plane in real-time.
//
// The watch filters for resources with the "managed-by: krane" and "component: deployment"
// labels, ignoring resources created by other controllers. When a Deployment is added
// or modified, the method calls [Controller.buildDeploymentStatus] to query pod state
// and reports via [Controller.reportDeploymentStatus]. Deletions are reported directly
// so the control plane can remove the deployment from its routing tables.
//
// The method returns an error if the initial watch setup fails. Once started, watch
// errors are logged but the goroutine continues processing events. The watch runs
// until the context is cancelled.
func (c *Controller) runActualStateReportLoop(ctx context.Context) error {
	w, err := c.clientSet.AppsV1().Deployments("").Watch(ctx, metav1.ListOptions{
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
				logger.Error("error watching deployment", "event", event.Object)
			case watch.Bookmark:
			case watch.Added, watch.Modified:
				dep, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					logger.Error("unable to cast object to deployment")
					continue
				}
				status, err := c.buildDeploymentStatus(ctx, dep)
				if err != nil {
					logger.Error("unable to build status", "error", err.Error())
					continue
				}
				err = c.reportDeploymentStatus(ctx, status)
				if err != nil {
					logger.Error("unable to report status", "error", err.Error())
					continue
				}
			case watch.Deleted:
				dep, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					logger.Error("unable to cast object to deployment")
					continue
				}
				err := c.reportDeploymentStatus(ctx, &ctrlv1.ReportDeploymentStatusRequest{
					Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{
						Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{
							K8SName: dep.Name,
						},
					},
				})
				if err != nil {
					logger.Error("unable to report status", "error", err.Error())
					continue
				}
			}
		}
	}()

	return nil
}
