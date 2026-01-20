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
// status report containing each pod's address, resource allocation, and phase.
// Pods without an IP address are skipped since they can't receive traffic yet.
// The address is formatted as a cluster-local DNS name for in-cluster routing.
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
			Address:       fmt.Sprintf("%s.%s.pod.cluster.local:%d", strings.ReplaceAll(pod.Status.PodIP, ".", "-"), pod.Namespace, DeploymentPort),
			CpuMillicores: 0,
			MemoryMib:     0,
			Status:        ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_UNSPECIFIED,
		}
		if pod.Spec.Resources != nil {
			instance.CpuMillicores = pod.Spec.Resources.Limits.Cpu().MilliValue()
			instance.MemoryMib = pod.Spec.Resources.Limits.Memory().Value() / (1024 * 1024)
		}

		switch pod.Status.Phase {
		case corev1.PodPending:
			instance.Status = ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING
		case corev1.PodRunning:
			allReady := true
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.ContainersReady && cond.Status != corev1.ConditionTrue {
					allReady = false
					break
				}
			}
			if allReady {
				instance.Status = ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING
			} else {
				instance.Status = ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_FAILED
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
