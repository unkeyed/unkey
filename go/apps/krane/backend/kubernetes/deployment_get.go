package kubernetes

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
	"github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes/labels"
	"github.com/unkeyed/unkey/go/pkg/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetDeployment retrieves the current status and instance information for a Kubernetes deployment.
//
// This method queries the Kubernetes API for StatefulSet status and associated
// pod information to provide a comprehensive view of the deployment state.
// It returns detailed information about each pod instance including stable
// DNS addresses, current status, and resource allocation.
func (k *k8s) GetDeployment(ctx context.Context, req backend.GetDeploymentRequest) (backend.GetDeploymentResponse, error) {
	deploymentID := req.DeploymentID
	namespace := req.Namespace
	const krane = "krane"

	err := assert.All(
		assert.NotEmpty(namespace),
		assert.NotEmpty(deploymentID),
	)
	if err != nil {
		return backend.GetDeploymentResponse{}, err
	}

	k.logger.Info("getting deployment", "deployment_id", deploymentID)

	// Create label selector for this deployment
	labelSelector := fmt.Sprintf("%s=%s,%s=%s", labels.DeploymentID, deploymentID, labels.ManagedBy, krane)

	// List StatefulSets with this deployment-id label
	// nolint: exhaustruct
	statefulSets, err := k.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return backend.GetDeploymentResponse{}, fmt.Errorf("failed to list statefulsets: %w", err)
	}

	if len(statefulSets.Items) == 0 {
		return backend.GetDeploymentResponse{}, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	// Use the first (and should be only) StatefulSet
	sfs := &statefulSets.Items[0]

	// Determine job status
	var status backend.DeploymentStatus
	if sfs.Status.AvailableReplicas == sfs.Status.Replicas {
		status = backend.DEPLOYMENT_STATUS_RUNNING
	} else {
		status = backend.DEPLOYMENT_STATUS_PENDING
	}

	// List Services with this deployment-id label
	// nolint: exhaustruct
	services, err := k.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return backend.GetDeploymentResponse{}, fmt.Errorf("failed to list services: %w", err)
	}

	if len(services.Items) == 0 {
		return backend.GetDeploymentResponse{}, fmt.Errorf("no service found for deployment: %s", deploymentID)
	}

	// Use the first service
	service := &services.Items[0]
	var port int32 = 8080 // default
	if len(service.Spec.Ports) > 0 {
		port = service.Spec.Ports[0].Port
	}

	// Get all pods belonging to this stateful set
	podLabelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchExpressions: nil,
		MatchLabels:      sfs.Spec.Selector.MatchLabels,
	})

	//nolint: exhaustruct
	pods, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: podLabelSelector,
	})
	if err != nil {
		return backend.GetDeploymentResponse{}, fmt.Errorf("failed to list pods: %w", err)
	}

	// Build instances from pods
	var instances []backend.Instance
	for _, pod := range pods.Items {
		// Determine pod status
		var podStatus backend.DeploymentStatus
		switch pod.Status.Phase {
		case corev1.PodPending:
			podStatus = backend.DEPLOYMENT_STATUS_PENDING
		case corev1.PodRunning:
			podStatus = backend.DEPLOYMENT_STATUS_RUNNING
		case corev1.PodFailed:
			podStatus = backend.DEPLOYMENT_STATUS_TERMINATING
		case corev1.PodSucceeded:
			// Handling to handle at this point
		case corev1.PodUnknown:
			podStatus = backend.DEPLOYMENT_STATUS_UNSPECIFIED
		default:
			podStatus = backend.DEPLOYMENT_STATUS_UNSPECIFIED
		}
		// pod-1.my-headless-service.default.svc.cluster.local

		// Create DNS entry for the pod
		// For StatefulSets, pods have predictable DNS names: <pod-name>.<service-name>.<namespace>.svc.cluster.local
		podDNS := fmt.Sprintf("%s.%s.%s.svc.cluster.local:%d", pod.Name, service.Name, pod.Namespace, port)
		instances = append(instances, backend.Instance{
			Id:      pod.Name,
			Address: podDNS,
			Status:  podStatus,
		})
	}

	k.logger.Info("deployment found",
		"deployment_id", deploymentID,
		"status", string(status),
		"port", port,
		"pod_count", len(instances),
	)

	return backend.GetDeploymentResponse{
		Instances: instances,
	}, nil
}
