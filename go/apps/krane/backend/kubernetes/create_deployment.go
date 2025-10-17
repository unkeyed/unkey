package kubernetes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CreateDeployment creates a new deployment using Kubernetes StatefulSets and Services.
//
// This method implements deployment creation using a two-resource approach:
// a headless Service for DNS-based service discovery and a StatefulSet for
// managing pod replicas with stable network identities. This design choice
// supports the existing microVM abstraction where each instance requires
// predictable DNS names for database connections and gateway routing.
//
// Resource Creation Sequence:
//  1. Create headless Service (ClusterIP: None) for stable DNS resolution
//  2. Create StatefulSet with pod template and resource specifications
//  3. Set Service ownership to StatefulSet for automatic cleanup
//
// StatefulSet vs Deployment Choice:
//
//	The implementation uses StatefulSets instead of standard Deployments
//	because krane requires stable network identities for each replica.
//	This represents a departure from typical stateless application patterns
//	but aligns with the existing system architecture expectations.
//
//	Note: This design decision may be reconsidered in future versions as
//	it may not align with cloud-native best practices for stateless services.
//
// Resource Configuration:
//   - Containers: Single container per pod with specified image and resources
//   - CPU/Memory: Both requests and limits set to same values for predictable scheduling
//   - Networking: Container port 8080 exposed, Service provides cluster-wide access
//   - Labels: Applied for krane management tracking and resource grouping
//   - Restart Policy: Always restart containers on failure
//
// Service Discovery:
//
//	Each pod receives a stable DNS name following the pattern:
//	{pod-name}-{index}.{service-name}.{namespace}.svc.cluster.local
//	Example: myapp-0.myapp.unkey.svc.cluster.local
//
// Resource Ownership:
//
//	The Service is configured as owned by the StatefulSet through Kubernetes
//	owner references, ensuring automatic cleanup when the StatefulSet is deleted.
//	This prevents orphaned Services from accumulating in the cluster.
//
// Error Handling:
//
//	Service or StatefulSet creation failures return CodeInternal errors.
//	The method ensures transactional behavior - if StatefulSet creation fails
//	after Service creation, the Service remains (and will be cleaned up by
//	Kubernetes garbage collection if properly configured).
//
// Returns DEPLOYMENT_STATUS_PENDING as pods may not be immediately scheduled
// and ready for traffic after creation.
func (k *k8s) CreateDeployment(ctx context.Context, req *connect.Request[kranev1.CreateDeploymentRequest]) (*connect.Response[kranev1.CreateDeploymentResponse], error) {
	k8sDeploymentID := safeIDForK8s(req.Msg.GetDeployment().GetDeploymentId())
	k.logger.Info("creating deployment",
		"namespace", req.Msg.GetDeployment().GetNamespace(),
		"deployment_id", k8sDeploymentID,
	)

	service, err := k.clientset.CoreV1().
		Services(req.Msg.GetDeployment().GetNamespace()).
		Create(ctx,
			// This implementation of using stateful sets is very likely not what we want to
			// use in v1.
			//
			// It's simply what fits our existing abstraction of microVMs best, because it gives us
			// stable dns addresses for each pod and that's what our database and gatway expect.
			//
			// I believe going forward we need to re-evaluate that cause it's the wrong abstraction.
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k8sDeploymentID,
					Namespace: req.Msg.GetDeployment().GetNamespace(),
					Labels: map[string]string{
						"unkey.deployment.id": k8sDeploymentID,
						"unkey.managed.by":    "krane",
					},

					Annotations: map[string]string{
						"unkey.deployment.id": k8sDeploymentID,
					},
				},

				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
					Selector: map[string]string{
						"unkey.deployment.id": k8sDeploymentID,
					},
					ClusterIP:                "None",
					PublishNotReadyAddresses: true,
					Ports: []corev1.ServicePort{
						{
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
							Protocol:   corev1.ProtocolTCP,
						},
					},
				},
			},
			metav1.CreateOptions{},
		)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create service: %w", err))
	}

	sfs, err := k.clientset.AppsV1().StatefulSets(req.Msg.GetDeployment().GetNamespace()).Create(ctx,
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      k8sDeploymentID,
				Namespace: req.Msg.GetDeployment().GetNamespace(),
				Labels: map[string]string{
					"unkey.deployment.id": k8sDeploymentID,
					"unkey.managed.by":    "krane",
				},
			},

			Spec: appsv1.StatefulSetSpec{
				ServiceName: service.Name,
				Replicas:    ptr.P(int32(req.Msg.GetDeployment().GetReplicas())), //nolint: gosec
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"unkey.deployment.id": k8sDeploymentID,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"unkey.deployment.id": k8sDeploymentID,
							"unkey.managed.by":    "krane",
						},
						Annotations: map[string]string{},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
						Containers: []corev1.Container{
							{
								Name:  "todo",
								Image: req.Msg.GetDeployment().GetImage(),
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 8080,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Resources: corev1.ResourceRequirements{
									// nolint: exhaustive
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.Msg.GetDeployment().GetCpuMillicores()), resource.DecimalSI),
										corev1.ResourceMemory: *resource.NewQuantity(int64(req.Msg.GetDeployment().GetMemorySizeMib())*1024*1024, resource.DecimalSI), //nolint: gosec

									},
									// nolint: exhaustive
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.Msg.GetDeployment().GetCpuMillicores()), resource.DecimalSI),
										corev1.ResourceMemory: *resource.NewQuantity(int64(req.Msg.GetDeployment().GetMemorySizeMib())*1024*1024, resource.DecimalSI), //nolint: gosec
									},
								},
							},
						},
					},
				},
				PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
					WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
					WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		k.logger.Info("Deleting service, because deployment creation failed")
		// Delete service
		if rollbackErr := k.clientset.CoreV1().Services(req.Msg.GetDeployment().GetNamespace()).Delete(ctx, service.Name, metav1.DeleteOptions{}); rollbackErr != nil {
			k.logger.Error("Failed to delete service", "error", rollbackErr.Error())
		}

		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	service.OwnerReferences = []metav1.OwnerReference{
		// Automatically clean up the service, when the deployment gets deleted
		{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
			Name:       k8sDeploymentID,
			UID:        sfs.UID,
		},
	}

	_, err = k.clientset.CoreV1().Services(req.Msg.GetDeployment().GetNamespace()).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment: %w", err))
	}

	k.logger.Info("Deployment created successfully",
		"deployment", sfs.String(),
		"service", service.String(),
	)

	return connect.NewResponse(&kranev1.CreateDeploymentResponse{
		Status: kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}
