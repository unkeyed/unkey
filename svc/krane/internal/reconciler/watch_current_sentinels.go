package reconciler

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// watchCurrentSentinels starts a Kubernetes watch for sentinel Deployments and
// reports replica availability back to the control plane in real-time.
//
// The watch filters for resources with the "managed-by: krane" and "component: sentinel"
// labels. When a Deployment's available replica count changes, the method notifies
// the control plane so it knows which sentinels are ready to receive traffic.
//
// This complements [Reconciler.refreshCurrentSentinels] which handles consistency
// for events that might be missed during network partitions or restarts.
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
			case watch.Bookmark:
			case watch.Added, watch.Modified:
				sentinel, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					r.logger.Error("unable to cast object to deployment")
					continue
				}
				r.logger.Info("sentinel added/modified", "name", sentinel.Name)
				err := r.updateSentinelState(ctx, &ctrlv1.UpdateSentinelStateRequest{
					K8SName:           sentinel.Name,
					AvailableReplicas: sentinel.Status.AvailableReplicas,
				})
				if err != nil {
					r.logger.Error("error updating sentinel state", "error", err.Error())
				}
			case watch.Deleted:
				sentinel, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					r.logger.Error("unable to cast object to deployment")
					continue
				}
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
