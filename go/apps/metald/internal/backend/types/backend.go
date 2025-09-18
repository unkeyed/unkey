package types

import (
	"context"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

// Backend defines the interface for hypervisor backends
type Backend interface {
	// CreateVM creates a new VM instance with the given configuration
	CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error)

	// DeleteVM removes a VM instance
	DeleteVM(ctx context.Context, vmID string) error

	// BootVM starts a created VM
	BootVM(ctx context.Context, vmID string) error

	// ShutdownVM gracefully stops a running VM
	ShutdownVM(ctx context.Context, vmID string) error

	// ShutdownVMWithOptions gracefully stops a running VM with force and timeout options
	ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error

	// PauseVM pauses a running VM
	PauseVM(ctx context.Context, vmID string) error

	// ResumeVM resumes a paused VM
	ResumeVM(ctx context.Context, vmID string) error

	// RebootVM restarts a running VM
	RebootVM(ctx context.Context, vmID string) error

	// GetVMInfo retrieves current VM state and configuration
	GetVMInfo(ctx context.Context, vmID string) (*VMInfo, error)

	// GetVMMetrics retrieves current VM resource usage metrics
	GetVMMetrics(ctx context.Context, vmID string) (*VMMetrics, error)

	// Ping checks if the backend is healthy and responsive
	Ping(ctx context.Context) error

	// Type returns the backend type as a string for metrics
	Type() string
}

// VMInfo contains VM state and configuration information
type VMInfo struct {
	Config *metaldv1.VmConfig
	State  metaldv1.VmState
}

// ListableVMInfo represents VM information for listing operations
type ListableVMInfo struct {
	ID     string
	State  metaldv1.VmState
	Config *metaldv1.VmConfig
}

// VMListProvider defines interface for backends that support VM listing
type VMListProvider interface {
	ListVMs() []ListableVMInfo
}

// BackendType represents the type of hypervisor backend
type BackendType string

const (
	BackendTypeFirecracker BackendType = "firecracker"
	BackendTypeDocker      BackendType = "docker"
	BackendTypeKubernetes  BackendType = "k8s"
)

func (s *BackendType) String() string {
	return string(*s)
}

// VMMetrics contains VM resource usage data for billing
type VMMetrics struct {
	Timestamp        time.Time `json:"timestamp"`
	CpuTimeNanos     int64     `json:"cpu_time_nanos"`
	MemoryUsageBytes int64     `json:"memory_usage_bytes"`
	DiskReadBytes    int64     `json:"disk_read_bytes"`
	DiskWriteBytes   int64     `json:"disk_write_bytes"`
	NetworkRxBytes   int64     `json:"network_rx_bytes"`
	NetworkTxBytes   int64     `json:"network_tx_bytes"`
}
