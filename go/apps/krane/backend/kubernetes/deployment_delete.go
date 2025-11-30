package kubernetes

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
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
func (k *k8s) DeleteDeployment(ctx context.Context, req backend.DeleteDeploymentRequest) error {
	deploymentID := req.DeploymentID
	namespace := req.Namespace

	k.logger.Info("deleting deployment",
		"namespace", namespace,
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
	serviceList, err := k.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range serviceList.Items {
		k.logger.Debug("deleting service",
			"name", service.Name,
			"deployment_id", deploymentID,
		)
		err = k.clientset.CoreV1().Services(namespace).Delete(ctx, service.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete service %s: %w", service.Name, err)
		}
	}

	// List and delete StatefulSets with this deployment-id label
	//nolint: exhaustruct
	statefulSetList, err := k.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list statefulsets: %w", err)
	}

	for _, statefulSet := range statefulSetList.Items {
		k.logger.Debug("deleting statefulset",
			"name", statefulSet.Name,
			"deployment_id", deploymentID,
		)
		err = k.clientset.AppsV1().StatefulSets(namespace).Delete(ctx, statefulSet.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete statefulset %s: %w", statefulSet.Name, err)
		}
	}

	k.logger.Info("deployment deleted successfully",
		"namespace", namespace,
		"deployment_id", deploymentID,
		"services_deleted", len(serviceList.Items),
		"statefulsets_deleted", len(statefulSetList.Items),
	)

	return nil
}
