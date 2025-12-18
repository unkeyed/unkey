package deploymentreflector

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
)

func (r *Reflector) updateState(ctx context.Context, req types.NamespacedName) error {

	replicaset, err := r.clientSet.AppsV1().ReplicaSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {

			_, err = r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
				return r.cluster.UpdateDeploymentState(ctx, connect.NewRequest(&ctrlv1.UpdateDeploymentStateRequest{
					Change: &ctrlv1.UpdateDeploymentStateRequest_Delete_{
						Delete: &ctrlv1.UpdateDeploymentStateRequest_Delete{
							K8SName: req.Name,
						},
					},
				}))
			})
			if err != nil {
				return err
			}

			// nolint:exhaustruct
			return nil
		}
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(replicaset.Spec.Selector)
	if err != nil {
		return err
	}

	pods, err := r.clientSet.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return err
	}

	update := &ctrlv1.UpdateDeploymentStateRequest_Update{
		K8SName:   req.Name,
		Instances: make([]*ctrlv1.UpdateDeploymentStateRequest_Update_Instance, len(pods.Items)),
	}

	for i, pod := range pods.Items {
		instance := &ctrlv1.UpdateDeploymentStateRequest_Update_Instance{
			K8SName:       pod.GetName(),
			Address:       fmt.Sprintf("%s.%s.pod.cluster.local", strings.ReplaceAll(pod.Status.PodIP, ".", "-"), pod.Namespace),
			CpuMillicores: pod.Spec.Resources.Limits.Cpu().MilliValue(),
			MemoryMib:     pod.Spec.Resources.Limits.Memory().Value() / (1024 * 1024),
			Status:        ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED,
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

	_, err = r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
		return r.cluster.UpdateDeploymentState(ctx, connect.NewRequest(&ctrlv1.UpdateDeploymentStateRequest{
			Change: &ctrlv1.UpdateDeploymentStateRequest_Update_{
				Update: update,
			},
		}))
	})

	if err != nil {
		return err
	}
	return nil
}
