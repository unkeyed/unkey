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

// DeleteDeployment removes a deployment and all associated Kubernetes resources.
//
// This method performs a complete cleanup of a deployment by removing both
// the Service and StatefulSet resources. The cleanup follows Kubernetes
// best practices for resource deletion with background propagation to
// ensure associated pods and other resources are properly terminated.
func (k *k8s) DeleteDeployment(ctx context.Context, req *connect.Request[kranev1.DeleteDeploymentRequest]) (*connect.Response[kranev1.DeleteDeploymentResponse], error) {
	k8sDeploymentID := strings.ReplaceAll(req.Msg.GetDeploymentId(), "_", "-")
	namespace := safeIDForK8s(req.Msg.GetNamespace())

	k.logger.Info("deleting deployment",
		"namespace", namespace,
		"deployment_id", k8sDeploymentID,
	)

	//nolint: exhaustruct
	err := k.clientset.CoreV1().Services(namespace).Delete(ctx, k8sDeploymentID, metav1.DeleteOptions{
		PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete service: %w", err))
	}

	//nolint: exhaustruct
	err = k.clientset.AppsV1().StatefulSets(namespace).Delete(ctx, k8sDeploymentID, metav1.DeleteOptions{
		PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete deployment: %w", err))
	}
	return connect.NewResponse(&kranev1.DeleteDeploymentResponse{}), nil
}
