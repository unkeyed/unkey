package sentinelreflector

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *Reflector) updateState(ctx context.Context, req types.NamespacedName) error {

	deployment, err := r.clientSet.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {

			_, err = r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
				return r.cluster.UpdateSentinelState(ctx, connect.NewRequest(&ctrlv1.UpdateSentinelStateRequest{
					K8SName:           req.Name,
					AvailableReplicas: 0,
				}))
			})
			if err != nil {
				return err
			}

			// nolint:exhaustruct
			return nil
		}
		return err
	}

	_, err = r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		return r.cluster.UpdateSentinelState(ctx, connect.NewRequest(&ctrlv1.UpdateSentinelStateRequest{
			K8SName:           req.Name,
			AvailableReplicas: deployment.Status.AvailableReplicas,
		}))
	})

	if err != nil {
		return err
	}
	return nil
}
