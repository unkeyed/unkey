package k8s

import (
	"context"

	controllerruntime "sigs.k8s.io/controller-runtime"
)

type Reconciler interface {
	Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error)
}
