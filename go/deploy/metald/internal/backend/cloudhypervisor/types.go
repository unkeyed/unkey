package cloudhypervisor

import (
	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
)

// Cloud Hypervisor API types

type cloudHypervisorVMInfo struct {
	State  string                `json:"state"`
	Config cloudHypervisorConfig `json:"config"`
}

type cloudHypervisorConfig struct {
	Cpus    *cpusConfig    `json:"cpus,omitempty"`
	Memory  *memoryConfig  `json:"memory,omitempty"`
	Payload *payloadConfig `json:"payload,omitempty"`
	Disks   []diskConfig   `json:"disks,omitempty"`
	Net     []netConfig    `json:"net,omitempty"`
	Rng     *rngConfig     `json:"rng,omitempty"`
	Balloon *balloonConfig `json:"balloon,omitempty"`
	Console *consoleConfig `json:"console,omitempty"`
	Serial  *consoleConfig `json:"serial,omitempty"`
}

type cpusConfig struct {
	BootVcpus int32        `json:"boot_vcpus"`
	MaxVcpus  int32        `json:"max_vcpus"`
	Topology  *cpuTopology `json:"topology,omitempty"`
}

type cpuTopology struct {
	ThreadsPerCore int32 `json:"threads_per_core"`
	CoresPerDie    int32 `json:"cores_per_die"`
	DiesPerPackage int32 `json:"dies_per_package"`
	Packages       int32 `json:"packages"`
}

type memoryConfig struct {
	Size           int64 `json:"size"`
	HotplugEnabled bool  `json:"hotplug_enabled,omitempty"`
	HotplugSize    int64 `json:"hotplug_size,omitempty"`
	Shared         bool  `json:"shared,omitempty"`
	Hugepages      bool  `json:"hugepages,omitempty"`
}

type payloadConfig struct {
	Kernel    string `json:"kernel"`
	Initramfs string `json:"initramfs,omitempty"`
	Cmdline   string `json:"cmdline,omitempty"`
}

type diskConfig struct {
	Path     string `json:"path"`
	Readonly bool   `json:"readonly,omitempty"`
	Direct   bool   `json:"direct,omitempty"`
}

type netConfig struct {
	Tap  string `json:"tap,omitempty"`
	Mac  string `json:"mac,omitempty"`
	IP   string `json:"ip,omitempty"`
	Mask string `json:"mask,omitempty"`
}

type rngConfig struct {
	Src string `json:"src"`
}

type balloonConfig struct {
	Size         int64 `json:"size"`
	DeflateOnOOM bool  `json:"deflate_on_oom,omitempty"`
}

type consoleConfig struct {
	Mode string `json:"mode"`
	File string `json:"file,omitempty"`
}

// genericToCloudHypervisorConfig converts generic VM config to Cloud Hypervisor API format
func (c *Client) genericToCloudHypervisorConfig(config *metaldv1.VmConfig) cloudHypervisorConfig {
	chConfig := cloudHypervisorConfig{} //exhaustruct:ignore

	// CPU configuration
	if config.GetCpu() != nil && config.GetCpu().GetVcpuCount() > 0 {
		//exhaustruct:ignore
		chConfig.Cpus = &cpusConfig{
			BootVcpus: config.GetCpu().GetVcpuCount(),
			MaxVcpus:  config.GetCpu().GetMaxVcpuCount(),
		}
		if config.GetCpu().GetMaxVcpuCount() == 0 {
			chConfig.Cpus.MaxVcpus = config.GetCpu().GetVcpuCount()
		}
	}

	// Memory configuration
	if config.GetMemory() != nil && config.GetMemory().GetSizeBytes() > 0 {
		//exhaustruct:ignore
		chConfig.Memory = &memoryConfig{
			Size: config.GetMemory().GetSizeBytes(),
		}
	}

	// Payload configuration
	if config.GetBoot() != nil && config.GetBoot().GetKernelPath() != "" {
		chConfig.Payload = &payloadConfig{
			Kernel:    config.GetBoot().GetKernelPath(),
			Initramfs: config.GetBoot().GetInitrdPath(),
			Cmdline:   config.GetBoot().GetKernelArgs(),
		}
	}

	// Disk configuration
	for _, disk := range config.GetStorage() {
		//exhaustruct:ignore
		chConfig.Disks = append(chConfig.Disks, diskConfig{
			Path:     disk.GetPath(),
			Readonly: disk.GetReadOnly(),
		})
	}

	// Network configuration
	for _, net := range config.GetNetwork() {
		//exhaustruct:ignore
		chConfig.Net = append(chConfig.Net, netConfig{
			Tap: net.GetTapDevice(),
			Mac: net.GetMacAddress(),
		})
	}

	// Default RNG configuration
	chConfig.Rng = &rngConfig{
		Src: "/dev/urandom",
	}

	// Console configuration
	if config.GetConsole() != nil && config.GetConsole().GetEnabled() {
		//exhaustruct:ignore
		chConfig.Console = &consoleConfig{
			Mode: "File",
			File: config.GetConsole().GetOutput(),
		}
	} else {
		//exhaustruct:ignore
		chConfig.Console = &consoleConfig{
			Mode: "Off",
		}
	}

	return chConfig
}

// cloudHypervisorStateToGeneric converts Cloud Hypervisor state to generic VM state
func (c *Client) cloudHypervisorStateToGeneric(state string) metaldv1.VmState {
	switch state {
	case "Created":
		return metaldv1.VmState_VM_STATE_CREATED
	case "Running":
		return metaldv1.VmState_VM_STATE_RUNNING
	case "Shutdown":
		return metaldv1.VmState_VM_STATE_SHUTDOWN
	case "Paused":
		return metaldv1.VmState_VM_STATE_PAUSED
	default:
		return metaldv1.VmState_VM_STATE_UNSPECIFIED
	}
}
