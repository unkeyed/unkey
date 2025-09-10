package fallbacks

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// K8sBackend implements DeploymentBackend using Kubernetes pods
type K8sBackend struct {
	logger      logging.Logger
	clientset   *kubernetes.Clientset
	namespace   string
	deployments map[string]*k8sDeployment
	mutex       sync.RWMutex
}

type k8sDeployment struct {
	DeploymentID   string
	DeploymentName string
	ServiceName    string
	VMIDs          []string
	Image          string
	CreatedAt      time.Time
}

// NewK8sBackend creates a new Kubernetes backend
func NewK8sBackend(logger logging.Logger) (*K8sBackend, error) {
	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Get namespace from service account or use default
	namespace := "default"
	if ns, err := readNamespaceFromServiceAccount(); err == nil && ns != "" {
		namespace = ns
	}

	return &K8sBackend{
		logger:      logger.With("backend", "k8s"),
		clientset:   clientset,
		namespace:   namespace,
		deployments: make(map[string]*k8sDeployment),
	}, nil
}

// CreateDeployment creates Kubernetes pods for the deployment
func (k *K8sBackend) CreateDeployment(ctx context.Context, deploymentID string, image string, vmCount int32) ([]string, error) {
	k.logger.Info("creating Kubernetes deployment",
		"deployment_id", deploymentID,
		"image", image,
		"vm_count", vmCount,
		"namespace", k.namespace)

	// Generate VM IDs
	vmIDs := make([]string, vmCount)
	for i := int32(0); i < vmCount; i++ {
		vmIDs[i] = uid.New("vm")
	}

	deploymentName := fmt.Sprintf("unkey-%s", deploymentID)
	serviceName := fmt.Sprintf("unkey-svc-%s", deploymentID)

	// Create deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"unkey.deployment.id": deploymentID,
				"unkey.managed.by":    "ctrl-fallback",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &vmCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"unkey.deployment.id": deploymentID,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"unkey.deployment.id": deploymentID,
						"unkey.managed.by":    "ctrl-fallback",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := k.clientset.AppsV1().Deployments(k.namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create service to expose the deployment
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"unkey.deployment.id": deploymentID,
				"unkey.managed.by":    "ctrl-fallback",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"unkey.deployment.id": deploymentID,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	_, err = k.clientset.CoreV1().Services(k.namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		// Clean up deployment if service creation fails
		k.clientset.AppsV1().Deployments(k.namespace).Delete(ctx, deploymentName, metav1.DeleteOptions{})
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Store deployment info
	k.mutex.Lock()
	k.deployments[deploymentID] = &k8sDeployment{
		DeploymentID:   deploymentID,
		DeploymentName: deploymentName,
		ServiceName:    serviceName,
		VMIDs:          vmIDs,
		Image:          image,
		CreatedAt:      time.Now(),
	}
	k.mutex.Unlock()

	k.logger.Info("Kubernetes deployment created",
		"deployment_id", deploymentID,
		"deployment_name", deploymentName,
		"service_name", serviceName,
		"vm_ids", vmIDs)

	return vmIDs, nil
}

// GetDeploymentStatus returns the status of deployment VMs
func (k *K8sBackend) GetDeploymentStatus(ctx context.Context, deploymentID string) ([]*metaldv1.GetDeploymentResponse_Vm, error) {
	k.mutex.RLock()
	deploymentInfo, exists := k.deployments[deploymentID]
	k.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("deployment %s not found", deploymentID)
	}

	// Get deployment status
	deployment, err := k.clientset.AppsV1().Deployments(k.namespace).Get(ctx, deploymentInfo.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get service to find the cluster IP
	service, err := k.clientset.CoreV1().Services(k.namespace).Get(ctx, deploymentInfo.ServiceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Create VM responses
	vms := make([]*metaldv1.GetDeploymentResponse_Vm, 0, len(deploymentInfo.VMIDs))

	// Determine state based on deployment status
	state := metaldv1.VmState_VM_STATE_UNSPECIFIED
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		state = metaldv1.VmState_VM_STATE_RUNNING
	} else if deployment.Status.ReadyReplicas > 0 {
		state = metaldv1.VmState_VM_STATE_RUNNING // Partially running
	} else {
		state = metaldv1.VmState_VM_STATE_CREATED
	}

	// For each VM ID, create a response
	for _, vmID := range deploymentInfo.VMIDs {
		vms = append(vms, &metaldv1.GetDeploymentResponse_Vm{
			Id:    vmID,
			State: state,
			Host:  service.Spec.ClusterIP,
			Port:  8080,
		})
	}

	return vms, nil
}

// DeleteDeployment removes the Kubernetes deployment and service
func (k *K8sBackend) DeleteDeployment(ctx context.Context, deploymentID string) error {
	k.mutex.Lock()
	deploymentInfo, exists := k.deployments[deploymentID]
	if exists {
		delete(k.deployments, deploymentID)
	}
	k.mutex.Unlock()

	if !exists {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}

	// Delete service
	if err := k.clientset.CoreV1().Services(k.namespace).Delete(ctx, deploymentInfo.ServiceName, metav1.DeleteOptions{}); err != nil {
		k.logger.Error("failed to delete service",
			"service_name", deploymentInfo.ServiceName,
			"error", err)
	}

	// Delete deployment
	if err := k.clientset.AppsV1().Deployments(k.namespace).Delete(ctx, deploymentInfo.DeploymentName, metav1.DeleteOptions{}); err != nil {
		k.logger.Error("failed to delete deployment",
			"deployment_name", deploymentInfo.DeploymentName,
			"error", err)
		return err
	}

	k.logger.Info("Kubernetes deployment deleted",
		"deployment_id", deploymentID,
		"deployment_name", deploymentInfo.DeploymentName,
		"service_name", deploymentInfo.ServiceName)

	return nil
}

// Type returns the backend type
func (k *K8sBackend) Type() string {
	return "k8s"
}

// Helper function to read namespace from service account
func readNamespaceFromServiceAccount() (string, error) {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
