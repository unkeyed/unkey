package kubernetes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes/labels"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetGateway retrieves the current status and instance information for a Kubernetes deployment.
//
// This method queries the Kubernetes API for Deployment status and associated
// pod information to provide a comprehensive view of the deployment state.
// It returns detailed information about each pod instance including stable
// DNS addresses, current status, and resource allocation.
func (k *k8s) GetGateway(ctx context.Context, req *connect.Request[kranev1.GetGatewayRequest]) (*connect.Response[kranev1.GetGatewayResponse], error) {
	err := assert.All(
		assert.NotEmpty(req.Msg.GetNamespace()),
		assert.NotEmpty(req.Msg.GetGatewayId()),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	gatewayID := req.Msg.GetGatewayId()
	namespace := req.Msg.GetNamespace()

	k.logger.Info("getting gateway", "gateway_id", gatewayID)

	// Create label selector for this gateway
	labelSelector := fmt.Sprintf("%s=%s", labels.GatewayID, gatewayID)

	// List Deployments with this gateway-id label
	// nolint: exhaustruct
	deployments, err := k.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list deployments: %w", err))
	}

	if len(deployments.Items) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("gateway not found: %s", gatewayID))
	}

	// Use the first (and should be only) Deployment
	deployment := &deployments.Items[0]

	// Check if this gateway is managed by Krane
	managedBy, exists := deployment.Labels[labels.ManagedBy]
	if !exists || managedBy != "krane" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("gateway not found: %s", gatewayID))
	}

	// Determine gateway status
	var status kranev1.GatewayStatus
	if deployment.Status.AvailableReplicas == deployment.Status.Replicas {
		status = kranev1.GatewayStatus_GATEWAY_STATUS_RUNNING
	} else {
		status = kranev1.GatewayStatus_GATEWAY_STATUS_PENDING
	}

	// List Services with this gateway-id label
	// nolint: exhaustruct
	services, err := k.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list services: %w", err))
	}

	if len(services.Items) == 0 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("no service found for gateway: %s", gatewayID))
	}

	// Use the first service
	service := &services.Items[0]
	var port int32 = 8040 // default gateway port
	if len(service.Spec.Ports) > 0 {
		port = service.Spec.Ports[0].Port
	}

	pods, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchExpressions: nil,
			MatchLabels:      deployment.Spec.Selector.MatchLabels,
		}),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list pods: %w", err))
	}

	// Build instances from pods
	var instances []*kranev1.GatewayInstance
	for _, pod := range pods.Items {
		// Determine pod status
		var podStatus kranev1.GatewayStatus
		switch pod.Status.Phase {
		case corev1.PodPending:
			podStatus = kranev1.GatewayStatus_GATEWAY_STATUS_PENDING
		case corev1.PodRunning:
			podStatus = kranev1.GatewayStatus_GATEWAY_STATUS_RUNNING
		case corev1.PodFailed:
			podStatus = kranev1.GatewayStatus_GATEWAY_STATUS_TERMINATING
		case corev1.PodSucceeded:
			// Handling to handle at this point
		case corev1.PodUnknown:
			podStatus = kranev1.GatewayStatus_GATEWAY_STATUS_UNSPECIFIED
		default:
			podStatus = kranev1.GatewayStatus_GATEWAY_STATUS_UNSPECIFIED
		}

		instances = append(instances, &kranev1.GatewayInstance{
			Id:     pod.Name,
			Status: podStatus,
		})
	}

	k.logger.Info("gateway found",
		"gateway_id", gatewayID,
		"status", status.String(),
		"port", port,
		"pod_count", len(instances),
	)

	return connect.NewResponse(&kranev1.GetGatewayResponse{
		Address:   fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace),
		Instances: instances,
	}), nil
}
