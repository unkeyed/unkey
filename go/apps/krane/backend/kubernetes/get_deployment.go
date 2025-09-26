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
	sfs, err := k.clientset.AppsV1().StatefulSets(req.Msg.Namespace).Get(ctx, k8sDeploymentID, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", k8sDeploymentID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	k.logger.Info("deployment retrieved", "deployment", sfs.String())

	// Check if this job is managed by Krane
	managedBy, exists := sfs.Labels["unkey.managed.by"]
	if !exists || managedBy != "krane" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", k8sDeploymentID))
	}

	// Determine job status
	var status kranev1.DeploymentStatus
	if sfs.Status.AvailableReplicas == sfs.Status.Replicas {
		status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING
	} else {
		status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
	}

	// Get the service to retrieve port info
	service, err := k.clientset.CoreV1().Services(req.Msg.GetNamespace()).Get(ctx, k8sDeploymentID, metav1.GetOptions{})
	var port int32 = 8080 // default
	if err == nil && len(service.Spec.Ports) > 0 {
		port = service.Spec.Ports[0].Port
	}

	// Get all pods belonging to this stateful set
	labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: sfs.Spec.Selector.MatchLabels,
	})

	pods, err := k.clientset.CoreV1().Pods(req.Msg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list pods: %w", err))
	}

	// Build instances from pods
	var instances []*kranev1.Instance
	for _, pod := range pods.Items {
		// Determine pod status
		var podStatus kranev1.DeploymentStatus
		switch pod.Status.Phase {
		case corev1.PodPending:
			podStatus = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
		case corev1.PodRunning:
			podStatus = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING
		case corev1.PodFailed:
			podStatus = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_TERMINATING
		default:
			podStatus = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED
		}
		//pod-1.my-headless-service.default.svc.cluster.local

		// Create DNS entry for the pod
		// For StatefulSets, pods have predictable DNS names: <pod-name>.<service-name>.<namespace>.svc.cluster.local
		podDNS := fmt.Sprintf("%s.%s.%s.svc.cluster.local:%d", pod.Name, service.Name, pod.Namespace, port)
		instances = append(instances, &kranev1.Instance{
			Id:      pod.Name,
			Address: podDNS,
			Status:  podStatus,
		})
	}

	k.logger.Info("deployment found",
		"deployment_id", k8sDeploymentID,
		"status", status.String(),
		"port", port,
		"pod_count", len(instances),
	)

	return connect.NewResponse(&kranev1.GetDeploymentResponse{
		Instances: instances,
	}), nil
}
