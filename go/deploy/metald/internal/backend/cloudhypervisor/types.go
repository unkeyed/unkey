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
	chConfig := cloudHypervisorConfig{}

	// CPU configuration
	if config.Cpu != nil && config.Cpu.VcpuCount > 0 {
		chConfig.Cpus = &cpusConfig{
			BootVcpus: config.Cpu.VcpuCount,
			MaxVcpus:  config.Cpu.MaxVcpuCount,
		}
		if config.Cpu.MaxVcpuCount == 0 {
			chConfig.Cpus.MaxVcpus = config.Cpu.VcpuCount
		}
	}

	// Memory configuration
	if config.Memory != nil && config.Memory.SizeBytes > 0 {
		chConfig.Memory = &memoryConfig{
			Size: config.Memory.SizeBytes,
		}
	}

	// Payload configuration
	if config.Boot != nil && config.Boot.KernelPath != "" {
		chConfig.Payload = &payloadConfig{
			Kernel:    config.Boot.KernelPath,
			Initramfs: config.Boot.InitrdPath,
			Cmdline:   config.Boot.KernelArgs,
		}
	}

	// Disk configuration
	for _, disk := range config.Storage {
		chConfig.Disks = append(chConfig.Disks, diskConfig{
			Path:     disk.Path,
			Readonly: disk.ReadOnly,
		})
	}

	// Network configuration
	for _, net := range config.Network {
		chConfig.Net = append(chConfig.Net, netConfig{
			Tap: net.TapDevice,
			Mac: net.MacAddress,
		})
	}

	// Default RNG configuration
	chConfig.Rng = &rngConfig{
		Src: "/dev/urandom",
	}

	// Console configuration
	if config.Console != nil && config.Console.Enabled {
		chConfig.Console = &consoleConfig{
			Mode: "File",
			File: config.Console.Output,
		}
	} else {
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
