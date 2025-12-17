package inbound

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type StatusReconciler struct {
	client client.Client
	logger logging.Logger

	cluster ctrlv1connect.ClusterServiceClient
	cb      circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.UpdateSentinelStateResponse]]
}
type Config struct {
	Client  client.Client
	Logger  logging.Logger
	Cluster ctrlv1connect.ClusterServiceClient
	Manager manager.Manager
}

func New(cfg Config) (*StatusReconciler, error) {

	r := &StatusReconciler{
		client:  cfg.Client,
		logger:  cfg.Logger,
		cluster: cfg.Cluster,
		cb:      circuitbreaker.New[*connect.Response[ctrlv1.UpdateSentinelStateResponse]]("sentinel_controller_status"),
	}

	err := ctrl.NewControllerManagedBy(cfg.Manager).
		For(
			&appsv1.Deployment{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(func(obj client.Object) bool {
					return k8s.IsComponentSentinel(obj.GetLabels())
				}),
			),
		).Complete(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *StatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	dpl := &appsv1.Deployment{}
	err := r.client.Get(ctx, req.NamespacedName, dpl)
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
				return r.cluster.UpdateSentinelState(ctx, connect.NewRequest(&ctrlv1.UpdateSentinelStateRequest{
					K8SCrdName:      req.Name,
					RunningReplicas: 0,
				}))
			})
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	_, err = r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		return r.cluster.UpdateSentinelState(ctx, connect.NewRequest(&ctrlv1.UpdateSentinelStateRequest{
			K8SCrdName:      req.Name,
			RunningReplicas: dpl.Status.ReadyReplicas,
		}))
	})

	return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
}
