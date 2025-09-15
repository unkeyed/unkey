package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
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
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Compile-time interface checks
var _ types.Backend = (*Backend)(nil)
var _ types.VMListProvider = (*Backend)(nil)

// Backend implements types.Backend using Kubernetes pods
type Backend struct {
	logger     logging.Logger
	clientset  *kubernetes.Clientset
	namespace  string
	vms        map[string]*vmInfo
	mutex      sync.RWMutex
	ttlSeconds int32 // TTL for auto-termination (0 = no TTL)
}

type vmInfo struct {
	vmID        string
	podName     string
	serviceName string
	config      *metaldv1.VmConfig
	state       metaldv1.VmState
	createdAt   time.Time
	useJob      bool
	jobName     string
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
		vms:        make(map[string]*vmInfo),
		ttlSeconds: 7200, // 2 hours default TTL for auto-cleanup
	}, nil
}

// CreateVM creates a new Kubernetes pod as a VM
func (b *Backend) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	vmID := uid.New("vm")
	b.logger.Info("creating Kubernetes VM",
		"vm_id", vmID,
		"image", config.Boot,
		"namespace", b.namespace)

	// Use boot configuration as container image
	imageName := config.Boot
	if imageName == "" {
		return "", fmt.Errorf("boot image is required")
	}

	// Generate RFC1123-compliant names
	podName, serviceName := b.sanitizeK8sNames(vmID)
	jobName := fmt.Sprintf("job-%s", podName)

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

	// Create pod template
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"unkey.vm.id":      vmID,
				"unkey.managed.by": "metald",
			},
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
				Namespace: b.namespace,
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

		_, err := b.clientset.BatchV1().Jobs(b.namespace).Create(ctx, job, metav1.CreateOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to create job: %w", err)
		}
	} else {
		// Create regular Pod
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: b.namespace,
				Labels: map[string]string{
					"unkey.vm.id":      vmID,
					"unkey.managed.by": "metald",
				},
			},
			Spec: podTemplate.Spec,
		}

		_, err := b.clientset.CoreV1().Pods(b.namespace).Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to create pod: %w", err)
		}
	}

	// Create service to expose the VM
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: b.namespace,
			Labels: map[string]string{
				"unkey.vm.id":      vmID,
				"unkey.managed.by": "metald",
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

	_, err := b.clientset.CoreV1().Services(b.namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		// Clean up pod/job if service creation fails
		if useJob {
			b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, jobName, metav1.DeleteOptions{})
		} else {
			b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		}
		return "", fmt.Errorf("failed to create service: %w", err)
	}

	// Store VM info
	b.mutex.Lock()
	b.vms[vmID] = &vmInfo{
		vmID:        vmID,
		podName:     podName,
		serviceName: serviceName,
		config:      config,
		state:       metaldv1.VmState_VM_STATE_CREATED,
		createdAt:   time.Now(),
		useJob:      useJob,
		jobName:     jobName,
	}
	b.mutex.Unlock()

	b.logger.Info("Kubernetes VM created",
		"vm_id", vmID,
		"pod_name", podName,
		"service_name", serviceName)

	return vmID, nil
}

// DeleteVM removes a Kubernetes pod VM
func (b *Backend) DeleteVM(ctx context.Context, vmID string) error {
	b.mutex.Lock()
	vm, exists := b.vms[vmID]
	if exists {
		delete(b.vms, vmID)
	}
	b.mutex.Unlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	// Delete service
	if err := b.clientset.CoreV1().Services(b.namespace).Delete(ctx, vm.serviceName, metav1.DeleteOptions{}); err != nil {
		b.logger.Error("failed to delete service",
			"vm_id", vmID,
			"service_name", vm.serviceName,
			"error", err)
	}

	// Delete pod or job
	if vm.useJob {
		if err := b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, vm.jobName, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete job: %w", err)
		}
	} else {
		if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, vm.podName, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	b.logger.Info("Kubernetes VM deleted", "vm_id", vmID)
	return nil
}

// BootVM starts a Kubernetes pod VM (no-op since pods auto-start)
func (b *Backend) BootVM(ctx context.Context, vmID string) error {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	// Kubernetes pods start automatically, so we just update state if needed
	b.mutex.Lock()
	vm.state = metaldv1.VmState_VM_STATE_RUNNING
	b.mutex.Unlock()

	b.logger.Info("Kubernetes VM boot requested", "vm_id", vmID)
	return nil
}

// ShutdownVM gracefully stops a Kubernetes pod VM
func (b *Backend) ShutdownVM(ctx context.Context, vmID string) error {
	return b.ShutdownVMWithOptions(ctx, vmID, false, 30)
}

// ShutdownVMWithOptions gracefully stops a Kubernetes pod VM with options
func (b *Backend) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	gracePeriodSeconds := int64(timeoutSeconds)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	if force {
		// Immediate deletion
		gracePeriodSeconds = 0
		deleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	}

	// Delete the pod (or job will handle pod deletion)
	if vm.useJob {
		// For jobs, delete the job which will stop the pod
		if err := b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, vm.jobName, deleteOptions); err != nil {
			return fmt.Errorf("failed to delete job: %w", err)
		}
	} else {
		// For standalone pods, delete the pod directly
		if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, vm.podName, deleteOptions); err != nil {
			return fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	// Update state
	b.mutex.Lock()
	vm.state = metaldv1.VmState_VM_STATE_SHUTDOWN
	b.mutex.Unlock()

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
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	// For Kubernetes, reboot means delete and recreate the pod
	// This is simplified - in production you might want to use a deployment with rolling restart
	if vm.useJob {
		// Cannot restart a job - would need to delete and recreate
		return fmt.Errorf("reboot not supported for job-based VMs, use delete and recreate")
	}

	// Delete the pod (it will be recreated if managed by a deployment)
	if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, vm.podName, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete pod for reboot: %w", err)
	}

	b.logger.Info("Kubernetes VM rebooted", "vm_id", vmID)
	return nil
}

// GetVMInfo retrieves current VM state and configuration
func (b *Backend) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	// Get current pod state
	var state metaldv1.VmState
	if vm.useJob {
		// Check job status
		job, err := b.clientset.BatchV1().Jobs(b.namespace).Get(ctx, vm.jobName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get job: %w", err)
		}

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
		// Check pod status
		pod, err := b.clientset.CoreV1().Pods(b.namespace).Get(ctx, vm.podName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get pod: %w", err)
		}

		switch pod.Status.Phase {
		case corev1.PodRunning:
			state = metaldv1.VmState_VM_STATE_RUNNING
		case corev1.PodPending:
			state = metaldv1.VmState_VM_STATE_CREATED
		case corev1.PodSucceeded, corev1.PodFailed:
			state = metaldv1.VmState_VM_STATE_SHUTDOWN
		default:
			state = metaldv1.VmState_VM_STATE_UNSPECIFIED
		}
	}

	// Update cached state
	b.mutex.Lock()
	vm.state = state
	b.mutex.Unlock()

	return &types.VMInfo{
		Config: vm.config,
		State:  state,
	}, nil
}

// GetVMMetrics retrieves current VM resource usage metrics
func (b *Backend) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	b.mutex.RLock()
	_, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	// Note: Full metrics would require the metrics server client
	// For now, return basic metrics with timestamp
	return &types.VMMetrics{
		Timestamp: time.Now(),
		// Note: Kubernetes metrics server integration would provide:
		// - CPU usage from container stats
		// - Memory usage from container stats
		// Disk and network I/O would need additional monitoring (e.g., Prometheus)
		CpuTimeNanos:     0, // Would be populated from metrics server
		MemoryUsageBytes: 0, // Would be populated from metrics server
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
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	vms := make([]types.ListableVMInfo, 0, len(b.vms))
	for _, vm := range b.vms {
		vms = append(vms, types.ListableVMInfo{
			ID:     vm.vmID,
			State:  vm.state,
			Config: vm.config,
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
