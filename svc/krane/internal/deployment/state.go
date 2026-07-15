package deployment

import (
	"context"
	"fmt"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildDeploymentStatus queries the pods belonging to a ReplicaSet and builds a
// status report for the control plane.
//
// The report includes each pod's cluster-local DNS address, CPU and memory limits,
// and health status. Pods without an IP address are excluded since they can't
// receive traffic yet. The address format is "{ip-with-dashes}.{namespace}.pod.cluster.local:{port}"
// which enables in-cluster DNS resolution without a headless Service.
//
// Pod phase is mapped to instance status: Running pods with ContainersReady=True
// become STATUS_RUNNING, Pending pods and Running pods whose ContainersReady
// condition is missing or False become STATUS_PENDING, and Failed pods become
// STATUS_FAILED.
func (c *Controller) buildDeploymentStatus(ctx context.Context, replicaset *appsv1.ReplicaSet) (*ctrlv1.ReportDeploymentStatusRequest, error) {
	selector, err := metav1.LabelSelectorAsSelector(replicaset.Spec.Selector)
	if err != nil {
		return nil, err
	}

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
		if pod.Status.PodIP == "" {
			continue
		}

		instance := &ctrlv1.ReportDeploymentStatusRequest_Update_Instance{
			K8SName:       pod.GetName(),
			Address:       fmt.Sprintf("%s.%s.pod.cluster.local:%d", strings.ReplaceAll(pod.Status.PodIP, ".", "-"), pod.Namespace, containerPort),
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
		case corev1.PodFailed:
			instance.Status = ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_FAILED
		case corev1.PodSucceeded, corev1.PodUnknown:
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
