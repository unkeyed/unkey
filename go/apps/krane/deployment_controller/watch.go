package deploymentcontroller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

func (c *DeploymentController) watch() error {

	c.logger.Info("Starting deployment watch")
	ctx := context.Background()
	w, err := c.clientset.CoreV1().Pods(k8s.UntrustedNamespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(k8s.NewLabels().
			ManagedByKrane().
			ComponentDeployment().
			ToMap()),
		Watch:           true,
		ResourceVersion: "", // send synthetic events of existing cluster state, then watch
	})
	if err != nil {
		return err
	}

	go func() {

		for e := range w.ResultChan() {
			c.logger.Info("pod event", "event", e.Type)

			switch e.Type {
			case watch.Added:
				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					c.logger.Warn("skipping non-pod event", "obj", e.Object)
					continue
				}

				deploymentID, ok := k8s.GetDeploymentID(pod.GetLabels())
				if !ok {
					c.logger.Warn("skipping non-deployment event", "name", pod.Name)
					continue
				}

				c.buffer.Buffer(&ctrlv1.UpdateInstanceRequest{
					Change: &ctrlv1.UpdateInstanceRequest_Create_{
						Create: &ctrlv1.UpdateInstanceRequest_Create{
							DeploymentId:  deploymentID,
							PodName:       pod.Name,
							Address:       fmt.Sprintf("%s-%s.untrusted.svc.cluster.local", pod.Name, "TODO"),
							CpuMillicores: 0, //int32(pod.Spec.Resources.Limits.Cpu().MilliValue()),
							MemoryMib:     0, //int32(pod.Spec.Resources.Limits.Memory().Value() / (1024 * 1024)),
							Status:        k8sStatusToProto(pod.Status),
						},
					},
				})

			case watch.Modified:
				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					c.logger.Warn("skipping non-pod event", "obj", e.Object)
					continue
				}

				deploymentID, ok := k8s.GetDeploymentID(pod.GetLabels())
				if !ok {
					c.logger.Warn("skipping non-deployment event", "name", pod.Name)
					continue
				}

				c.buffer.Buffer(&ctrlv1.UpdateInstanceRequest{
					Change: &ctrlv1.UpdateInstanceRequest_Update_{
						Update: &ctrlv1.UpdateInstanceRequest_Update{
							DeploymentId: deploymentID,
							PodName:      pod.Name,
							Status:       k8sStatusToProto(pod.Status),
						},
					},
				})

			case watch.Deleted:
				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					c.logger.Warn("skipping non-pod event", "obj", e.Object)
					continue
				}

				deploymentID, ok := k8s.GetDeploymentID(pod.GetLabels())
				if !ok {
					c.logger.Warn("skipping non-deployment event", "name", pod.Name)
					continue
				}

				c.buffer.Buffer(&ctrlv1.UpdateInstanceRequest{
					Change: &ctrlv1.UpdateInstanceRequest_Delete_{
						Delete: &ctrlv1.UpdateInstanceRequest_Delete{
							DeploymentId: deploymentID,
							PodName:      pod.Name,
						},
					},
				})

			case watch.Error:

				c.logger.Error("watch error", "obj", e.Object)

			case watch.Bookmark:
				// don't care
			}

		}
	}()

	return nil
}

func k8sStatusToProto(status corev1.PodStatus) ctrlv1.UpdateInstanceRequest_Status {
	switch status.Phase {
	case corev1.PodPending:
		return ctrlv1.UpdateInstanceRequest_STATUS_PENDING
	case corev1.PodRunning:
		return ctrlv1.UpdateInstanceRequest_STATUS_RUNNING
	case corev1.PodFailed:
		return ctrlv1.UpdateInstanceRequest_STATUS_FAILED
	case corev1.PodSucceeded, corev1.PodUnknown:
		return ctrlv1.UpdateInstanceRequest_STATUS_UNSPECIFIED
	default:
		return ctrlv1.UpdateInstanceRequest_STATUS_UNSPECIFIED
	}
}
