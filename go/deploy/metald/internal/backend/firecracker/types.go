package firecracker

import (
	"fmt"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
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
func (c *Client) firecrackerStateToGeneric(state string) metaldv1.VmState {
	switch state {
	case "NotStarted":
		return metaldv1.VmState_VM_STATE_CREATED
	case "Running":
		return metaldv1.VmState_VM_STATE_RUNNING
	case "Paused":
		return metaldv1.VmState_VM_STATE_PAUSED
	default:
		return metaldv1.VmState_VM_STATE_UNSPECIFIED
	}
}

// AIDEV-NOTE: Convert generic VM config to Firecracker-specific configurations
func (c *Client) genericToFirecrackerConfig(config *metaldv1.VmConfig) (firecrackerMachineConfig, firecrackerBootSource, []firecrackerDrive, []firecrackerNetworkInterface) {
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
		// AIDEV-NOTE: When using jailer, paths must be relative to the chroot
		// Jailer expects files to be available at the absolute paths inside the chroot
		// So we keep absolute paths - jailer will look for them relative to chroot root
		bootSource.KernelImagePath = config.Boot.KernelPath
		bootSource.BootArgs = config.Boot.KernelArgs
		bootSource.InitrdPath = config.Boot.InitrdPath
	}

	// Storage configuration
	for i, disk := range config.Storage {
		driveID := disk.Id
		if driveID == "" {
			if disk.IsRootDevice || i == 0 {
				driveID = "rootfs"
			} else {
				driveID = fmt.Sprintf("drive_%d", i)
			}
		}

		drive := firecrackerDrive{
			DriveID:      driveID,
			PathOnHost:   disk.Path, // AIDEV-NOTE: Same as kernel - jailer expects absolute paths
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
