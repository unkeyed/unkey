package deploymentcontroller

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type podWatcher struct {
	client  client.Client
	changes *buffer.Buffer[*ctrlv1.UpdateInstanceRequest]
}

func (pw *podWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	pod := &corev1.Pod{}
	err := pw.client.Get(ctx, req.NamespacedName, pod)
	if err == nil {
		deploymentID, ok := k8s.GetDeploymentID(pod.GetLabels())
		if !ok {
			return ctrl.Result{}, fmt.Errorf("pod %s.%s doesn't have a deployment id label", pod.Namespace, pod.Name)
		}
		pw.changes.Buffer(&ctrlv1.UpdateInstanceRequest{
			Change: &ctrlv1.UpdateInstanceRequest_Upsert_{
				Upsert: &ctrlv1.UpdateInstanceRequest_Upsert{
					DeploymentId:  deploymentID,
					PodName:       pod.Name,
					Address:       fmt.Sprintf("%s.%s.pod.cluster.local", strings.ReplaceAll(pod.Status.PodIP, ".", "-"), pod.Namespace),
					CpuMillicores: int32(pod.Spec.Resources.Limits.Cpu().MilliValue()),
					MemoryMib:     int32(pod.Spec.Resources.Limits.Memory().Value()),
					Status:        k8sPodStatusToUnkeyPodStatus(pod.Status),
				},
			},
		})
	}
	if apierrors.IsNotFound(err) {
		pw.changes.Buffer(&ctrlv1.UpdateInstanceRequest{
			Change: &ctrlv1.UpdateInstanceRequest_Delete_{
				Delete: &ctrlv1.UpdateInstanceRequest_Delete{
					PodName: pod.Name,
				},
			},
		})
	}
	return ctrl.Result{}, err

}
