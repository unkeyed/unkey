package kubernetes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *k8s) GetDeployment(ctx context.Context, req *connect.Request[kranev1.GetDeploymentRequest]) (*connect.Response[kranev1.GetDeploymentResponse], error) {

	err := assert.All(
		assert.NotEmpty(req.Msg.Namespace),
		assert.NotEmpty(req.Msg.DeploymentId),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	k8sDeploymentID := safeIDForK8s(req.Msg.GetDeploymentId())

	k.logger.Info("getting deployment", "deployment_id", k8sDeploymentID)

	// Get the Job by name (deployment_id)
	deployment, err := k.clientset.AppsV1().Deployments(req.Msg.Namespace).Get(ctx, k8sDeploymentID, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", k8sDeploymentID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	k.logger.Info("deployment retrieved", "deployment", deployment.String())

	// Check if this job is managed by Krane
	managedBy, exists := deployment.Labels["unkey.managed.by"]
	if !exists || managedBy != "krane" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", k8sDeploymentID))
	}

	// Determine job status
	var status kranev1.DeploymentStatus
	if deployment.Status.AvailableReplicas == deployment.Status.Replicas {
		status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING
	} else if deployment.Status.UnavailableReplicas > 0 {
		status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
	} else {
		status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED
	}

	// Get the service to retrieve port info
	service, err := k.clientset.CoreV1().Services(req.Msg.GetNamespace()).Get(ctx, k8sDeploymentID, metav1.GetOptions{})
	var port int32 = 8080 // default
	if err == nil && len(service.Spec.Ports) > 0 {
		port = service.Spec.Ports[0].Port
	}

	k.logger.Info("deployment found",
		"deployment_id", k8sDeploymentID,
		"status", status.String(),
		"port", port,
	)

	return connect.NewResponse(&kranev1.GetDeploymentResponse{
		Status: status,
		Instances: []*kranev1.Instance{
			{
				Id:      fmt.Sprintf("%s-%s", service.Namespace, service.Name),
				Address: fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, port),
			},
		},
	}), nil
}
