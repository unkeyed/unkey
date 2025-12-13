package kubernetes

import (
	"context"
	"encoding/base64"
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

// CreateDeployment creates a StatefulSet with a headless Service for stable DNS-based discovery.
// Uses StatefulSets (not Deployments) because each replica needs a predictable DNS name
// (e.g., myapp-0.myapp.unkey.svc.cluster.local) for gateway routing.
func (k *k8s) CreateDeployment(ctx context.Context, req *connect.Request[kranev1.CreateDeploymentRequest]) (*connect.Response[kranev1.CreateDeploymentResponse], error) {
	k8sDeploymentID := safeIDForK8s(req.Msg.GetDeployment().GetDeploymentId())
	namespace := safeIDForK8s(req.Msg.GetDeployment().GetNamespace())

	k.logger.Info("creating deployment",
		"namespace", namespace,
		"deployment_id", k8sDeploymentID,
	)

	service, err := k.clientset.CoreV1().
		Services(namespace).
		Create(ctx,
			//nolint:exhaustruct
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k8sDeploymentID,
					Namespace: namespace,
					Labels: map[string]string{
						"unkey.deployment.id": k8sDeploymentID,
						"unkey.managed.by":    "krane",
					},

					Annotations: map[string]string{
						"unkey.deployment.id": k8sDeploymentID,
					},
				},

				//nolint:exhaustruct
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Selector: map[string]string{
						"unkey.deployment.id": k8sDeploymentID,
					},
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
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create service: %w", err))
	}

	sfs, err := k.clientset.AppsV1().StatefulSets(namespace).Create(ctx,
		//nolint: exhaustruct
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      k8sDeploymentID,
				Namespace: namespace,
				Labels: map[string]string{
					"unkey.deployment.id": k8sDeploymentID,
					"unkey.managed.by":    "krane",
				},
			},

			//nolint: exhaustruct
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
							"unkey.com/inject":    "true",
						},
						Annotations: map[string]string{
							"unkey.com/deployment-id": req.Msg.GetDeployment().GetDeploymentId(),
							"unkey.com/build-id":      req.Msg.GetDeployment().GetBuildId(),
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName:           "customer-workload",
						AutomountServiceAccountToken: ptr.P(true),
						RestartPolicy:                corev1.RestartPolicyAlways,
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
								Env: func() []corev1.EnvVar {
									deployment := req.Msg.GetDeployment()
									env := []corev1.EnvVar{
										{
											Name:  "UNKEY_DEPLOYMENT_ID",
											Value: deployment.GetDeploymentId(),
										},
										{
											Name:  "UNKEY_ENVIRONMENT_ID",
											Value: deployment.GetEnvironmentId(),
										},
										{
											Name:  "UNKEY_REGION",
											Value: k.region,
										},
										{
											Name:  "UNKEY_ENVIRONMENT_SLUG",
											Value: deployment.GetEnvironmentSlug(),
										},
										{
											Name: "UNKEY_INSTANCE_ID",
											//nolint:exhaustruct
											ValueFrom: &corev1.EnvVarSource{
												//nolint:exhaustruct
												FieldRef: &corev1.ObjectFieldSelector{
													FieldPath: "metadata.name",
												},
											},
										},
									}

									if len(deployment.GetEncryptedSecretsBlob()) > 0 {
										env = append(env, corev1.EnvVar{
											Name:  "UNKEY_SECRETS_BLOB",
											Value: base64.StdEncoding.EncodeToString(deployment.GetEncryptedSecretsBlob()),
										})
									}

									return env
								}(),
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
		// nolint: exhaustruct
		if rollbackErr := k.clientset.CoreV1().Services(namespace).Delete(ctx, service.Name, metav1.DeleteOptions{}); rollbackErr != nil {
			k.logger.Error("Failed to delete service", "error", rollbackErr.Error())
		}

		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	service.OwnerReferences = []metav1.OwnerReference{
		//nolint:exhaustruct
		{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
			Name:       k8sDeploymentID,
			UID:        sfs.UID,
		},
	}
	//nolint:exhaustruct
	_, err = k.clientset.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})
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
