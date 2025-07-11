package client

import (
	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
)

// AIDEV-NOTE: Type definitions for metald client requests and responses
// These provide a cleaner interface while wrapping the underlying protobuf types

// CreateVMRequest represents a request to create a new virtual machine
type CreateVMRequest struct {
	// VMID is the unique identifier for the VM (optional, will be generated if empty)
	VMID string

	// Config is the VM configuration including CPU, memory, storage, and network
	Config *vmprovisionerv1.VmConfig
}

// CreateVMResponse represents the response from creating a virtual machine
type CreateVMResponse struct {
	// VMID is the unique identifier of the created VM
	VMID string

	// State is the current state of the VM after creation
	State vmprovisionerv1.VmState
}

// BootVMResponse represents the response from booting a virtual machine
type BootVMResponse struct {
	// Success indicates if the boot operation was successful
	Success bool

	// State is the current state of the VM after boot attempt
	State vmprovisionerv1.VmState
}

// ShutdownVMRequest represents a request to shutdown a virtual machine
type ShutdownVMRequest struct {
	// VMID is the unique identifier of the VM to shutdown
	VMID string

	// Force indicates whether to force shutdown if graceful shutdown fails
	Force bool

	// TimeoutSeconds is the timeout for graceful shutdown before forcing (0 = no timeout)
	TimeoutSeconds uint32
}

// ShutdownVMResponse represents the response from shutting down a virtual machine
type ShutdownVMResponse struct {
	// Success indicates if the shutdown operation was successful
	Success bool

	// State is the current state of the VM after shutdown attempt
	State vmprovisionerv1.VmState
}

// DeleteVMRequest represents a request to delete a virtual machine
type DeleteVMRequest struct {
	// VMID is the unique identifier of the VM to delete
	VMID string

	// Force indicates whether to force deletion even if VM is running
	Force bool
}

// DeleteVMResponse represents the response from deleting a virtual machine
type DeleteVMResponse struct {
	// Success indicates if the deletion operation was successful
	Success bool
}

// VMInfo represents detailed information about a virtual machine
type VMInfo struct {
	// VMID is the unique identifier of the VM
	VMID string

	// State is the current state of the VM
	State vmprovisionerv1.VmState

	// Config is the VM configuration
	Config *vmprovisionerv1.VmConfig

	// Metrics contains runtime metrics for the VM
	Metrics *vmprovisionerv1.VmMetrics

	// NetworkInfo contains network configuration and status
	NetworkInfo *vmprovisionerv1.VmNetworkInfo
}

// ListVMsRequest represents a request to list virtual machines
type ListVMsRequest struct {
	// PageSize is the maximum number of VMs to return (default: 50, max: 100)
	PageSize int32

	// PageToken is the token for pagination (empty for first page)
	PageToken string
}

// ListVMsResponse represents the response from listing virtual machines
type ListVMsResponse struct {
	// VMs is the list of virtual machines for the authenticated customer
	VMs []*vmprovisionerv1.VmInfo

	// NextPageToken is the token for the next page (empty if no more pages)
	NextPageToken string

	// TotalCount is the total number of VMs for the customer
	TotalCount int32
}

// PauseVMResponse represents the response from pausing a virtual machine
type PauseVMResponse struct {
	// Success indicates if the pause operation was successful
	Success bool

	// State is the current state of the VM after pause attempt
	State vmprovisionerv1.VmState
}

// ResumeVMResponse represents the response from resuming a virtual machine
type ResumeVMResponse struct {
	// Success indicates if the resume operation was successful
	Success bool

	// State is the current state of the VM after resume attempt
	State vmprovisionerv1.VmState
}

// RebootVMRequest represents a request to reboot a virtual machine
type RebootVMRequest struct {
	// VMID is the unique identifier of the VM to reboot
	VMID string

	// Force indicates whether to force reboot (hard reset vs graceful restart)
	Force bool
}

// RebootVMResponse represents the response from rebooting a virtual machine
type RebootVMResponse struct {
	// Success indicates if the reboot operation was successful
	Success bool

	// State is the current state of the VM after reboot attempt
	State vmprovisionerv1.VmState
}
