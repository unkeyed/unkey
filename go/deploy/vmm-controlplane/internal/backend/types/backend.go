package types

import (
	"context"

	vmmv1 "vmm-controlplane/gen/vmm/v1"
)

// Backend defines the interface for hypervisor backends
// AIDEV-NOTE: This interface abstracts VM operations for all hypervisor types
type Backend interface {
	// CreateVM creates a new VM instance with the given configuration
	CreateVM(ctx context.Context, config *vmmv1.VmConfig) (string, error)

	// DeleteVM removes a VM instance
	DeleteVM(ctx context.Context, vmID string) error

	// BootVM starts a created VM
	BootVM(ctx context.Context, vmID string) error

	// ShutdownVM gracefully stops a running VM
	ShutdownVM(ctx context.Context, vmID string) error

	// PauseVM pauses a running VM
	PauseVM(ctx context.Context, vmID string) error

	// ResumeVM resumes a paused VM
	ResumeVM(ctx context.Context, vmID string) error

	// RebootVM restarts a running VM
	RebootVM(ctx context.Context, vmID string) error

	// GetVMInfo retrieves current VM state and configuration
	GetVMInfo(ctx context.Context, vmID string) (*VMInfo, error)

	// Ping checks if the backend is healthy and responsive
	Ping(ctx context.Context) error
}

// VMInfo contains VM state and configuration information
type VMInfo struct {
	Config *vmmv1.VmConfig
	State  vmmv1.VmState
}

// BackendType represents the type of hypervisor backend
type BackendType string

const (
	BackendTypeCloudHypervisor BackendType = "cloudhypervisor"
	BackendTypeFirecracker     BackendType = "firecracker"
)
