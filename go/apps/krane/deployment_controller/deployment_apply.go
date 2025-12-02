package deploymentcontroller

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ApplyDeployment creates or updates a deployment using Kubernetes StatefulSets and Services.
//
// This method implements idempotent deployment management using a two-resource approach:
// a headless Service for DNS-based service discovery and a StatefulSet for
// managing pod replicas with stable network identities. The operation is fully
// idempotent - it can be called multiple times with the same parameters without
// creating duplicate resources or returning errors.
//
// Idempotency Implementation:
//  1. List existing Services by label selector
//  2. Reuse existing Service if found, create if missing
//  3. List existing StatefulSets by label selector
//  4. Reuse existing StatefulSet if found, create if missing
//  5. Update Service ownership to StatefulSet for automatic cleanup
//
// The method handles these idempotent scenarios:
//   - Service exists, StatefulSet exists: Updates owner reference only
//   - Service exists, StatefulSet missing: Creates StatefulSet
//   - Service missing, StatefulSet exists: Creates Service
//   - Neither exists: Creates both resources
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
//	The method is designed to be retry-safe. Network errors, timeouts, and
//	partial failures can all be recovered by retrying this method. Multiple
//	resources with the same labels are treated as an error condition.
func (c *DeploymentController) ApplyDeployment(ctx context.Context, req *ctrlv1.ApplyDeployment) error {

	deploymentID := req.GetDeploymentId()

	c.logger.Info("creating deployment",
		"deployment_id", deploymentID,
	)

	usedLabels := k8s.NewLabels().
		DeploymentID(deploymentID).
		ManagedByKrane().
		ComponentDeployment().
		ToMap()

	labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: usedLabels,
	})

	services, err := c.clientset.CoreV1().Services(k8s.UntrustedNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	var service *corev1.Service
	if len(services.Items) == 1 {
		service = &services.Items[0]
	} else if len(services.Items) > 1 {
		return fmt.Errorf("multiple services found")
	} else {

		service, err = c.clientset.CoreV1().
			Services(k8s.UntrustedNamespace).
			Create(ctx,
				// This implementation of using stateful sets is very likely not what we want to
				// use in v1.
				//
				// It's simply what fits our existing abstraction of microVMs best, because it gives us
				// stable dns addresses for each pod and that's what our database and gatway expect.
				//
				// I believe going forward we need to re-evaluate that cause it's the wrong abstraction.
				//nolint:exhaustruct
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "svc-",
						Namespace:    k8s.UntrustedNamespace,
						Labels:       usedLabels,
					},

					//nolint:exhaustruct
					Spec: corev1.ServiceSpec{
						Type:                     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
						Selector:                 usedLabels,
						ClusterIP:                "None",
						PublishNotReadyAddresses: true,

						//nolint:exhaustruct
						Ports: []corev1.ServicePort{
							{
								Port:       8080,
								TargetPort: intstr.FromInt(8080),
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
	}

	statefulsets, err := c.clientset.AppsV1().StatefulSets(k8s.UntrustedNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list stateful sets: %w", err)
	}

	var sfs *appsv1.StatefulSet
	if len(statefulsets.Items) == 1 {
		sfs = &statefulsets.Items[0]
	} else if len(statefulsets.Items) > 1 {
		return fmt.Errorf("multiple stateful sets found")
	} else {

		sfs, err = c.clientset.AppsV1().StatefulSets(k8s.UntrustedNamespace).Create(ctx,
			//nolint: exhaustruct
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "dpl-",
					Namespace:    k8s.UntrustedNamespace,
					Labels:       usedLabels,
				},

				//nolint: exhaustruct
				Spec: appsv1.StatefulSetSpec{
					ServiceName: service.Name,
					Replicas:    ptr.P(int32(req.GetReplicas())), //nolint: gosec
					Selector: &metav1.LabelSelector{
						MatchLabels: k8s.NewLabels().
							DeploymentID(deploymentID).
							ToMap(),
					},
					PodManagementPolicy: appsv1.ParallelPodManagement,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      usedLabels,
							Annotations: map[string]string{},
						},
						Spec: corev1.PodSpec{
							ImagePullSecrets: func() []corev1.LocalObjectReference {
								// Only add imagePullSecrets if using Depot registry
								if strings.HasPrefix(req.GetImage(), "registry.depot.dev/") {
									return []corev1.LocalObjectReference{
										{
											Name: "depot-registry",
										},
									}
								}
								return nil
							}(),
							RestartPolicy: corev1.RestartPolicyAlways,
							Containers: []corev1.Container{
								{
									Name:  "container",
									Image: req.GetImage(),
									Env: []corev1.EnvVar{
										{Name: "UNKEY_PROJECT_ID", Value: req.GetProjectId()},
										{Name: "UNKEY_ENVIRONMENT_ID", Value: req.GetEnvironmentId()},
										{Name: "UNKEY_DEPLOYMENT_ID", Value: req.GetDeploymentId()},
									},
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
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
					PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
						WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
					},
				},
			}, metav1.CreateOptions{})
		if err != nil {

			return fmt.Errorf("failed to create stateful set: %w", err)
		}
	}

	service.OwnerReferences = []metav1.OwnerReference{
		// Automatically clean up the service, when the deployment gets deleted
		//nolint:exhaustruct
		{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
			Name:       sfs.Name,
			UID:        sfs.UID,
		},
	}
	//nolint:exhaustruct
	_, err = c.clientset.CoreV1().Services(k8s.UntrustedNamespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	return nil
}
