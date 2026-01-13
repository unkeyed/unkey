package reconciler

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// refreshCurrentdeployments performs periodic synchronization of all deployment resources.
//
// This function runs every minute to ensure all deployment resources in the
// cluster are synchronized with their desired state from the control plane.
// This periodic refresh provides consistency guarantees despite possible
// missed events, network partitions, or controller restarts.
//
// The function:
//  1. Lists all deployment resources managed by krane across all namespaces
//  2. Queries control plane for the desired state of each deployment
//  3. Buffers the desired state events for processing
//
// This approach ensures eventual consistency between the database state
// and Kubernetes cluster state, acting as a safety net for the event-based
// synchronization mechanism.
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
			switch event.Type {
			case watch.Error:
				r.logger.Error("error watching deployment", "event", event.Object)
				continue
			case watch.Bookmark:
				continue
			case watch.Added, watch.Modified, watch.Deleted:
			}

			replicaset, ok := event.Object.(*appsv1.ReplicaSet)
			if !ok {
				r.logger.Error("unable to cast object to replicaset")
				continue
			}

			switch event.Type {
			case watch.Bookmark, watch.Error:
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
				err := r.updateDeploymentState(ctx, &ctrlv1.UpdateDeploymentStateRequest{
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
			}
		}
	}()
	return nil
}
