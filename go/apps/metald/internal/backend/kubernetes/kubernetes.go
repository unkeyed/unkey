package kubernetes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Compile-time interface checks
var _ types.Backend = (*Backend)(nil)
var _ types.VMListProvider = (*Backend)(nil)

// Backend implements types.Backend using Kubernetes pods
type Backend struct {
	logger     logging.Logger
	clientset  *kubernetes.Clientset
	namespace  string
	ttlSeconds int32 // TTL for auto-termination (0 = no TTL)
}

// New creates a new Kubernetes backend
func New(logger logging.Logger) (*Backend, error) {
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

	// Note: Metrics client would be created here if k8s.io/metrics was available
	// For now, metrics collection is disabled

	// Get namespace from service account or use default
	namespace := "default"
	if ns, err := readNamespaceFromServiceAccount(); err == nil && ns != "" {
		namespace = ns
	}

	return &Backend{
		logger:     logger.With("backend", "kubernetes"),
		clientset:  clientset,
		namespace:  namespace,
		ttlSeconds: 7200, // 2 hours default TTL for auto-cleanup
	}, nil
}

// CreateVM creates a new Kubernetes pod as a VM
func (b *Backend) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	// Use the provided VM ID from config
	vmID := config.Id
	if vmID == "" {
		return "", fmt.Errorf("VM ID is required")
	}

	b.logger.Info("creating Kubernetes VM",
		"vm_id", vmID,
		"image", config.Boot,
		"namespace", b.namespace)

	// Use boot configuration as container image
	imageName := config.Boot
	if imageName == "" {
		return "", fmt.Errorf("boot image is required")
	}

	// Parse network configuration
	var networkInfo map[string]string
	if config.NetworkConfig != "" {
		if err := json.Unmarshal([]byte(config.NetworkConfig), &networkInfo); err != nil {
			return "", fmt.Errorf("failed to parse network config: %w", err)
		}
	}

	// Determine namespace - use deployment-specific namespace for isolation
	targetNamespace := b.namespace
	if deploymentID, ok := networkInfo["deployment_id"]; ok && deploymentID != "" {
		// Sanitize deployment ID for Kubernetes namespace (RFC 1123)
		sanitizedDeploymentID := strings.ToLower(deploymentID)
		sanitizedDeploymentID = strings.ReplaceAll(sanitizedDeploymentID, "_", "-")
		targetNamespace = fmt.Sprintf("unkey-deployment-%s", sanitizedDeploymentID)

		// Ensure namespace exists
		if err := b.ensureNamespace(ctx, targetNamespace, deploymentID); err != nil {
			return "", fmt.Errorf("failed to ensure namespace: %w", err)
		}

		b.logger.Info("using deployment namespace",
			"vm_id", vmID,
			"namespace", targetNamespace,
			"deployment_id", deploymentID)
	}

	// Sanitize VM ID for Kubernetes resources (RFC 1123)
	sanitizedVMID := strings.ToLower(vmID)
	podName := sanitizedVMID
	serviceName := sanitizedVMID
	jobName := fmt.Sprintf("job-%s", sanitizedVMID)

	// Use Jobs with TTL for auto-cleanup
	useJob := b.ttlSeconds > 0

	// Convert VM config to Kubernetes resources
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	// Apply VM configuration to resources if specified
	if config.VcpuCount > 0 {
		cpuRequest := fmt.Sprintf("%dm", config.VcpuCount*100) // 100m per vCPU as request
		cpuLimit := fmt.Sprintf("%d", config.VcpuCount)        // 1 CPU per vCPU as limit
		resources.Requests[corev1.ResourceCPU] = resource.MustParse(cpuRequest)
		resources.Limits[corev1.ResourceCPU] = resource.MustParse(cpuLimit)
	}

	if config.MemorySizeMib > 0 {
		memRequest := fmt.Sprintf("%dMi", config.MemorySizeMib/2) // Half as request
		memLimit := fmt.Sprintf("%dMi", config.MemorySizeMib)     // Full as limit
		resources.Requests[corev1.ResourceMemory] = resource.MustParse(memRequest)
		resources.Limits[corev1.ResourceMemory] = resource.MustParse(memLimit)
	}

	// Create pod template with optional specific IP
	podLabels := map[string]string{
		"unkey.vm.id":      vmID,
		"unkey.managed.by": "metald",
	}
	podAnnotations := map[string]string{}

	// Note: Specific IP allocation via CNI annotations is not reliable across all K8s setups
	// For now, let Kubernetes assign IPs naturally and rely on Service discovery
	if allocatedIP, ok := networkInfo["allocated_ip"]; ok && allocatedIP != "" {
		b.logger.Info("metald allocated IP (for reference only)",
			"vm_id", vmID,
			"allocated_ip", allocatedIP,
			"note", "using K8s service discovery instead of static IP")
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      podLabels,
			Annotations: podAnnotations,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever, // Required for Jobs
			Containers: []corev1.Container{
				{
					Name:  "vm",
					Image: imageName,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: resources,
				},
			},
		},
	}

	if useJob {
		// Create Job with TTL for auto-termination
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: targetNamespace,
				Labels: map[string]string{
					"unkey.vm.id":      vmID,
					"unkey.managed.by": "metald",
				},
			},
			Spec: batchv1.JobSpec{
				TTLSecondsAfterFinished: &b.ttlSeconds,            // Auto-cleanup after completion
				ActiveDeadlineSeconds:   ptr(int64(b.ttlSeconds)), // Max runtime
				Parallelism:             ptr(int32(1)),            // One pod
				Completions:             ptr(int32(1)),            // One completion
				Template:                podTemplate,
			},
		}

		_, err := b.clientset.BatchV1().Jobs(targetNamespace).Create(ctx, job, metav1.CreateOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to create job: %w", err)
		}
	} else {
		// Create regular Pod
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: targetNamespace,
				Labels: map[string]string{
					"unkey.vm.id":      vmID,
					"unkey.managed.by": "metald",
				},
			},
			Spec: podTemplate.Spec,
		}

		_, err := b.clientset.CoreV1().Pods(targetNamespace).Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to create pod: %w", err)
		}
	}

	// Create service to expose the VM
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: targetNamespace,
			Labels: map[string]string{
				"unkey.vm.id":      vmID,
				"unkey.managed.by": "metald",
			},
			Annotations: map[string]string{
				// Add annotation to help gateway discovery
				"unkey.deployment.id": networkInfo["deployment_id"],
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
			Selector: map[string]string{
				"unkey.vm.id": vmID,
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

	_, err := b.clientset.CoreV1().Services(targetNamespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		// Clean up pod/job if service creation fails
		if useJob {
			b.clientset.BatchV1().Jobs(targetNamespace).Delete(ctx, jobName, metav1.DeleteOptions{})
		} else {
			b.clientset.CoreV1().Pods(targetNamespace).Delete(ctx, podName, metav1.DeleteOptions{})
		}
		return "", fmt.Errorf("failed to create service: %w", err)
	}

	b.logger.Info("Kubernetes VM created",
		"vm_id", vmID,
		"pod_name", podName,
		"service_name", serviceName)

	return vmID, nil
}

// DeleteVM removes a Kubernetes pod VM
func (b *Backend) DeleteVM(ctx context.Context, vmID string) error {

	// Delete service
	if err := b.clientset.CoreV1().Services(b.namespace).Delete(ctx, vmID, metav1.DeleteOptions{}); err != nil {
		b.logger.Error("failed to delete service",
			"vm_id", vmID,
			"error", err)
	}

	// Try deleting as job first, then as pod
	jobName := fmt.Sprintf("job-%s", vmID)
	if err := b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, jobName, metav1.DeleteOptions{}); err != nil {
		// Not a job, try deleting as pod
		if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, vmID, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	b.logger.Info("Kubernetes VM deleted", "vm_id", vmID)
	return nil
}

// BootVM starts a Kubernetes pod VM (no-op since pods auto-start)
func (b *Backend) BootVM(ctx context.Context, vmID string) error {
	// Kubernetes pods start automatically, nothing to do
	b.logger.Info("Kubernetes VM boot requested", "vm_id", vmID)
	return nil
}

// ShutdownVM gracefully stops a Kubernetes pod VM
func (b *Backend) ShutdownVM(ctx context.Context, vmID string) error {
	return b.ShutdownVMWithOptions(ctx, vmID, false, 30)
}

// ShutdownVMWithOptions gracefully stops a Kubernetes pod VM with options
func (b *Backend) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	gracePeriodSeconds := int64(timeoutSeconds)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	if force {
		// Immediate deletion
		gracePeriodSeconds = 0
		deleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	}

	// Try deleting as job first, then as pod
	jobName := fmt.Sprintf("job-%s", vmID)
	if err := b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, jobName, deleteOptions); err != nil {
		// Not a job, try deleting as pod
		if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, vmID, deleteOptions); err != nil {
			return fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	b.logger.Info("Kubernetes VM shutdown", "vm_id", vmID, "force", force)
	return nil
}

// PauseVM pauses a Kubernetes pod VM (not supported - return error)
func (b *Backend) PauseVM(ctx context.Context, vmID string) error {
	return fmt.Errorf("pause operation not supported by Kubernetes backend")
}

// ResumeVM resumes a paused Kubernetes pod VM (not supported - return error)
func (b *Backend) ResumeVM(ctx context.Context, vmID string) error {
	return fmt.Errorf("resume operation not supported by Kubernetes backend")
}

// RebootVM restarts a Kubernetes pod VM
func (b *Backend) RebootVM(ctx context.Context, vmID string) error {
	// For Kubernetes, we delete the pod/job (it will be recreated if desired)
	jobName := fmt.Sprintf("job-%s", vmID)
	if err := b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, jobName, metav1.DeleteOptions{}); err != nil {
		// Not a job, try deleting as pod
		if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, vmID, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete pod for reboot: %w", err)
		}
	}

	b.logger.Info("Kubernetes VM rebooted", "vm_id", vmID)
	return nil
}

// GetVMInfo retrieves current VM state and configuration
func (b *Backend) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	// Try to get pod info first
	pod, err := b.clientset.CoreV1().Pods(b.namespace).Get(ctx, vmID, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod info: %w", err)
	}

	// Determine state based on pod phase
	state := metaldv1.VmState_VM_STATE_UNSPECIFIED
	switch pod.Status.Phase {
	case "Pending":
		state = metaldv1.VmState_VM_STATE_CREATED
	case "Running":
		state = metaldv1.VmState_VM_STATE_RUNNING
	case "Succeeded", "Failed":
		state = metaldv1.VmState_VM_STATE_SHUTDOWN
	}

	// Reconstruct config from pod spec
	config := &metaldv1.VmConfig{
		Id: vmID,
	}
	if len(pod.Spec.Containers) > 0 {
		config.Boot = pod.Spec.Containers[0].Image
	}

	return &types.VMInfo{
		Config: config,
		State:  state,
	}, nil
}

// GetVMMetrics retrieves current VM resource usage metrics
func (b *Backend) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	// Kubernetes metrics require metrics-server, which may not be available
	// Return basic metrics for now
	return &types.VMMetrics{
		Timestamp: time.Now(),
		DiskReadBytes:    0, // Not available from standard K8s metrics
		DiskWriteBytes:   0, // Not available from standard K8s metrics
		NetworkRxBytes:   0, // Not available from standard K8s metrics
		NetworkTxBytes:   0, // Not available from standard K8s metrics
	}, nil
}

// Ping checks if the Kubernetes API server is healthy and responsive
func (b *Backend) Ping(ctx context.Context) error {
	// Simple API server health check
	_, err := b.clientset.CoreV1().Namespaces().Get(ctx, b.namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Kubernetes API server ping failed: %w", err)
	}
	return nil
}

// ListVMs returns a list of all VMs managed by this backend
func (b *Backend) ListVMs() []types.ListableVMInfo {
	ctx := context.Background()
	pods, err := b.clientset.CoreV1().Pods(b.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "unkey.managed.by=metald",
	})
	if err != nil {
		b.logger.Error("failed to list pods", "error", err)
		return []types.ListableVMInfo{}
	}

	vms := make([]types.ListableVMInfo, 0, len(pods.Items))
	for _, pod := range pods.Items {
		vmID := pod.Labels["unkey.vm.id"]
		if vmID == "" {
			vmID = pod.Name
		}

		// Determine state from pod phase
		state := metaldv1.VmState_VM_STATE_CREATED
		switch pod.Status.Phase {
		case "Running":
			state = metaldv1.VmState_VM_STATE_RUNNING
		case "Succeeded", "Failed":
			state = metaldv1.VmState_VM_STATE_SHUTDOWN
		}

		config := &metaldv1.VmConfig{
			Id: vmID,
		}
		if len(pod.Spec.Containers) > 0 {
			config.Boot = pod.Spec.Containers[0].Image
		}

		vms = append(vms, types.ListableVMInfo{
			ID:     vmID,
			State:  state,
			Config: config,
		})
	}
	return vms
}

// Helper methods

func (b *Backend) sanitizeK8sNames(vmID string) (podName, serviceName string) {
	// Generate a short hash suffix for uniqueness (6 hex chars)
	hash := sha256.Sum256([]byte(vmID))
	hashSuffix := fmt.Sprintf("%x", hash[:3]) // 3 bytes = 6 hex chars

	// Replace any character not in [a-z0-9-] with a hyphen
	invalidCharsRegex := regexp.MustCompile(`[^a-z0-9-]+`)
	sanitized := invalidCharsRegex.ReplaceAllString(strings.ToLower(vmID), "-")

	// Collapse consecutive hyphens to a single hyphen
	multiHyphenRegex := regexp.MustCompile(`-+`)
	sanitized = multiHyphenRegex.ReplaceAllString(sanitized, "-")

	// Trim leading and trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// If empty after sanitization, use a default
	if sanitized == "" {
		sanitized = "vm"
	}

	// Calculate max lengths for the core ID part
	// Format: "unkey-vm-<core>-<hash>" (pod) and "unkey-svc-<core>-<hash>" (service)
	// Max total length is 63 chars
	maxPodCore := 63 - len("unkey-vm-") - 1 - 6      // = 46 chars
	maxServiceCore := 63 - len("unkey-svc-") - 1 - 6 // = 44 chars

	// Use the smaller limit to ensure both names are valid
	maxCore := min(maxPodCore, maxServiceCore)
	if len(sanitized) > maxCore {
		sanitized = sanitized[:maxCore]
		// Trim any trailing hyphen from truncation
		sanitized = strings.TrimRight(sanitized, "-")
	}

	// Build the final names with hash suffix
	podName = fmt.Sprintf("unkey-vm-%s-%s", sanitized, hashSuffix)
	serviceName = fmt.Sprintf("unkey-svc-%s-%s", sanitized, hashSuffix)

	// Final validation: ensure names start and end with alphanumeric
	podName = strings.Trim(podName, "-")
	serviceName = strings.Trim(serviceName, "-")

	return podName, serviceName
}

func readNamespaceFromServiceAccount() (string, error) {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func ptr[T any](v T) *T {
	return &v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ensureNamespace creates a namespace for the deployment if it doesn't exist
func (b *Backend) ensureNamespace(ctx context.Context, namespace, deploymentID string) error {
	// Check if namespace already exists
	_, err := b.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists
		return nil
	}

	// Create namespace with labels for identification and management
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"unkey.deployment.id": deploymentID,
				"unkey.managed.by":    "metald",
				"unkey.type":          "deployment",
			},
		},
	}

	_, err = b.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}

	b.logger.Info("created deployment namespace",
		"namespace", namespace,
		"deployment_id", deploymentID)

	return nil
}
