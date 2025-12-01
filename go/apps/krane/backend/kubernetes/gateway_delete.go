package kubernetes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes/labels"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteGateway removes a gateway and all associated Kubernetes resources.
//
// This method performs a complete cleanup of a gateway by removing both
// the Service and Deployment resources. Resources are selected by their
// gateway-id label rather than by name, following Kubernetes best practices
// for resource management.
func (k *k8s) DeleteGateway(ctx context.Context, req *connect.Request[kranev1.DeleteGatewayRequest]) (*connect.Response[kranev1.DeleteGatewayResponse], error) {
	gatewayID := req.Msg.GetGatewayId()
	namespace := req.Msg.GetNamespace()

	k.logger.Info("deleting gateway",
		"namespace", namespace,
		"gateway_id", gatewayID,
	)

	// Create label selector for this gateway
	labelSelector := fmt.Sprintf("%s=%s,%s=%s", labels.GatewayID, gatewayID, labels.ManagedBy, krane)

	//nolint: exhaustruct
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
	}

	// List and delete Services with this gateway-id label
	//nolint: exhaustruct
	serviceList, err := k.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list services: %w", err))
	}

	for _, service := range serviceList.Items {
		k.logger.Debug("deleting service",
			"name", service.Name,
			"gateway_id", gatewayID,
		)
		err = k.clientset.CoreV1().Services(namespace).Delete(ctx, service.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete service %s: %w", service.Name, err))
		}
	}

	// List and delete Deployments with this gateway-id label
	//nolint: exhaustruct
	deploymentList, err := k.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list deployments: %w", err))
	}

	for _, deployment := range deploymentList.Items {
		k.logger.Debug("deleting deployment",
			"name", deployment.Name,
			"gateway_id", gatewayID,
		)
		err = k.clientset.AppsV1().Deployments(namespace).Delete(ctx, deployment.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete deployment %s: %w", deployment.Name, err))
		}
	}

	k.logger.Info("gateway deleted successfully",
		"namespace", namespace,
		"gateway_id", gatewayID,
		"services_deleted", len(serviceList.Items),
		"deployments_deleted", len(deploymentList.Items),
	)

	return connect.NewResponse(&kranev1.DeleteGatewayResponse{}), nil
}
