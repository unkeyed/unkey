package gatewaycontroller

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
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
func (c *GatewayController) DeleteGateway(ctx context.Context, req *ctrlv1.DeleteGateway) error {

	c.logger.Info("deleting gateway",
		"namespace", req.Namespace,
		"gateway_id", req.GetGatewayId(),
	)

	// Create label selector for this gateway
	labelSelector := fmt.Sprintf("%s=%s,%s=%s", k8s.LabelGatewayID, req.GetGatewayId(), k8s.LabelManagedBy, "krane")

	//nolint: exhaustruct
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
	}

	// List and delete Services with this gateway-id label
	//nolint: exhaustruct
	serviceList, err := c.clientset.CoreV1().Services(req.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range serviceList.Items {
		c.logger.Debug("deleting service",
			"name", service.Name,
			"gateway_id", req.GetGatewayId(),
		)
		err = c.clientset.CoreV1().Services(req.Namespace).Delete(ctx, service.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete service %s: %w", service.Name, err)
		}
	}

	// List and delete Deployments with this gateway-id label
	//nolint: exhaustruct
	deploymentList, err := c.clientset.AppsV1().Deployments(req.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deployment := range deploymentList.Items {
		c.logger.Debug("deleting deployment",
			"name", deployment.Name,
			"gateway_id", req.GetGatewayId(),
		)
		err = c.clientset.AppsV1().Deployments(req.Namespace).Delete(ctx, deployment.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete deployment %s: %w", deployment.Name, err)
		}
	}

	c.logger.Info("gateway deleted successfully",
		"namespace", req.Namespace,
		"gateway_id", req.GetGatewayId(),
		"services_deleted", len(serviceList.Items),
		"deployments_deleted", len(deploymentList.Items),
	)

	return nil
}
