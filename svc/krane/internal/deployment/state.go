package deployment

import (
	"context"
	"fmt"
	"net"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildDeploymentStatus queries the pods belonging to a workload and builds a
// status report for the control plane.
//
// The report includes each active pod's IP address, CPU and memory
// limits, and health status. Pods without an IP address are excluded since they
// can't receive traffic yet. Failed and succeeded pods are also excluded because
// the ReplicaSet replaces them but Kubernetes retains their objects for garbage
// collection. Reporting them would retain a stale instance for every replacement.
// Terminating pods are excluded so a rollout stops routing to replaced pods
// before they exit. The address format is "{pod-ip}:{port}" so instances are
// reachable from peered clusters without relying on cluster-local DNS. The port
// comes from each pod, because pods of two revisions coexist during a rollout.
//
// Pod phase is mapped to instance status: Running pods with ContainersReady=True
// become STATUS_RUNNING, Pending pods and Running pods whose ContainersReady
// condition is missing or False become STATUS_PENDING.
func (c *Controller) buildDeploymentStatus(ctx context.Context, w workload) (*ctrlv1.ReportDeploymentStatusRequest, error) {
	selector, err := metav1.LabelSelectorAsSelector(w.selector)
	if err != nil {
		return nil, err
	}

	pods, err := c.clientSet.CoreV1().Pods(w.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	update := &ctrlv1.ReportDeploymentStatusRequest_Update{
		K8SName:   w.k8sName,
		Instances: make([]*ctrlv1.ReportDeploymentStatusRequest_Update_Instance, 0, len(pods.Items)),
	}

	for _, pod := range pods.Items {
		if pod.Status.PodIP == "" || pod.DeletionTimestamp != nil {
			continue
		}

		instance := &ctrlv1.ReportDeploymentStatusRequest_Update_Instance{
			K8SName:       pod.GetName(),
			Address:       net.JoinHostPort(pod.Status.PodIP, strconv.Itoa(int(podContainerPort(&pod)))),
			CpuMillicores: 0,
			MemoryMib:     0,
			Status:        ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_UNSPECIFIED,
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

func podContainerPort(pod *corev1.Pod) int32 {
	if containers := pod.Spec.Containers; len(containers) > 0 {
		if ports := containers[0].Ports; len(ports) > 0 {
			return ports[0].ContainerPort
		}
	}
	return 8080
}
