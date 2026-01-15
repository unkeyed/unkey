package reconciler

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// updateDeploymentState pushes deployment state to the control plane through the circuit
// breaker. The circuit breaker prevents cascading failures during control plane outages
// by failing fast after repeated errors rather than blocking all reconciliation.
func (r *Reconciler) updateDeploymentState(ctx context.Context, state *ctrlv1.UpdateDeploymentStateRequest) error {
	_, err := r.cb.Do(ctx, func(innerCtx context.Context) (any, error) {
		return r.cluster.UpdateDeploymentState(innerCtx, connect.NewRequest(state))
	})
	if err != nil {
		return fmt.Errorf("failed to update deployment state: %w", err)
	}
	return nil
}

// updateSentinelState pushes sentinel state to the control plane through the circuit breaker.
func (r *Reconciler) updateSentinelState(ctx context.Context, state *ctrlv1.UpdateSentinelStateRequest) error {
	_, err := r.cb.Do(ctx, func(innerCtx context.Context) (any, error) {
		return r.cluster.UpdateSentinelState(innerCtx, connect.NewRequest(state))
	})
	if err != nil {
		return fmt.Errorf("failed to update sentinel state: %w", err)
	}
	return nil
}

// getDeploymentState queries the pods belonging to a ReplicaSet and builds a state
// update request containing each pod's address, resource allocation, and phase.
// Pods without an IP address are skipped since they can't receive traffic yet.
// The address is formatted as a cluster-local DNS name for in-cluster routing.
func (r *Reconciler) getDeploymentState(ctx context.Context, replicaset *appsv1.ReplicaSet) (*ctrlv1.UpdateDeploymentStateRequest, error) {
	selector, err := metav1.LabelSelectorAsSelector(replicaset.Spec.Selector)
	if err != nil {
		return nil, err
	}

	pods, err := r.clientSet.CoreV1().Pods(replicaset.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	update := &ctrlv1.UpdateDeploymentStateRequest_Update{
		K8SName:   replicaset.Name,
		Instances: make([]*ctrlv1.UpdateDeploymentStateRequest_Update_Instance, 0, len(pods.Items)),
	}

	for _, pod := range pods.Items {
		if pod.Status.PodIP == "" {
			continue
		}

		instance := &ctrlv1.UpdateDeploymentStateRequest_Update_Instance{
			K8SName:       pod.GetName(),
			Address:       fmt.Sprintf("%s.%s.pod.cluster.local", strings.ReplaceAll(pod.Status.PodIP, ".", "-"), pod.Namespace),
			CpuMillicores: 0,
			MemoryMib:     0,
			Status:        ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED,
		}
		if pod.Spec.Resources != nil {
			instance.CpuMillicores = pod.Spec.Resources.Limits.Cpu().MilliValue()
			instance.MemoryMib = pod.Spec.Resources.Limits.Memory().Value() / (1024 * 1024)
		}

		switch pod.Status.Phase {
		case corev1.PodPending:
			instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_PENDING
		case corev1.PodRunning:
			// Check if all containers are ready to determine running vs failed
			allReady := true
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.ContainersReady && cond.Status != corev1.ConditionTrue {
					allReady = false
					break
				}
			}
			if allReady {
				instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_RUNNING
			} else {
				instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_FAILED
			}
		case corev1.PodFailed:
			instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_FAILED
		default:
			instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED
		}

		update.Instances = append(update.Instances, instance)
	}

	return &ctrlv1.UpdateDeploymentStateRequest{
		Change: &ctrlv1.UpdateDeploymentStateRequest_Update_{
			Update: update,
		},
	}, nil
}
