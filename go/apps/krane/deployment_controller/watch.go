package deploymentcontroller

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

func (c *DeploymentController) watch() error {

	c.logger.Info("Starting cluster-wide pod watch for deployments")
	ctx := context.Background()

	// Watch ALL namespaces (empty string = cluster-wide)
	w, err := c.clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(k8s.NewLabels().
			ManagedByKrane().
			ToMap()),
		Watch:           true,
		ResourceVersion: "", // send synthetic events of existing cluster state, then watch
	})
	if err != nil {
		return err
	}

	go func() {

		for e := range w.ResultChan() {
			c.logger.Debug("pod event", "event", e.Type)

			switch e.Type {
			case watch.Added:
				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					c.logger.Warn("skipping non-pod event", "obj", e.Object)
					continue
				}

				// Check if this pod belongs to an UnkeyDeployment
				deploymentID, ok := k8s.GetDeploymentID(pod.GetLabels())
				if !ok {
					c.logger.Debug("skipping pod without deployment ID", "name", pod.Name)
					continue
				}

				// Also check it has all the labels we expect for UnkeyDeployments
				if _, hasWorkspace := k8s.GetWorkspaceID(pod.GetLabels()); !hasWorkspace {
					c.logger.Debug("skipping non-UnkeyDeployment pod (no workspace)", "name", pod.Name)
					continue
				}

				cpuMillicores := int64(0)
				memoryMib := int64(0)
				for _, container := range pod.Spec.Containers {
					if container.Name == "container" {
						cpuMillicores = container.Resources.Limits.Cpu().MilliValue()
						memoryMib = container.Resources.Limits.Memory().Value() / 1024 / 1024
					}
				}

				// Send direct pod IP instead of DNS name
				address := pod.Status.PodIP
				if address == "" {
					// Pod might not have an IP yet if it's just been created
					c.logger.Debug("pod has no IP yet", "name", pod.Name)
					address = "" // Will be updated when Modified event comes
				}

				c.updates.Buffer(&ctrlv1.UpdateInstanceRequest{
					Change: &ctrlv1.UpdateInstanceRequest_Create_{
						Create: &ctrlv1.UpdateInstanceRequest_Create{
							DeploymentId:  deploymentID,
							PodName:       pod.Name,
							Address:       address, // Direct pod IP
							CpuMillicores: int32(cpuMillicores),
							MemoryMib:     int32(memoryMib),
							Status:        k8sStatusToProto(pod.Status),
						},
					},
				})
				c.logger.Debug("reported pod creation",
					"pod", pod.Name,
					"namespace", pod.Namespace,
					"deployment_id", deploymentID,
					"ip", address,
				)

			case watch.Modified:
				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					c.logger.Warn("skipping non-pod event", "obj", e.Object)
					continue
				}

				deploymentID, ok := k8s.GetDeploymentID(pod.GetLabels())
				if !ok {
					c.logger.Debug("skipping pod without deployment ID", "name", pod.Name)
					continue
				}

				// Also check it has all the labels we expect for UnkeyDeployments
				if _, hasWorkspace := k8s.GetWorkspaceID(pod.GetLabels()); !hasWorkspace {
					c.logger.Debug("skipping non-UnkeyDeployment pod (no workspace)", "name", pod.Name)
					continue
				}

				// For modified events, we need to check if the IP changed
				// The reconciler handles most updates, but pod IP changes need immediate reporting
				address := pod.Status.PodIP

				c.updates.Buffer(&ctrlv1.UpdateInstanceRequest{
					Change: &ctrlv1.UpdateInstanceRequest_Update_{
						Update: &ctrlv1.UpdateInstanceRequest_Update{
							DeploymentId: deploymentID,
							PodName:      pod.Name,
							Status:       k8sStatusToProto(pod.Status),
							// Note: The proto doesn't have an address field for updates
							// If we need to update the IP, we might need to modify the proto
							// or use Create with the new IP
						},
					},
				})
				c.logger.Debug("reported pod update",
					"pod", pod.Name,
					"namespace", pod.Namespace,
					"deployment_id", deploymentID,
					"ip", address,
					"status", pod.Status.Phase,
				)

			case watch.Deleted:
				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					c.logger.Warn("skipping non-pod event", "obj", e.Object)
					continue
				}

				deploymentID, ok := k8s.GetDeploymentID(pod.GetLabels())
				if !ok {
					c.logger.Debug("skipping pod without deployment ID", "name", pod.Name)
					continue
				}

				// Also check it has all the labels we expect for UnkeyDeployments
				if _, hasWorkspace := k8s.GetWorkspaceID(pod.GetLabels()); !hasWorkspace {
					c.logger.Debug("skipping non-UnkeyDeployment pod (no workspace)", "name", pod.Name)
					continue
				}

				c.updates.Buffer(&ctrlv1.UpdateInstanceRequest{
					Change: &ctrlv1.UpdateInstanceRequest_Delete_{
						Delete: &ctrlv1.UpdateInstanceRequest_Delete{
							DeploymentId: deploymentID,
							PodName:      pod.Name,
						},
					},
				})
				c.logger.Debug("reported pod deletion",
					"pod", pod.Name,
					"namespace", pod.Namespace,
					"deployment_id", deploymentID,
				)

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
