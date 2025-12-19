package deploymentreflector

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
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
func (r *Reflector) watchCurrentDeployments(ctx context.Context) {

	w, err := r.clientSet.AppsV1().ReplicaSets("").Watch(ctx, metav1.ListOptions{
		LabelSelector: k8s.NewLabels().
			ManagedByKrane().
			ComponentDeployment().
			ToSelector().String(),
	})

	defer w.Stop()

	for event := range w.ResultChan() {
		replicaset, ok := event.Object.(*appsv1.ReplicaSet)
		if !ok {
			r.logger.Error("unable to cast object to deployment", "error", err.Error())
			continue
		}

		switch event.Type {
		case watch.Added, watch.Modified:
			state, err := r.getState(ctx, replicaset)
			if err != nil {
				r.logger.Error("unable to get state", "error", err.Error())
				continue
			}
			err = r.updateState(ctx, state)
			if err != nil {
				r.logger.Error("unable to update state", "error", err.Error())
				continue
			}
		case watch.Deleted:
			err = r.updateState(ctx, &ctrlv1.UpdateDeploymentStateRequest{
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

}
