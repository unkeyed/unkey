package reconciler

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// watchCurrentDeployments starts a Kubernetes watch for deployment ReplicaSets and
// reports state changes back to the control plane in real-time.
//
// The watch filters for resources with the "managed-by: krane" and "component: deployment"
// labels, ignoring resources created by other controllers. When a ReplicaSet is added,
// modified, or deleted, the method queries pod status and pushes an update to the
// control plane so routing tables stay synchronized with actual cluster state.
//
// This complements [Reconciler.refreshCurrentDeployments] which handles consistency
// for events that might be missed during network partitions or restarts.
func (r *Reconciler) watchCurrentDeployments(ctx context.Context) error {

	w, err := r.clientSet.AppsV1().ReplicaSets("").Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentDeployment().
			ToString(),
	})
	if err != nil {
		return err
	}
	go func() {

		for event := range w.ResultChan() {
			replicaset, ok := event.Object.(*appsv1.ReplicaSet)
			if !ok {
				r.logger.Error("unable to cast object to deployment", "error", err.Error())
				continue
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				state, err := r.getDeploymentState(ctx, replicaset)
				if err != nil {
					r.logger.Error("unable to get state", "error", err.Error())
					continue
				}
				err = r.updateDeploymentState(ctx, state)
				if err != nil {
					r.logger.Error("unable to update state", "error", err.Error())
					continue
				}
			case watch.Deleted:
				err = r.updateDeploymentState(ctx, &ctrlv1.UpdateDeploymentStateRequest{
					Change: &ctrlv1.UpdateDeploymentStateRequest_Delete_{
						Delete: &ctrlv1.UpdateDeploymentStateRequest_Delete{
							K8SName: replicaset.Name,
						},
					},
				})
				if err != nil {
					r.logger.Error("unable to update state", "error", err.Error())
					continue
				}
			case watch.Bookmark:
			case watch.Error:
				r.logger.Error("error watching deployment", "error", err.Error())
			}
		}
	}()
	return nil
}
