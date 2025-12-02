package gatewaycontroller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

func (c *GatewayController) watch() error {

	c.logger.Info("Starting gateway watch")
	ctx := context.Background()
	w, err := c.clientset.AppsV1().Deployments(k8s.UntrustedNamespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(k8s.NewLabels().
			ManagedByKrane().
			ComponentGateway().
			ToMap()),
		Watch:           true,
		ResourceVersion: "", // send synthetic events of existing cluster state, then watch
	})
	if err != nil {
		return err
	}

	go func() {

		for e := range w.ResultChan() {

			switch e.Type {
			case watch.Added:
				deployment, ok := e.Object.(*appsv1.Deployment)
				if !ok {
					c.logger.Warn("skipping non-deployment event", "obj", e.Object)
					continue
				}

				gatewayID, ok := k8s.GetGatewayID(deployment.GetLabels())
				if !ok {
					c.logger.Error("deployment missing gateway id", "name", deployment.Name)
					continue
				}

				c.buffer.Buffer(&ctrlv1.UpdateGatewayRequest{
					Change: &ctrlv1.UpdateGatewayRequest_Create_{
						Create: &ctrlv1.UpdateGatewayRequest_Create{
							GatewayId:       gatewayID,
							RunningReplicas: deployment.Status.ReadyReplicas,
						},
					},
				})

			case watch.Modified:
				deployment, ok := e.Object.(*appsv1.Deployment)
				if !ok {
					c.logger.Warn("skipping non-deployment event", "obj", e.Object)
					continue
				}

				gatewayID, ok := k8s.GetGatewayID(deployment.GetLabels())
				if !ok {
					c.logger.Error("deployment missing gateway id", "name", deployment.Name)
					continue
				}
				c.buffer.Buffer(&ctrlv1.UpdateGatewayRequest{
					Change: &ctrlv1.UpdateGatewayRequest_Update_{
						Update: &ctrlv1.UpdateGatewayRequest_Update{
							GatewayId:       gatewayID,
							RunningReplicas: deployment.Status.ReadyReplicas,
						},
					},
				})

			case watch.Deleted:
				deployment, ok := e.Object.(*appsv1.Deployment)
				if !ok {
					c.logger.Warn("skipping non-deployment event", "obj", e.Object)
					continue
				}

				gatewayID, ok := k8s.GetGatewayID(deployment.GetLabels())
				if !ok {
					c.logger.Error("deployment missing gateway id", "name", deployment.Name)
					continue
				}

				c.buffer.Buffer(&ctrlv1.UpdateGatewayRequest{
					Change: &ctrlv1.UpdateGatewayRequest_Delete_{
						Delete: &ctrlv1.UpdateGatewayRequest_Delete{
							GatewayId: gatewayID,
						},
					},
				})

			case watch.Error:

				c.logger.Error("watch error", "obj", e.Object)

			case watch.Bookmark:
			}

		}
	}()

	return nil
}
