package kubernetes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes/labels"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (k *k8s) CreateGateway(ctx context.Context, req *connect.Request[kranev1.CreateGatewayRequest]) (*connect.Response[kranev1.CreateGatewayResponse], error) {
	gatewayID := req.Msg.GetGateway().GetGatewayId()
	namespace := req.Msg.GetGateway().GetNamespace()

	k.logger.Info("creating gateway",
		"namespace", namespace,
		"gateway_id", gatewayID,
	)

	// Ensure namespace exists
	// It's not ideal to do it here, but it's the best I can do for now without rebuilding the entire workspace creation system.
	_, err := k.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// If namespace doesn't exist, create it
		if errors.IsNotFound(err) {
			k.logger.Info("namespace not found, creating it", "namespace", namespace)
			_, createErr := k.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						labels.ManagedBy: krane,
					},
				},
			}, metav1.CreateOptions{})
			if createErr != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create namespace: %w", createErr))
			}
			k.logger.Info("namespace created successfully", "namespace", namespace)
		} else {
			// Some other error occurred while getting the namespace
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get namespace: %w", err))
		}
	}

	// Create Deployment
	deployment, err := k.clientset.AppsV1().Deployments(namespace).Create(ctx,
		//nolint: exhaustruct
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "gw-",
				Namespace:    namespace,
				Labels: map[string]string{
					labels.GatewayID: gatewayID,
					labels.ManagedBy: krane,
				},
			},

			//nolint: exhaustruct
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.P(int32(req.Msg.GetGateway().GetReplicas())), //nolint: gosec
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						labels.GatewayID: gatewayID,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							labels.GatewayID: gatewayID,
							labels.ManagedBy: krane,
						},
						Annotations: map[string]string{},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
						Containers: []corev1.Container{
							{
								Name:  "gateway",
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
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create gateway: %w", err))
	}

	// Create Service with owner reference to the Deployment
	service, err := k.clientset.CoreV1().
		Services(namespace).
		Create(ctx,
			// This implementation uses Deployments with ClusterIP services
			// for better scalability while maintaining internal accessibility via service name.
			// The service can be accessed within the cluster using: <service-name>.<namespace>.svc.cluster.local
			//nolint:exhaustruct
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "gw-svc-",
					Namespace:    namespace,
					Labels: map[string]string{
						labels.GatewayID: gatewayID,
						labels.ManagedBy: krane,
					},
					OwnerReferences: []metav1.OwnerReference{
						// Automatically clean up the service when the Deployment gets deleted
						//nolint:exhaustruct
						{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       deployment.Name,
							UID:        deployment.UID,
						},
					},
				},
				//nolint:exhaustruct
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
					Selector: map[string]string{
						labels.GatewayID: gatewayID,
					},
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
		k.logger.Info("Deleting deployment, because service creation failed")
		// Delete deployment
		// nolint: exhaustruct
		if rollbackErr := k.clientset.AppsV1().Deployments(namespace).Delete(ctx, deployment.Name, metav1.DeleteOptions{}); rollbackErr != nil {
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
