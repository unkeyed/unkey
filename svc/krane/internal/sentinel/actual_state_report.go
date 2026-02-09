package sentinel

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// runActualStateReportLoop starts a Kubernetes watch for sentinel Deployments
// and reports actual state changes back to the control plane in real-time.
//
// The watch filters for resources with the "managed-by: krane" and "component: sentinel"
// labels. When a Deployment's available replica count changes, the method reports
// the actual state to the control plane so it knows which sentinels are ready.
func (c *Controller) runActualStateReportLoop(ctx context.Context) error {
	w, err := c.clientSet.AppsV1().Deployments(NamespaceSentinel).Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentSentinel().
			ToString(),
	})
	if err != nil {
		return err
	}

	go func() {
		for event := range w.ResultChan() {
			switch event.Type {
			case watch.Error:
				logger.Error("error watching sentinel", "event", event.Object)
			case watch.Bookmark:
			case watch.Added, watch.Modified:
				sentinel, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					logger.Error("unable to cast object to deployment")
					continue
				}
				logger.Info("sentinel added/modified", "name", sentinel.Name)

				desiredReplicas := int32(0)
				if sentinel.Spec.Replicas != nil {
					desiredReplicas = *sentinel.Spec.Replicas
				}

				var health ctrlv1.Health
				if desiredReplicas == 0 {
					health = ctrlv1.Health_HEALTH_PAUSED
				} else if sentinel.Status.AvailableReplicas > 0 {
					health = ctrlv1.Health_HEALTH_HEALTHY
				} else {
					health = ctrlv1.Health_HEALTH_UNHEALTHY
				}

				err := c.reportSentinelStatus(ctx, &ctrlv1.ReportSentinelStatusRequest{
					K8SName:           sentinel.Name,
					AvailableReplicas: sentinel.Status.AvailableReplicas,
					Health:            health,
				})
				if err != nil {
					logger.Error("error reporting sentinel status", "error", err.Error())
				}
			case watch.Deleted:
				sentinel, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					logger.Error("unable to cast object to deployment")
					continue
				}
				logger.Info("sentinel deleted", "name", sentinel.Name)
				err := c.reportSentinelStatus(ctx, &ctrlv1.ReportSentinelStatusRequest{
					K8SName:           sentinel.Name,
					AvailableReplicas: 0,
					Health:            ctrlv1.Health_HEALTH_UNHEALTHY,
				})
				if err != nil {
					logger.Error("error reporting sentinel status", "error", err.Error())
				}
			}
		}
	}()

	return nil
}
