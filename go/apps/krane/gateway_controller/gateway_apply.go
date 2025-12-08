package gatewaycontroller

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (c *GatewayController) ApplyGateway(ctx context.Context, req *ctrlv1.ApplyGateway) error {

	// Define labels for resource selection
	usedLabels := k8s.NewLabels().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		EnvironmentID(req.GetEnvironmentId()).
		ComponentGateway().
		GatewayID(req.GetGatewayId()).
		ManagedByKrane().
		ToMap()

	deployment, service := buildResourceConfig(req, usedLabels)

	labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: usedLabels,
	})

	// Check if Deployment already exists (idempotent)
	deployments, err := c.clientset.AppsV1().Deployments(k8s.UntrustedNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments.Items) > 0 {
		deployment.Name = deployments.Items[0].Name
		deployment, err = c.clientset.AppsV1().Deployments(k8s.UntrustedNamespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update gateway deployment: %w", err)
		}
	} else {
		deployment, err = c.clientset.AppsV1().Deployments(k8s.UntrustedNamespace).Create(ctx,
			deployment, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create gateway deployment: %w", err)
		}
	}

	// Check if Service already exists (idempotent)
	services, err := c.clientset.CoreV1().Services(k8s.UntrustedNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if len(services.Items) > 0 {
		service.Name = services.Items[0].Name
		service, err = c.clientset.CoreV1().Services(k8s.UntrustedNamespace).Update(ctx, service, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to get service: %w", err)
		}
	} else {
		service, err = c.clientset.CoreV1().
			Services(k8s.UntrustedNamespace).
			Create(ctx,
				service,
				//nolint:exhaustruct
				metav1.CreateOptions{},
			)
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	// Update Service with owner reference to Deployment (idempotent - always set)
	needsUpdate := false
	if len(service.OwnerReferences) == 0 {
		needsUpdate = true
	} else {
		// Check if the deployment is already an owner
		found := false
		for _, ref := range service.OwnerReferences {
			if ref.UID == deployment.UID {
				found = true
				break
			}
		}
		if !found {
			needsUpdate = true
		}
	}

	if needsUpdate {
		service.OwnerReferences = []metav1.OwnerReference{
			// Automatically clean up the service when the Deployment gets deleted
			//nolint:exhaustruct
			{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       deployment.Name,
				UID:        deployment.UID,
			},
		}
		//nolint:exhaustruct
		_, err = c.clientset.CoreV1().Services(k8s.UntrustedNamespace).Update(ctx, service, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update service owner references: %w", err)
		}
	}

	c.logger.Debug("applied gateway", "gateway_id", req.GetGatewayId())

	return nil
}

func buildResourceConfig(req *ctrlv1.ApplyGateway, labels map[string]string) (*appsv1.Deployment, *corev1.Service) {

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "gw-",
			Namespace:    k8s.UntrustedNamespace,
			Labels:       labels,
		},

		//nolint: exhaustruct
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.P(int32(req.GetReplicas())), //nolint: gosec
			Selector: &metav1.LabelSelector{
				MatchLabels: k8s.NewLabels().
					WorkspaceID(req.GetWorkspaceId()).
					ProjectID(req.GetProjectId()).
					EnvironmentID(req.GetEnvironmentId()).
					GatewayID(req.GetGatewayId()).
					ToMap(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:    "gateway",
							Image:   req.GetImage(),
							Command: []string{"run", "gateway"},
							Env: []corev1.EnvVar{
								{Name: "UNKEY_WORKSPACE_ID", Value: req.GetWorkspaceId()},
								{Name: "UNKEY_PROJECT_ID", Value: req.GetProjectId()},
								{Name: "UNKEY_ENVIRONMENT_ID", Value: req.GetEnvironmentId()},
								{Name: "UNKEY_GATEWAY_ID", Value: req.GetGatewayId()},
								{Name: "UNKEY_IMAGE", Value: req.GetImage()},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8040,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								// nolint: exhaustive
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.GetCpuMillicores()), resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(int64(req.GetMemorySizeMib())*1024*1024, resource.DecimalSI), //nolint: gosec

								},
								// nolint: exhaustive
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.GetCpuMillicores()), resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(int64(req.GetMemorySizeMib())*1024*1024, resource.DecimalSI), //nolint: gosec
								},
							},
						},
					},
				},
			},
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "gw-svc-",
			Namespace:    k8s.UntrustedNamespace,
			Labels:       labels,
		},
		//nolint:exhaustruct
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
			Selector: labels,
			//nolint:exhaustruct
			Ports: []corev1.ServicePort{
				{
					Port:       8040,
					TargetPort: intstr.FromInt(8040),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	return deployment, service
}
