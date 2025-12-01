package kubernetes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	k8sgatewayID := safeIDForK8s(req.Msg.GetGatewayId())
	namespace := safeIDForK8s(req.Msg.GetNamespace())

	k.logger.Info("getting gateway", "gateway_id", k8sgatewayID)

	// Get the deployment by name (gateway_id)
	// nolint: exhaustruct
	deployment, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, k8sgatewayID, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", k8sgatewayID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	// Check if this gateway is managed by Krane
	managedBy, exists := deployment.Labels["unkey.managed.by"]
	if !exists || managedBy != "krane" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", k8sgatewayID))
	}

	// Determine gateway status
	var status kranev1.GatewayStatus
	if deployment.Status.AvailableReplicas == deployment.Status.Replicas {
		status = kranev1.GatewayStatus_GATEWAY_STATUS_RUNNING
	} else {
		status = kranev1.GatewayStatus_GATEWAY_STATUS_PENDING
	}

	// Get the service to retrieve port info
	service, err := k.clientset.CoreV1().Services(namespace).Get(ctx, k8sgatewayID, metav1.GetOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("could not load service: %s", k8sgatewayID))
	}
	var port int32 = 8080 // default
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

	k.logger.Info("deployment found",
		"deployment_id", k8sgatewayID,
		"status", status.String(),
		"port", port,
		"pod_count", len(instances),
	)

	return connect.NewResponse(&kranev1.GetGatewayResponse{
		Address:   fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace),
		Instances: instances,
	}), nil
}
