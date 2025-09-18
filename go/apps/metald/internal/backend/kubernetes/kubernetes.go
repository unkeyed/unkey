package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	"github.com/unkeyed/unkey/go/pkg/ptr"
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

	// Sanitize VM ID for Kubernetes resources (RFC 1123)
	sanitizedVMID := strings.ToLower(vmID)
	podName := sanitizedVMID
	serviceName := sanitizedVMID

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
	podAnnotations := map[string]string{}
	podLabels := map[string]string{
		"unkey.vm.id":      vmID,
		"unkey.managed.by": "metald",
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

	// This creates a job to kill the VM after the given TTL
	if useJob {
		// Create Job with TTL for auto-termination
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sanitizedVMID,
				Namespace: b.namespace,
				Labels: map[string]string{
					"unkey.vm.id":      vmID,
					"unkey.managed.by": "metald",
				},
			},
			Spec: batchv1.JobSpec{
				TTLSecondsAfterFinished: &b.ttlSeconds,              // Auto-cleanup after completion
				ActiveDeadlineSeconds:   ptr.P(int64(b.ttlSeconds)), // Max runtime
				Parallelism:             ptr.P(int32(1)),            // One pod
				Completions:             ptr.P(int32(1)),            // One completion
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

	// Create service to expose the VM with specific ClusterIP
	serviceSpec := corev1.ServiceSpec{
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
	}

	// Set specific ClusterIP if provided in network config
	if allocatedIP, ok := networkInfo["allocated_ip"]; ok && allocatedIP != "" {
		serviceSpec.ClusterIP = allocatedIP
		b.logger.Info("setting service ClusterIP to allocated IP",
			"vm_id", vmID,
			"cluster_ip", allocatedIP)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: b.namespace,
			Labels: map[string]string{
				"unkey.vm.id":      vmID,
				"unkey.managed.by": "metald",
			},
			Annotations: map[string]string{
				// Add annotation to help gateway discovery
				"unkey.deployment.id": networkInfo["deployment_id"],
			},
		},
		Spec: serviceSpec,
	}

	_, err := b.clientset.CoreV1().Services(b.namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		// Clean up pod/job if service creation fails
		if useJob {
			b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, sanitizedVMID, metav1.DeleteOptions{})
		} else {
			b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		}

		return "", fmt.Errorf("failed to create service: %w", err)
	}

	b.logger.Info("Kubernetes VM created",
		"vm_id", vmID,
		"pod_name", podName,
		"service_name", serviceName,
	)

	return vmID, nil
}

// DeleteVM removes a Kubernetes pod VM
func (b *Backend) DeleteVM(ctx context.Context, vmID string) error {
	// Sanitize VM ID for Kubernetes resources
	sanitizedVMID := strings.ToLower(vmID)

	// Delete service
	if err := b.clientset.CoreV1().Services(b.namespace).Delete(ctx, sanitizedVMID, metav1.DeleteOptions{}); err != nil {
		b.logger.Error("failed to delete service",
			"vm_id", vmID,
			"error", err)
	}

	// Try deleting as job first, then as pod
	if err := b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, sanitizedVMID, metav1.DeleteOptions{}); err != nil {
		// Not a job, try deleting as pod
		if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, sanitizedVMID, metav1.DeleteOptions{}); err != nil {
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

	// Sanitize VM ID for Kubernetes resources
	sanitizedVMID := strings.ToLower(vmID)

	// Try deleting as job first, then as pod
	if err := b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, sanitizedVMID, deleteOptions); err != nil {
		// Not a job, try deleting as pod
		if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, sanitizedVMID, deleteOptions); err != nil {
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
	// Sanitize VM ID for Kubernetes resources
	sanitizedVMID := strings.ToLower(vmID)

	// For Kubernetes, we delete the pod/job (it will be recreated if desired)
	if err := b.clientset.BatchV1().Jobs(b.namespace).Delete(ctx, sanitizedVMID, metav1.DeleteOptions{}); err != nil {
		// Not a job, try deleting as pod
		if err := b.clientset.CoreV1().Pods(b.namespace).Delete(ctx, sanitizedVMID, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete pod for reboot: %w", err)
		}
	}

	b.logger.Info("Kubernetes VM rebooted", "vm_id", vmID)
	return nil
}

// GetVMInfo retrieves current VM state and configuration
func (b *Backend) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	// Sanitize VM ID for Kubernetes resources
	sanitizedVMID := strings.ToLower(vmID)

	// First try to get the pod by the VM ID (direct pod creation)
	pod, err := b.clientset.CoreV1().Pods(b.namespace).Get(ctx, sanitizedVMID, metav1.GetOptions{})
	if err != nil {
		// If not found, try to find pod created by the job using labels
		pods, listErr := b.clientset.CoreV1().Pods(b.namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("unkey.vm.id=%s", vmID),
		})

		if listErr != nil {
			return nil, fmt.Errorf("failed to find pod for VM: %w", listErr)
		}

		if len(pods.Items) == 0 {
			return nil, fmt.Errorf("no pod found for VM %s", vmID)
		}
		// Use the first pod found (should only be one)
		pod = &pods.Items[0]
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
	return &types.VMMetrics{
		Timestamp:      time.Now(),
		DiskReadBytes:  0, // Not available from standard K8s metrics
		DiskWriteBytes: 0, // Not available from standard K8s metrics
		NetworkRxBytes: 0, // Not available from standard K8s metrics
		NetworkTxBytes: 0, // Not available from standard K8s metrics
	}, nil
}

// Ping checks if the Kubernetes API server is healthy and responsive
func (b *Backend) Ping(ctx context.Context) error {
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

// Type returns the backend type as a string for metrics
func (b *Backend) Type() string {
	return string(types.BackendTypeKubernetes)
}

// Helper methods
func readNamespaceFromServiceAccount() (string, error) {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}
