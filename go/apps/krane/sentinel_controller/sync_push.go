package sentinelcontroller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type pusher struct {
	client  client.Client
	cb      circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.PushSentinelStateResponse]]
	cluster ctrlv1connect.ClusterServiceClient
}

func (p *pusher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	dpl := &appsv1.Deployment{}
	err := p.client.Get(ctx, req.NamespacedName, dpl)
	if err == nil {

		_, err = p.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.PushSentinelStateResponse], error) {
			return p.cluster.PushSentinelState(ctx, connect.NewRequest(&ctrlv1.PushSentinelStateRequest{
				Change: &ctrlv1.PushSentinelStateRequest_Upsert_{
					Upsert: &ctrlv1.PushSentinelStateRequest_Upsert{
						K8SCrdName:      req.Name,
						RunningReplicas: dpl.Status.ReadyReplicas,
					},
				},
			}))
		})

		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to push sentinel state: %w", err)
		}

	}
	if apierrors.IsNotFound(err) {
		_, err = p.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.PushSentinelStateResponse], error) {
			return p.cluster.PushSentinelState(ctx, connect.NewRequest(&ctrlv1.PushSentinelStateRequest{
				Change: &ctrlv1.PushSentinelStateRequest_Delete_{
					Delete: &ctrlv1.PushSentinelStateRequest_Delete{
						K8SCrdName: req.Name,
					},
				},
			}))
		})

		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to push sentinel state: %w", err)
		}

	}
	return ctrl.Result{}, err

}
