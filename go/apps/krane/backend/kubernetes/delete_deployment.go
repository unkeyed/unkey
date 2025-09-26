package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *k8s) DeleteDeployment(ctx context.Context, req *connect.Request[kranev1.DeleteDeploymentRequest]) (*connect.Response[kranev1.DeleteDeploymentResponse], error) {
	k8sDeploymentID := strings.ReplaceAll(req.Msg.GetDeploymentId(), "_", "-")

	k.logger.Info("deleting deployment",
		"namespace", req.Msg.GetNamespace(),
		"deployment_id", k8sDeploymentID,
	)

	err := k.clientset.CoreV1().Services(req.Msg.GetNamespace()).Delete(ctx, k8sDeploymentID, metav1.DeleteOptions{
		PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete service: %w", err))
	}

	err = k.clientset.AppsV1().Deployments(req.Msg.GetNamespace()).Delete(ctx, k8sDeploymentID, metav1.DeleteOptions{
		PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete deployment: %w", err))
	}
	return connect.NewResponse(&kranev1.DeleteDeploymentResponse{}), nil
}
