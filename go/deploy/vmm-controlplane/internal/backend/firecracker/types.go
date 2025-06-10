package firecracker

import (
	vmmv1 "vmm-controlplane/gen/vmm/v1"
)

// Firecracker API types based on OpenAPI specification

type firecrackerVMInfo struct {
	State  string              `json:"state"`
	Config firecrackerVMConfig `json:"vm_config"`
}

type firecrackerVMConfig struct {
	VcpuCount   int32  `json:"vcpu_count"`
	MemSizeMib  int32  `json:"mem_size_mib"`
	HtEnabled   bool   `json:"ht_enabled,omitempty"`
	CpuTemplate string `json:"cpu_template,omitempty"`
}

type firecrackerMachineConfig struct {
	VcpuCount   int32  `json:"vcpu_count"`
	MemSizeMib  int32  `json:"mem_size_mib"`
	HtEnabled   bool   `json:"ht_enabled,omitempty"`
	CpuTemplate string `json:"cpu_template,omitempty"`
}

type firecrackerBootSource struct {
	KernelImagePath string `json:"kernel_image_path"`
	BootArgs        string `json:"boot_args,omitempty"`
	InitrdPath      string `json:"initrd_path,omitempty"`
}

type firecrackerDrive struct {
	DriveID      string `json:"drive_id"`
	PathOnHost   string `json:"path_on_host"`
	IsRootDevice bool   `json:"is_root_device"`
	IsReadOnly   bool   `json:"is_read_only"`
}

type firecrackerNetworkInterface struct {
	IfaceID     string `json:"iface_id"`
	GuestMac    string `json:"guest_mac,omitempty"`
	HostDevName string `json:"host_dev_name"`
}

type firecrackerVsock struct {
	GuestCid int32  `json:"guest_cid"`
	UdsPath  string `json:"uds_path"`
}

type firecrackerAction struct {
	ActionType string `json:"action_type"`
}

// AIDEV-NOTE: Firecracker state mapping to generic VM states
func (c *Client) firecrackerStateToGeneric(state string) vmmv1.VmState {
	switch state {
	case "NotStarted":
		return vmmv1.VmState_VM_STATE_CREATED
	case "Running":
		return vmmv1.VmState_VM_STATE_RUNNING
	case "Paused":
		return vmmv1.VmState_VM_STATE_PAUSED
	default:
		return vmmv1.VmState_VM_STATE_UNSPECIFIED
	}
}

// AIDEV-NOTE: Convert generic VM config to Firecracker-specific configurations
func (c *Client) genericToFirecrackerConfig(config *vmmv1.VmConfig) (firecrackerMachineConfig, firecrackerBootSource, []firecrackerDrive, []firecrackerNetworkInterface) {
	machineConfig := firecrackerMachineConfig{}
	bootSource := firecrackerBootSource{}
	var drives []firecrackerDrive
	var netInterfaces []firecrackerNetworkInterface

	// CPU and Memory configuration
	if config.Cpu != nil {
		machineConfig.VcpuCount = config.Cpu.VcpuCount
	}
	if config.Memory != nil {
		machineConfig.MemSizeMib = int32(config.Memory.SizeBytes / (1024 * 1024)) // Convert bytes to MiB
	}

	// Boot configuration
	if config.Boot != nil {
		bootSource.KernelImagePath = config.Boot.KernelPath
		bootSource.BootArgs = config.Boot.KernelArgs
		bootSource.InitrdPath = config.Boot.InitrdPath
	}

	// Storage configuration
	for i, disk := range config.Storage {
		drive := firecrackerDrive{
			DriveID:      disk.Path, // Use path as ID for simplicity
			PathOnHost:   disk.Path,
			IsRootDevice: disk.IsRootDevice || i == 0, // First disk is root if not explicitly set
			IsReadOnly:   disk.ReadOnly,
		}
		drives = append(drives, drive)
	}

	// Network configuration
	for _, net := range config.Network {
		netInterface := firecrackerNetworkInterface{
			IfaceID:     net.Id,
			GuestMac:    net.MacAddress,
			HostDevName: net.TapDevice,
		}
		netInterfaces = append(netInterfaces, netInterface)
	}

	return machineConfig, bootSource, drives, netInterfaces
}
