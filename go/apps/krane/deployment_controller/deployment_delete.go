package deploymentcontroller

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteDeployment removes a deployment and all associated Kubernetes resources.
//
// This method performs a complete cleanup of a deployment by removing both
// the Service and StatefulSet resources. Resources are selected by their
// deployment-id label rather than by name, following Kubernetes best practices
// for resource management.
func (c *DeploymentController) DeleteDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) error {
	deploymentID := req.GetDeploymentId()

	c.logger.Info("deleting deployment",
		"deployment_id", deploymentID,
	)

	// Create label selector for this deployment
	labelSelector := fmt.Sprintf("deployment-id=%s", deploymentID)

	//nolint: exhaustruct
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
	}

	// List and delete Services with this deployment-id label
	//nolint: exhaustruct
	serviceList, err := c.clientset.CoreV1().Services(k8s.UntrustedNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range serviceList.Items {
		c.logger.Debug("deleting service",
			"name", service.Name,
			"deployment_id", deploymentID,
		)
		err = c.clientset.CoreV1().Services(k8s.UntrustedNamespace).Delete(ctx, service.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete service %s: %w", service.Name, err)
		}
	}

	// List and delete StatefulSets with this deployment-id label
	//nolint: exhaustruct
	statefulSetList, err := c.clientset.AppsV1().StatefulSets(k8s.UntrustedNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list statefulsets: %w", err)
	}

	for _, statefulSet := range statefulSetList.Items {
		c.logger.Debug("deleting statefulset",
			"name", statefulSet.Name,
			"deployment_id", deploymentID,
		)
		err = c.clientset.AppsV1().StatefulSets(k8s.UntrustedNamespace).Delete(ctx, statefulSet.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete statefulset %s: %w", statefulSet.Name, err)
		}
	}

	c.logger.Info("deployment deleted successfully",
		"deployment_id", deploymentID,
		"services_deleted", len(serviceList.Items),
		"statefulsets_deleted", len(statefulSetList.Items),
	)

	return nil
}
