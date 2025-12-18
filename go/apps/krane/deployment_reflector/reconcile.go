package deploymentreflector

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reflector) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	err := r.updateState(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Minute}, nil
}
