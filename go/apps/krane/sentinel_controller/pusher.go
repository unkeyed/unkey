package sentinelcontroller

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type pusher struct {
	client  client.Client
	cb      circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.UpdateSentinelStateResponse]]
	cluster ctrlv1connect.ClusterServiceClient
}

func (p *pusher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	dpl := &appsv1.Deployment{}
	err := p.client.Get(ctx, req.NamespacedName, dpl)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	_, err = p.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		return p.cluster.UpdateSentinelState(ctx, connect.NewRequest(&ctrlv1.UpdateSentinelStateRequest{
			K8SCrdName:      req.Name,
			RunningReplicas: dpl.Status.ReadyReplicas,
		}))
	})

	return ctrl.Result{}, err
}
