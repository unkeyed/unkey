package kubernetes

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
	"github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes/labels"
	"github.com/unkeyed/unkey/go/pkg/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetGateway retrieves the current status and instance information for a Kubernetes deployment.
//
// This method queries the Kubernetes API for Deployment status and associated
// pod information to provide a comprehensive view of the deployment state.
// It returns detailed information about each pod instance including stable
// DNS addresses, current status, and resource allocation.
func (k *k8s) GetGateway(ctx context.Context, req backend.GetGatewayRequest) (backend.GetGatewayResponse, error) {
	err := assert.All(
		assert.NotEmpty(req.Namespace),
		assert.NotEmpty(req.GatewayID),
	)
	if err != nil {
		return backend.GetGatewayResponse{}, err
	}

	k.logger.Info("getting gateway", "gateway_id", req.GatewayID)

	// Create label selector for this gateway
	labelSelector := fmt.Sprintf("%s=%s,%s=%s", labels.GatewayID, req.GatewayID, labels.ManagedBy, krane)

	// List Deployments with this gateway-id label
	// nolint: exhaustruct
	deployments, err := k.clientset.AppsV1().Deployments(req.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return backend.GetGatewayResponse{}, fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments.Items) == 0 {
		return backend.GetGatewayResponse{}, fmt.Errorf("gateway not found: %s", req.GatewayID)
	}

	// Use the first (and should be only) Deployment
	deployment := &deployments.Items[0]

	// Chec

	// Determine gateway status
	var status backend.GatewayStatus
	if deployment.Status.AvailableReplicas == deployment.Status.Replicas {
		status = backend.GATEWAY_STATUS_RUNNING
	} else {
		status = backend.GATEWAY_STATUS_PENDING
	}

	// List Services with this gateway-id label
	// nolint: exhaustruct
	services, err := k.clientset.CoreV1().Services(req.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return backend.GetGatewayResponse{}, fmt.Errorf("failed to list services: %w", err)
	}

	if len(services.Items) == 0 {
		return backend.GetGatewayResponse{}, fmt.Errorf("no service found for gateway: %s", req.GatewayID)
	}

	// Use the first service
	service := &services.Items[0]
	var port int32 = 8040 // default gateway port
	if len(service.Spec.Ports) > 0 {
		port = service.Spec.Ports[0].Port
	}

	// Gateway status is the overall status from deployment - we don't need pod-level details
	// for the new interface
	k.logger.Info("gateway found",
		"gateway_id", req.GatewayID,
		"status", string(status),
		"port", port,
	)

	return backend.GetGatewayResponse{
		Status: status,
	}, nil
}
