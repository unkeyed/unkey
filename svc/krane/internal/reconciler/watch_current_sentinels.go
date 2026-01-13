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
func (r *Reconciler) watchCurrentSentinels(ctx context.Context) error {

	w, err := r.clientSet.AppsV1().Deployments("").Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentSentinel().
			ToString(),
	})
	if err != nil {
		return err
	}

	go func() {
		for event := range w.ResultChan() {
			switch event.Type {
			case watch.Error:
				r.logger.Error("error watching sentinel", "event", event.Object)
				continue
			case watch.Bookmark:
				continue
			}

			sentinel, ok := event.Object.(*appsv1.Deployment)
			if !ok {
				r.logger.Error("unable to cast object to deployment")
				continue
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				r.logger.Info("sentinel added/modified", "name", sentinel.Name)
				err := r.updateSentinelState(ctx, &ctrlv1.UpdateSentinelStateRequest{
					K8SName:           sentinel.Name,
					AvailableReplicas: sentinel.Status.AvailableReplicas,
				})
				if err != nil {
					r.logger.Error("error updating sentinel state", "error", err.Error())
				}
			case watch.Deleted:
				r.logger.Info("sentinel deleted", "name", sentinel.Name)
				err := r.updateSentinelState(ctx, &ctrlv1.UpdateSentinelStateRequest{
					K8SName:           sentinel.Name,
					AvailableReplicas: 0,
				})
				if err != nil {
					r.logger.Error("error updating sentinel state", "error", err.Error())
				}
			}
		}
	}()

	return nil

}
