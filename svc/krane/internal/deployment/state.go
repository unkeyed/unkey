package deployment

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildDeploymentStatus queries the pods belonging to a ReplicaSet and builds a
// status report for the control plane.
//
// Pod phase is mapped to instance status: Running pods with ContainersReady=True
// become STATUS_RUNNING, Pending pods and Running pods whose ContainersReady
// condition is missing or False become STATUS_PENDING.
func (c *Controller) buildDeploymentStatus(ctx context.Context, replicaset *appsv1.ReplicaSet) (*ctrlv1.ReportDeploymentStatusRequest, error) {
	selector, err := metav1.LabelSelectorAsSelector(replicaset.Spec.Selector)
	if err != nil {
		return nil, err
	}

	observedAtUnixNano := time.Now().UnixNano()
	pods, err := c.clientSet.CoreV1().Pods(replicaset.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Read the port from the ReplicaSet's container spec
	containerPort := int32(8080)
	if containers := replicaset.Spec.Template.Spec.Containers; len(containers) > 0 {
		if ports := containers[0].Ports; len(ports) > 0 {
			containerPort = ports[0].ContainerPort
		}
	}

	update := &ctrlv1.ReportDeploymentStatusRequest_Update{
		K8SName:   replicaset.Name,
		Instances: make([]*ctrlv1.ReportDeploymentStatusRequest_Update_Instance, 0, len(pods.Items)),
	}

	for _, pod := range pods.Items {
		instance := &ctrlv1.ReportDeploymentStatusRequest_Update_Instance{
			K8SName:              pod.GetName(),
			Address:              "",
			CpuMillicores:        0,
			MemoryMib:            0,
			Status:               ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_UNSPECIFIED,
			ContainerObservation: observeContainerStatus(&pod, observedAtUnixNano),
		}
		if pod.Status.PodIP != "" {
			instance.Address = net.JoinHostPort(pod.Status.PodIP, strconv.Itoa(int(containerPort)))
		}
		if containers := pod.Spec.Containers; len(containers) > 0 {
			if limits := containers[0].Resources.Limits; limits != nil {
				instance.CpuMillicores = limits.Cpu().MilliValue()
				instance.MemoryMib = limits.Memory().Value() / (1024 * 1024)
			}
		}

		switch pod.Status.Phase {
		case corev1.PodPending:
			instance.Status = ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING
		case corev1.PodRunning:
			// Require an explicit ContainersReady=True; a missing condition
			// means kubelet has not published readiness yet, which is PENDING
			// not RUNNING. A False condition is startup-in-progress, not a
			// permanent failure.
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.ContainersReady && cond.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			if ready {
				instance.Status = ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING
			} else {
				instance.Status = ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING
			}
		case corev1.PodFailed, corev1.PodSucceeded:
			continue
		case corev1.PodUnknown:
			instance.Status = ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_UNSPECIFIED
		}

		update.Instances = append(update.Instances, instance)
	}

	return &ctrlv1.ReportDeploymentStatusRequest{
		Change: &ctrlv1.ReportDeploymentStatusRequest_Update_{
			Update: update,
		},
	}, nil
}

func observeContainerStatus(pod *corev1.Pod, observedAtUnixNano int64) *ctrlv1.ContainerObservation {
	var observation *ctrlv1.ContainerObservation
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != "deployment" || (status.State.Running == nil && status.State.Waiting == nil && status.State.Terminated == nil) {
			continue
		}
		observation = &ctrlv1.ContainerObservation{
			RestartCount:       status.RestartCount,
			ObservedAtUnixNano: observedAtUnixNano,
		}
		if waiting := status.State.Waiting; isActionableWaiting(waiting) {
			observation.Waiting = &ctrlv1.Waiting{Reason: waiting.Reason, Message: waiting.Message}
		}
		for _, exit := range []*corev1.ContainerStateTerminated{status.LastTerminationState.Terminated, status.State.Terminated} {
			if exit == nil || exit.ExitCode == 0 || exit.FinishedAt.IsZero() || exit.FinishedAt.UnixMilli() <= observation.LastFailureFinishedAt {
				continue
			}
			observation.LastFailure = &ctrlv1.Terminated{ExitCode: exit.ExitCode, Signal: exit.Signal, Reason: exit.Reason, Message: exit.Message}
			observation.LastFailureFinishedAt = exit.FinishedAt.UnixMilli()
		}
		break
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse && condition.Reason == corev1.PodReasonUnschedulable {
			if observation == nil {
				observation = &ctrlv1.ContainerObservation{ObservedAtUnixNano: observedAtUnixNano}
			}
			observation.Waiting = &ctrlv1.Waiting{Reason: condition.Reason, Message: condition.Message}
			break
		}
	}
	return observation
}
