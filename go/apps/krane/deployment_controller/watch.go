package deploymentcontroller

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

func (c *DeploymentController) watch() error {

	c.logger.Info("Starting statefulset watch")
	ctx := context.Background()

	w, err := c.clientset.CoreV1().Pods("untrusted").Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(map[string]string{
			k8s.LabelManagedBy: "krane",
			k8s.LavelComponent: "deployment",
		}),
		Watch: true,
	})
	if err != nil {
		return err
	}

	go func() {

		for e := range w.ResultChan() {

			switch e.Type {
			case watch.Added, watch.Modified:
				pod := e.Object.(*corev1.Pod)

				c.logger.Info("Pod modified",
					"name", pod.Name,
					"status", pod.Status.Phase,
				)
			case watch.Deleted:
				c.logger.Info("Pod deleted", "pod", e.Object.(*corev1.Pod).Name)
			case watch.Error:
				c.logger.Error("Pod watch error", "error", e.Object.(*corev1.Pod).Name)
			}

			req := &ctrlv1.UpdateDeploymentStatusRequest{
				Region:    "",
				Instances: []*ctrlv1.UpdateDeploymentStatusRequest_Instance{},
			}

			c.buffer.Buffer(req)
		}
	}()

	return nil
}
