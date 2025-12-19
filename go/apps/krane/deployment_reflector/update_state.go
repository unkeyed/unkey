package deploymentreflector

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func (r *Reflector) updateState(ctx context.Context, state *ctrlv1.UpdateDeploymentStateRequest) error {

	_, err := r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
		return r.cluster.UpdateDeploymentState(ctx, connect.NewRequest(state))
	})
	if err != nil {
		return fmt.Errorf("failed to update deployment state: %w", err)
	}
	return nil
}

func (r *Reflector) getState(ctx context.Context, replicaset *appsv1.ReplicaSet) (*ctrlv1.UpdateDeploymentStateRequest, error) {

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
		Instances: make([]*ctrlv1.UpdateDeploymentStateRequest_Update_Instance, len(pods.Items)),
	}

	for i, pod := range pods.Items {

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
			instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_RUNNING

		case corev1.PodSucceeded, corev1.PodUnknown:
			instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED
		case corev1.PodFailed:
			instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_FAILED
		default:
			instance.Status = ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED
		}

		update.Instances[i] = instance
	}

	return &ctrlv1.UpdateDeploymentStateRequest{
		Change: &ctrlv1.UpdateDeploymentStateRequest_Update_{
			Update: update,
		},
	}, nil
}
