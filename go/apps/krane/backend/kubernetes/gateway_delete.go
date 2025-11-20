package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *k8s) DeleteGateway(ctx context.Context, req *connect.Request[kranev1.DeleteGatewayRequest]) (*connect.Response[kranev1.DeleteGatewayResponse], error) {
	k8sGatewayID := strings.ReplaceAll(req.Msg.GetGatewayId(), "_", "-")

	namespace := safeIDForK8s(req.Msg.GetGatewayId())

	k.logger.Info("deleting deployment",
		"namespace", namespace,
		"gateway_id", k8sGatewayID,
	)

	err := k.clientset.AppsV1().Deployments(req.Msg.GetNamespace()).Delete(ctx, k8sGatewayID, metav1.DeleteOptions{
		PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete deployment: %w", err))
	}
	return connect.NewResponse(&kranev1.DeleteGatewayResponse{}), nil
}
