package backends

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// K8sBackend implements DeploymentBackend using Kubernetes pods
type K8sBackend struct {
	logger      logging.Logger
	clientset   *kubernetes.Clientset
	namespace   string
	deployments map[string]*k8sDeployment
	mutex       sync.RWMutex
	ttlSeconds  int32 // TTL for auto-termination (0 = no TTL)
}

type k8sDeployment struct {
	DeploymentID   string
	DeploymentName string
	JobName        string // Job name for TTL-enabled deployments
	ServiceName    string
	VMIDs          []string
	Image          string
	CreatedAt      time.Time
	UseJob         bool // Whether this deployment uses a Job (for TTL) or Deployment
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
		ttlSeconds:  7200, // 2 hours default TTL for auto-cleanup
	}, nil
}

// CreateDeployment creates Kubernetes pods for the deployment
func (k *K8sBackend) CreateDeployment(ctx context.Context, deploymentID string, image string, vmCount uint32) ([]string, error) {
	k.logger.Info("creating Kubernetes deployment",
		"deployment_id", deploymentID,
		"image", image,
		"vm_count", vmCount,
		"namespace", k.namespace)

	vmIDs := make([]string, vmCount)
	for i := range vmCount {
		vmIDs[i] = uid.New("vm")
	}

	// Sanitize deployment ID for Kubernetes RFC 1123 compliance
	deploymentName, serviceName := k.sanitizeK8sNames(deploymentID)
	jobName := fmt.Sprintf("job-%s", deploymentName)

	// Decide whether to use Job (with TTL) or Deployment
	useJob := k.ttlSeconds > 0

	if useJob {
		// Create Job with TTL for auto-termination
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: k.namespace,
				Labels: map[string]string{
					"unkey.deployment.id": deploymentID,
					"unkey.managed.by":    "ctrl-fallback",
				},
			},
			Spec: batchv1.JobSpec{
				TTLSecondsAfterFinished: &k.ttlSeconds,              // Auto-cleanup after completion
				ActiveDeadlineSeconds:   ptr.P(int64(k.ttlSeconds)), // Max runtime
				Parallelism:             ptr.P(int32(vmCount)),
				Completions:             ptr.P(int32(vmCount)),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"unkey.deployment.id": deploymentID,
							"unkey.managed.by":    "ctrl-fallback",
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever, // Required for Jobs
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

		_, err := k.clientset.BatchV1().Jobs(k.namespace).Create(ctx, job, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create job: %w", err)
		}
	} else {
		// Create regular Deployment without TTL
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
				Replicas: ptr.P(int32(vmCount)),
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
			Type: corev1.ServiceTypeNodePort,
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

	_, err := k.clientset.CoreV1().Services(k.namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		// Clean up deployment/job if service creation fails
		if useJob {
			k.clientset.BatchV1().Jobs(k.namespace).Delete(ctx, jobName, metav1.DeleteOptions{})
		} else {
			k.clientset.AppsV1().Deployments(k.namespace).Delete(ctx, deploymentName, metav1.DeleteOptions{})
		}
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Store deployment info
	k.mutex.Lock()
	k.deployments[deploymentID] = &k8sDeployment{
		DeploymentID:   deploymentID,
		DeploymentName: deploymentName,
		JobName:        jobName,
		ServiceName:    serviceName,
		VMIDs:          vmIDs,
		Image:          image,
		CreatedAt:      time.Now(),
		UseJob:         useJob,
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

	// Get deployment/job status
	var state metaldv1.VmState
	if deploymentInfo.UseJob {
		job, err := k.clientset.BatchV1().Jobs(k.namespace).Get(ctx, deploymentInfo.JobName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get job: %w", err)
		}

		// Determine state based on job status
		if job.Status.Succeeded > 0 {
			state = metaldv1.VmState_VM_STATE_RUNNING
		} else if job.Status.Failed > 0 {
			state = metaldv1.VmState_VM_STATE_SHUTDOWN
		} else if job.Status.Active > 0 {
			state = metaldv1.VmState_VM_STATE_RUNNING
		} else {
			state = metaldv1.VmState_VM_STATE_CREATED
		}
	} else {
		deployment, err := k.clientset.AppsV1().Deployments(k.namespace).Get(ctx, deploymentInfo.DeploymentName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment: %w", err)
		}

		// Determine state based on deployment status
		if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
			state = metaldv1.VmState_VM_STATE_RUNNING
		} else if deployment.Status.ReadyReplicas > 0 {
			state = metaldv1.VmState_VM_STATE_RUNNING // Partially running
		} else {
			state = metaldv1.VmState_VM_STATE_CREATED
		}
	}

	// Get service to find the cluster IP
	service, err := k.clientset.CoreV1().Services(k.namespace).Get(ctx, deploymentInfo.ServiceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Create VM responses
	vms := make([]*metaldv1.GetDeploymentResponse_Vm, 0, len(deploymentInfo.VMIDs))

	// Always use cluster IP and container port for backend communication
	host := service.Spec.ClusterIP
	port := int32(8080) // Always use container port for backend service calls

	// For each VM ID, create a response
	for _, vmID := range deploymentInfo.VMIDs {
		vm := &metaldv1.GetDeploymentResponse_Vm{
			Id:    vmID,
			State: state,
			Host:  host,
			Port:  uint32(port),
		}

		vms = append(vms, vm)
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

	// Delete deployment or job
	if deploymentInfo.UseJob {
		if err := k.clientset.BatchV1().Jobs(k.namespace).Delete(ctx, deploymentInfo.JobName, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete job %s: %w", deploymentInfo.JobName, err)
		}
	} else {
		if err := k.clientset.AppsV1().Deployments(k.namespace).Delete(ctx, deploymentInfo.DeploymentName, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete deployment %s: %w", deploymentInfo.DeploymentName, err)
		}
	}

	k.logger.Info("Kubernetes deployment deleted",
		"deployment_id", deploymentID,
		"deployment_name", deploymentInfo.DeploymentName,
		"service_name", deploymentInfo.ServiceName)

	return nil
}

// Type returns the backend type
func (k *K8sBackend) Type() string {
	return BackendTypeK8s
}

// sanitizeK8sNames generates RFC1123-compliant names for Kubernetes resources
// It ensures names are lowercase, alphanumeric with hyphens, max 63 chars, and unique
func (k *K8sBackend) sanitizeK8sNames(deploymentID string) (deploymentName, serviceName string) {
	// Generate a short hash suffix for uniqueness (6 hex chars)
	hash := sha256.Sum256([]byte(deploymentID))
	hashSuffix := fmt.Sprintf("%x", hash[:3]) // 3 bytes = 6 hex chars

	// Replace any character not in [a-z0-9-] with a hyphen
	// This regex will be compiled once at startup for efficiency
	invalidCharsRegex := regexp.MustCompile(`[^a-z0-9-]+`)
	sanitized := invalidCharsRegex.ReplaceAllString(strings.ToLower(deploymentID), "-")

	// Collapse consecutive hyphens to a single hyphen
	multiHyphenRegex := regexp.MustCompile(`-+`)
	sanitized = multiHyphenRegex.ReplaceAllString(sanitized, "-")

	// Trim leading and trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// If empty after sanitization, use a default
	if sanitized == "" {
		sanitized = "deployment"
	}

	// Calculate max lengths for the core ID part
	// Format: "unkey-<core>-<hash>" (deployment) and "unkey-svc-<core>-<hash>" (service)
	// Max total length is 63 chars
	// Reserve: "unkey-" (6) + "-" (1) + hash (6) = 13 chars for deployment
	// Reserve: "unkey-svc-" (10) + "-" (1) + hash (6) = 17 chars for service
	maxDeploymentCore := 63 - 13 // = 50 chars
	maxServiceCore := 63 - 17    // = 46 chars

	// Use the smaller limit to ensure both names are valid
	maxCore := min(maxDeploymentCore, maxServiceCore)
	if len(sanitized) > maxCore {
		sanitized = sanitized[:maxCore]
		// Trim any trailing hyphen from truncation
		sanitized = strings.TrimRight(sanitized, "-")
	}

	// Build the final names with hash suffix
	deploymentName = fmt.Sprintf("unkey-%s-%s", sanitized, hashSuffix)
	serviceName = fmt.Sprintf("unkey-svc-%s-%s", sanitized, hashSuffix)

	// Final validation: ensure names start and end with alphanumeric
	// (should already be the case, but double-check)
	deploymentName = strings.Trim(deploymentName, "-")
	serviceName = strings.Trim(serviceName, "-")

	return deploymentName, serviceName
}

// Helper function to read namespace from service account
func readNamespaceFromServiceAccount() (string, error) {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
