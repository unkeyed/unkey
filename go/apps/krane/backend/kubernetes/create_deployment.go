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

func (k *k8s) CreateDeployment(ctx context.Context, req *connect.Request[kranev1.CreateDeploymentRequest]) (*connect.Response[kranev1.CreateDeploymentResponse], error) {
	k8sDeploymentID := safeIDForK8s(req.Msg.GetDeployment().GetDeploymentId())
	k.logger.Info("creating deployment",
		"namespace", req.Msg.GetDeployment().GetNamespace(),
		"deployment_id", k8sDeploymentID,
	)

	service, err := k.clientset.CoreV1().
		Services(req.Msg.GetDeployment().GetNamespace()).
		Create(ctx,
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
				Replicas:    ptr.P(int32(req.Msg.GetDeployment().GetReplicas())),
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
							corev1.Container{
								Name:  "todo",
								Image: req.Msg.GetDeployment().GetImage(),
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 8080,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.Msg.GetDeployment().GetCpuMillicores()), resource.DecimalSI),
										corev1.ResourceMemory: *resource.NewQuantity(int64(req.Msg.GetDeployment().GetMemorySizeMib())*1024*1024, resource.DecimalSI),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.Msg.GetDeployment().GetCpuMillicores()), resource.DecimalSI),
										corev1.ResourceMemory: *resource.NewQuantity(int64(req.Msg.GetDeployment().GetMemorySizeMib())*1024*1024, resource.DecimalSI),
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
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	sfs.OwnerReferences = []metav1.OwnerReference{
		// Automatically clean up the service, when the deployment gets deleted
		{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
			Name:       k8sDeploymentID,
			UID:        sfs.UID,
		},
	}

	_, err = k.clientset.AppsV1().StatefulSets(req.Msg.GetDeployment().GetNamespace()).Update(ctx, sfs, metav1.UpdateOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment: %w", err))
	}

	k.logger.Info("Deployment created successfully",
		"deployment", sfs.String(),
	)

	// Create service to expose the VM with specific ClusterIP

	k.logger.Info("Service created successfully",
		"service", service.String(),
	)

	return connect.NewResponse(&kranev1.CreateDeploymentResponse{
		Status: kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}
