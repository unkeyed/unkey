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

func (k *k8s) CreateGateway(ctx context.Context, req *connect.Request[kranev1.CreateGatewayRequest]) (*connect.Response[kranev1.CreateGatewayResponse], error) {
	k8sGatewayID := safeIDForK8s(req.Msg.GetGateway().GetGatewayId())
	k.logger.Info("creating deployment",
		"namespace", req.Msg.GetGateway().GetNamespace(),
		"deployment_id", k8sGatewayID,
	)

	// Create Deployment
	deployment, err := k.clientset.AppsV1().Deployments(req.Msg.GetGateway().GetNamespace()).Create(ctx,
		//nolint: exhaustruct
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      k8sGatewayID,
				Namespace: req.Msg.GetGateway().GetNamespace(),
				Labels: map[string]string{
					"unkey.gateway.id": k8sGatewayID,
					"unkey.managed.by": "krane",
				},
			},

			//nolint: exhaustruct
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.P(int32(req.Msg.GetGateway().GetReplicas())), //nolint: gosec
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"unkey.gateway.id": k8sGatewayID,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"unkey.gateway.id": k8sGatewayID,
							"unkey.managed.by": "krane",
						},
						Annotations: map[string]string{},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
						Containers: []corev1.Container{
							{
								Name:  req.Msg.GetGateway().GetGatewayId(),
								Image: req.Msg.GetGateway().GetImage(),
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 8040,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Resources: corev1.ResourceRequirements{
									// nolint: exhaustive
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.Msg.GetGateway().GetCpuMillicores()), resource.DecimalSI),
										corev1.ResourceMemory: *resource.NewQuantity(int64(req.Msg.GetGateway().GetMemorySizeMib())*1024*1024, resource.DecimalSI), //nolint: gosec

									},
									// nolint: exhaustive
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.Msg.GetGateway().GetCpuMillicores()), resource.DecimalSI),
										corev1.ResourceMemory: *resource.NewQuantity(int64(req.Msg.GetGateway().GetMemorySizeMib())*1024*1024, resource.DecimalSI), //nolint: gosec
									},
								},
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	// Create Service with owner reference to the Deployment
	service, err := k.clientset.CoreV1().
		Services(req.Msg.GetGateway().GetNamespace()).
		Create(ctx,
			// This implementation uses Deployments with ClusterIP services
			// for better scalability while maintaining internal accessibility via service name.
			// The service can be accessed within the cluster using: <service-name>.<namespace>.svc.cluster.local
			//nolint:exhaustruct
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k8sGatewayID,
					Namespace: req.Msg.GetGateway().GetNamespace(),
					Labels: map[string]string{
						"unkey.deployment.id": k8sGatewayID,
						"unkey.managed.by":    "krane",
					},
					Annotations: map[string]string{
						"unkey.deployment.id": k8sGatewayID,
					},
					OwnerReferences: []metav1.OwnerReference{
						// Automatically clean up the service when the Deployment gets deleted
						//nolint:exhaustruct
						{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       k8sGatewayID,
							UID:        deployment.UID,
						},
					},
				},
				//nolint:exhaustruct
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
					Selector: map[string]string{
						"unkey.deployment.id": k8sGatewayID,
					},
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
		k.logger.Info("Deleting deployment, because service creation failed")
		// Delete deployment
		// nolint: exhaustruct
		if rollbackErr := k.clientset.AppsV1().Deployments(req.Msg.GetGateway().GetNamespace()).Delete(ctx, deployment.Name, metav1.DeleteOptions{}); rollbackErr != nil {
			k.logger.Error("Failed to delete deployment", "error", rollbackErr.Error())
		}

		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create service: %w", err))
	}

	k.logger.Info("Deployment created successfully",
		"deployment", deployment.String(),
		"service", service.String(),
	)

	return connect.NewResponse(&kranev1.CreateGatewayResponse{
		Status: kranev1.GatewayStatus_GATEWAY_STATUS_PENDING,
	}), nil
}
