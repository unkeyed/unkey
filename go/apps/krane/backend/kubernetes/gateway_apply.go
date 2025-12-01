package kubernetes

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
	"github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes/labels"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ApplyGateway creates or updates a gateway using Kubernetes Deployments and Services.
//
// This method implements idempotent gateway management. Unlike application deployments
// which use StatefulSets, gateways use standard Deployments since they don't require
// stable network identities. The operation is fully idempotent and can be safely
// retried without creating duplicate resources.
//
// Idempotency Implementation:
//  1. Check/create namespace if it doesn't exist
//  2. List existing Deployments by label selector
//  3. Reuse existing Deployment if found, create if missing
//  4. List existing Services by label selector
//  5. Reuse existing Service if found, create if missing
//  6. Update Service owner references for cleanup
//
// The method handles these idempotent scenarios:
//   - All resources exist: Updates owner references only
//   - Partial resources exist: Creates missing resources
//   - No resources exist: Creates all resources
//   - Multiple resources found: Returns error (unexpected state)
//
// Safe to retry on any error without creating duplicates.
func (k *k8s) ApplyGateway(ctx context.Context, req backend.ApplyGatewayRequest) error {

	k.logger.Info("creating gateway",
		"namespace", req.Namespace,
		"gateway_id", req.GatewayID,
	)

	// Ensure namespace exists (idempotent)
	// It's not ideal to do it here, but it's the best I can do for now without rebuilding the entire workspace creation system.
	_, err := k.clientset.CoreV1().Namespaces().Get(ctx, req.Namespace, metav1.GetOptions{})
	if err != nil {
		// If namespace doesn't exist, create it
		if errors.IsNotFound(err) {
			k.logger.Info("namespace not found, creating it", "namespace", req.Namespace)
			_, createErr := k.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: req.Namespace,
					Labels: map[string]string{
						labels.ManagedBy: krane,
					},
				},
			}, metav1.CreateOptions{})
			if createErr != nil && !errors.IsAlreadyExists(createErr) {
				return fmt.Errorf("failed to create namespace: %w", createErr)
			}
			k.logger.Info("namespace created successfully", "namespace", req.Namespace)
		} else {
			// Some other error occurred while getting the namespace
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}

	// Define labels for resource selection
	usedLabels := map[string]string{
		labels.WorkspaceID:   req.WorkspaceID,
		labels.ProjectID:     req.ProjectID,
		labels.EnvironmentID: req.EnvironmentID,
		labels.GatewayID:     req.GatewayID,
		labels.ManagedBy:     krane,
	}

	labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: usedLabels,
	})

	// Check if Deployment already exists (idempotent)
	deployments, err := k.clientset.AppsV1().Deployments(req.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	var deployment *appsv1.Deployment
	if len(deployments.Items) == 1 {
		deployment = &deployments.Items[0]
		k.logger.Info("deployment already exists, using existing", "name", deployment.Name)
	} else if len(deployments.Items) > 1 {
		return fmt.Errorf("multiple deployments found for gateway %s", req.GatewayID)
	} else {
		// Create Deployment only if it doesn't exist
		deployment, err = k.clientset.AppsV1().Deployments(req.Namespace).Create(ctx,
			//nolint: exhaustruct
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "gw-",
					Namespace:    req.Namespace,
					Labels:       usedLabels,
				},

				//nolint: exhaustruct
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.P(int32(req.Replicas)), //nolint: gosec
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							labels.WorkspaceID:   req.WorkspaceID,
							labels.ProjectID:     req.ProjectID,
							labels.EnvironmentID: req.EnvironmentID,
							labels.GatewayID:     req.GatewayID,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      usedLabels,
							Annotations: map[string]string{},
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyAlways,
							Containers: []corev1.Container{
								{
									Name:  "gateway",
									Image: req.Image,
									Env: []corev1.EnvVar{
										{Name: "UNKEY_WORKSPACE_ID", Value: req.WorkspaceID},
										{Name: "UNKEY_PROJECT_ID", Value: req.ProjectID},
										{Name: "UNKEY_ENVIRONMENT_ID", Value: req.EnvironmentID},
										{Name: "UNKEY_GATEWAY_ID", Value: req.GatewayID},
										{Name: "UNKEY_IMAGE", Value: req.Image},
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
											corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.CpuMillicores), resource.DecimalSI),
											corev1.ResourceMemory: *resource.NewQuantity(int64(req.MemorySizeMib)*1024*1024, resource.DecimalSI), //nolint: gosec

										},
										// nolint: exhaustive
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.CpuMillicores), resource.DecimalSI),
											corev1.ResourceMemory: *resource.NewQuantity(int64(req.MemorySizeMib)*1024*1024, resource.DecimalSI), //nolint: gosec
										},
									},
								},
							},
						},
					},
				},
			}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create gateway deployment: %w", err)
		}
		k.logger.Info("deployment created successfully", "name", deployment.Name)
	}

	// Check if Service already exists (idempotent)
	services, err := k.clientset.CoreV1().Services(req.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	var service *corev1.Service
	if len(services.Items) == 1 {
		service = &services.Items[0]
		k.logger.Info("service already exists, using existing", "name", service.Name)
	} else if len(services.Items) > 1 {
		return fmt.Errorf("multiple services found for gateway %s", req.GatewayID)
	} else {
		// Create Service only if it doesn't exist
		service, err = k.clientset.CoreV1().
			Services(req.Namespace).
			Create(ctx,
				// This implementation uses Deployments with ClusterIP services
				// for better scalability while maintaining internal accessibility via service name.
				// The service can be accessed within the cluster using: <service-name>.<namespace>.svc.cluster.local
				//nolint:exhaustruct
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "gw-svc-",
						Namespace:    req.Namespace,
						Labels:       usedLabels,
					},
					//nolint:exhaustruct
					Spec: corev1.ServiceSpec{
						Type:     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
						Selector: usedLabels,
						//nolint:exhaustruct
						Ports: []corev1.ServicePort{
							{
								Port:       8040,
								TargetPort: intstr.FromInt(8040),
								Protocol:   corev1.ProtocolTCP,
							},
						},
					},
				},
				//nolint:exhaustruct
				metav1.CreateOptions{},
			)
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
		k.logger.Info("service created successfully", "name", service.Name)
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
		_, err = k.clientset.CoreV1().Services(req.Namespace).Update(ctx, service, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update service owner references: %w", err)
		}
		k.logger.Info("service owner references updated")
	}

	k.logger.Info("Gateway resources ready",
		"deployment", deployment.Name,
		"service", service.Name,
		"gateway_id", req.GatewayID,
	)

	return nil
}
