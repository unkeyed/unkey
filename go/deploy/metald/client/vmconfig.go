package client

import (
	"fmt"

	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
)

// AIDEV-NOTE: VM configuration builder for customizable VM creation
// This provides a fluent interface for building VM configurations with sensible defaults

// VMConfigBuilder provides a fluent interface for building VM configurations
type VMConfigBuilder struct {
	config *vmprovisionerv1.VmConfig
}

// GetConfig returns the current configuration (for accessing intermediate state)
func (b *VMConfigBuilder) GetConfig() *vmprovisionerv1.VmConfig {
	return b.config
}

// NewVMConfigBuilder creates a new VM configuration builder with defaults
func NewVMConfigBuilder() *VMConfigBuilder {
	return &VMConfigBuilder{
		config: &vmprovisionerv1.VmConfig{
			Cpu: &vmprovisionerv1.CpuConfig{
				VcpuCount:    2,
				MaxVcpuCount: 4,
			},
			Memory: &vmprovisionerv1.MemoryConfig{
				SizeBytes:      1 * 1024 * 1024 * 1024, // 1GB
				HotplugEnabled: true,
				MaxSizeBytes:   4 * 1024 * 1024 * 1024, // 4GB max
			},
			Boot: &vmprovisionerv1.BootConfig{
				KernelPath: "/opt/vm-assets/vmlinux",
				KernelArgs: "console=ttyS0 reboot=k panic=1 pci=off",
			},
			Storage: []*vmprovisionerv1.StorageDevice{},
			Network: []*vmprovisionerv1.NetworkInterface{},
			Console: &vmprovisionerv1.ConsoleConfig{
				Enabled:     true,
				Output:      "/tmp/vm-console.log",
				ConsoleType: "serial",
			},
			Metadata: make(map[string]string),
		},
	}
}

// WithCPU configures CPU settings
func (b *VMConfigBuilder) WithCPU(vcpuCount, maxVcpuCount uint32) *VMConfigBuilder {
	b.config.Cpu = &vmprovisionerv1.CpuConfig{
		VcpuCount:    int32(vcpuCount),
		MaxVcpuCount: int32(maxVcpuCount),
	}
	return b
}

// WithMemory configures memory settings
func (b *VMConfigBuilder) WithMemory(sizeBytes, maxSizeBytes uint64, hotplugEnabled bool) *VMConfigBuilder {
	b.config.Memory = &vmprovisionerv1.MemoryConfig{
		SizeBytes:      int64(sizeBytes),
		MaxSizeBytes:   int64(maxSizeBytes),
		HotplugEnabled: hotplugEnabled,
	}
	return b
}

// WithMemoryMB configures memory settings using megabytes for convenience
func (b *VMConfigBuilder) WithMemoryMB(sizeMB, maxSizeMB uint64, hotplugEnabled bool) *VMConfigBuilder {
	return b.WithMemory(sizeMB*1024*1024, maxSizeMB*1024*1024, hotplugEnabled)
}

// WithMemoryGB configures memory settings using gigabytes for convenience
func (b *VMConfigBuilder) WithMemoryGB(sizeGB, maxSizeGB uint64, hotplugEnabled bool) *VMConfigBuilder {
	return b.WithMemory(sizeGB*1024*1024*1024, maxSizeGB*1024*1024*1024, hotplugEnabled)
}

// WithBoot configures boot settings
func (b *VMConfigBuilder) WithBoot(kernelPath, initrdPath, kernelArgs string) *VMConfigBuilder {
	b.config.Boot = &vmprovisionerv1.BootConfig{
		KernelPath: kernelPath,
		InitrdPath: initrdPath,
		KernelArgs: kernelArgs,
	}
	return b
}

// WithDefaultBoot configures standard boot settings with kernel args
func (b *VMConfigBuilder) WithDefaultBoot(kernelArgs string) *VMConfigBuilder {
	if kernelArgs == "" {
		kernelArgs = "console=ttyS0 reboot=k panic=1 pci=off"
	}
	return b.WithBoot("/opt/vm-assets/vmlinux", "", kernelArgs)
}

// AddStorage adds a storage device to the VM
func (b *VMConfigBuilder) AddStorage(id, path string, readOnly, isRoot bool, interfaceType string) *VMConfigBuilder {
	if interfaceType == "" {
		interfaceType = "virtio-blk"
	}

	storage := &vmprovisionerv1.StorageDevice{
		Id:            id,
		Path:          path,
		ReadOnly:      readOnly,
		IsRootDevice:  isRoot,
		InterfaceType: interfaceType,
		Options:       make(map[string]string),
	}

	b.config.Storage = append(b.config.Storage, storage)
	return b
}

// AddRootStorage adds the root filesystem storage device
func (b *VMConfigBuilder) AddRootStorage(path string) *VMConfigBuilder {
	return b.AddStorage("rootfs", path, false, true, "virtio-blk")
}

// AddDataStorage adds a data storage device
func (b *VMConfigBuilder) AddDataStorage(id, path string, readOnly bool) *VMConfigBuilder {
	return b.AddStorage(id, path, readOnly, false, "virtio-blk")
}

// AddStorageWithOptions adds a storage device with custom options
func (b *VMConfigBuilder) AddStorageWithOptions(id, path string, readOnly, isRoot bool, interfaceType string, options map[string]string) *VMConfigBuilder {
	if interfaceType == "" {
		interfaceType = "virtio-blk"
	}

	storage := &vmprovisionerv1.StorageDevice{
		Id:            id,
		Path:          path,
		ReadOnly:      readOnly,
		IsRootDevice:  isRoot,
		InterfaceType: interfaceType,
		Options:       options,
	}

	b.config.Storage = append(b.config.Storage, storage)
	return b
}

// AddNetwork adds a network interface to the VM
func (b *VMConfigBuilder) AddNetwork(id, interfaceType string, mode vmprovisionerv1.NetworkMode) *VMConfigBuilder {
	if interfaceType == "" {
		interfaceType = "virtio-net"
	}
	if mode == vmprovisionerv1.NetworkMode_NETWORK_MODE_UNSPECIFIED {
		mode = vmprovisionerv1.NetworkMode_NETWORK_MODE_DUAL_STACK
	}

	network := &vmprovisionerv1.NetworkInterface{
		Id:            id,
		InterfaceType: interfaceType,
		Mode:          mode,
		Ipv4Config: &vmprovisionerv1.IPv4Config{
			Dhcp: true,
		},
		Ipv6Config: &vmprovisionerv1.IPv6Config{
			Slaac:             true,
			PrivacyExtensions: true,
		},
	}

	b.config.Network = append(b.config.Network, network)
	return b
}

// AddDefaultNetwork adds a standard dual-stack network interface
func (b *VMConfigBuilder) AddDefaultNetwork() *VMConfigBuilder {
	return b.AddNetwork("eth0", "virtio-net", vmprovisionerv1.NetworkMode_NETWORK_MODE_DUAL_STACK)
}

// AddIPv4OnlyNetwork adds an IPv4-only network interface
func (b *VMConfigBuilder) AddIPv4OnlyNetwork(id string) *VMConfigBuilder {
	return b.AddNetwork(id, "virtio-net", vmprovisionerv1.NetworkMode_NETWORK_MODE_IPV4_ONLY)
}

// AddIPv6OnlyNetwork adds an IPv6-only network interface
func (b *VMConfigBuilder) AddIPv6OnlyNetwork(id string) *VMConfigBuilder {
	return b.AddNetwork(id, "virtio-net", vmprovisionerv1.NetworkMode_NETWORK_MODE_IPV6_ONLY)
}

// AddNetworkWithCustomConfig adds a network interface with custom IPv4/IPv6 configuration
func (b *VMConfigBuilder) AddNetworkWithCustomConfig(id, interfaceType string, mode vmprovisionerv1.NetworkMode,
	ipv4Config *vmprovisionerv1.IPv4Config, ipv6Config *vmprovisionerv1.IPv6Config,
) *VMConfigBuilder {
	if interfaceType == "" {
		interfaceType = "virtio-net"
	}

	network := &vmprovisionerv1.NetworkInterface{
		Id:            id,
		InterfaceType: interfaceType,
		Mode:          mode,
		Ipv4Config:    ipv4Config,
		Ipv6Config:    ipv6Config,
	}

	b.config.Network = append(b.config.Network, network)
	return b
}

// WithConsole configures console settings
func (b *VMConfigBuilder) WithConsole(enabled bool, output, consoleType string) *VMConfigBuilder {
	b.config.Console = &vmprovisionerv1.ConsoleConfig{
		Enabled:     enabled,
		Output:      output,
		ConsoleType: consoleType,
	}
	return b
}

// WithDefaultConsole configures standard console settings
func (b *VMConfigBuilder) WithDefaultConsole(output string) *VMConfigBuilder {
	if output == "" {
		output = "/tmp/vm-console.log"
	}
	return b.WithConsole(true, output, "serial")
}

// DisableConsole disables console output
func (b *VMConfigBuilder) DisableConsole() *VMConfigBuilder {
	return b.WithConsole(false, "", "")
}

// AddMetadata adds metadata key-value pairs
func (b *VMConfigBuilder) AddMetadata(key, value string) *VMConfigBuilder {
	if b.config.Metadata == nil {
		b.config.Metadata = make(map[string]string)
	}
	b.config.Metadata[key] = value
	return b
}

// WithMetadata sets all metadata at once
func (b *VMConfigBuilder) WithMetadata(metadata map[string]string) *VMConfigBuilder {
	b.config.Metadata = metadata
	return b
}

// Build returns the configured VM configuration
func (b *VMConfigBuilder) Build() *vmprovisionerv1.VmConfig {
	return b.config
}

// ValidateVMConfig validates a VM configuration and returns any errors
func ValidateVMConfig(config *vmprovisionerv1.VmConfig) error {
	// Validate CPU configuration
	if config.Cpu == nil {
		return fmt.Errorf("CPU configuration is required")
	}
	if config.Cpu.VcpuCount == 0 {
		return fmt.Errorf("vCPU count must be greater than 0")
	}
	if config.Cpu.MaxVcpuCount < config.Cpu.VcpuCount {
		return fmt.Errorf("max vCPU count (%d) must be >= current vCPU count (%d)",
			config.Cpu.MaxVcpuCount, config.Cpu.VcpuCount)
	}

	// Validate memory configuration
	if config.Memory == nil {
		return fmt.Errorf("memory configuration is required")
	}
	if config.Memory.SizeBytes == 0 {
		return fmt.Errorf("memory size must be greater than 0")
	}
	if config.Memory.MaxSizeBytes < config.Memory.SizeBytes {
		return fmt.Errorf("max memory size (%d) must be >= current memory size (%d)",
			config.Memory.MaxSizeBytes, config.Memory.SizeBytes)
	}

	// Validate boot configuration
	if config.Boot == nil {
		return fmt.Errorf("boot configuration is required")
	}
	if config.Boot.KernelPath == "" {
		return fmt.Errorf("kernel path is required")
	}

	// Validate storage - must have at least one root device
	hasRoot := false
	for _, storage := range config.Storage {
		if storage.IsRootDevice {
			if hasRoot {
				return fmt.Errorf("multiple root devices found - only one root device is allowed")
			}
			hasRoot = true
		}
		if storage.Id == "" {
			return fmt.Errorf("storage device ID cannot be empty")
		}
		if storage.Path == "" {
			return fmt.Errorf("storage device path cannot be empty for device %s", storage.Id)
		}
	}
	if !hasRoot && len(config.Storage) > 0 {
		return fmt.Errorf("at least one storage device must be marked as root device")
	}

	// Validate network interfaces
	for _, network := range config.Network {
		if network.Id == "" {
			return fmt.Errorf("network interface ID cannot be empty")
		}
		if network.Mode == vmprovisionerv1.NetworkMode_NETWORK_MODE_UNSPECIFIED {
			return fmt.Errorf("network mode must be specified for interface %s", network.Id)
		}
	}

	return nil
}

// Validate validates the configuration and returns any errors
func (b *VMConfigBuilder) Validate() error {
	return ValidateVMConfig(b.config)
}

// VMTemplate represents common VM configuration templates
type VMTemplate string

const (
	// TemplateMinimal creates a minimal VM with basic resources
	TemplateMinimal VMTemplate = "minimal"
	// TemplateStandard creates a standard VM with balanced resources
	TemplateStandard VMTemplate = "standard"
	// TemplateHighCPU creates a VM optimized for CPU-intensive workloads
	TemplateHighCPU VMTemplate = "high-cpu"
	// TemplateHighMemory creates a VM optimized for memory-intensive workloads
	TemplateHighMemory VMTemplate = "high-memory"
	// TemplateDevelopment creates a VM suitable for development work
	TemplateDevelopment VMTemplate = "development"
)

// ForDockerImage configures the VM for running a specific Docker image
func (b *VMConfigBuilder) ForDockerImage(imageName string) *VMConfigBuilder {
	// Add Docker-specific metadata and storage configuration
	b.AddMetadata("docker_image", imageName).
		AddMetadata("runtime", "docker")

	// AIDEV-NOTE: Use standardized rootfs path instead of Docker image-specific naming
	// This aligns with assetmanagerd's PrepareAssets method which uses "rootfs.ext4"
	// AIDEV-QUESTION: These hardcoded paths confuse users who think they need to set them.
	// The paths are actually placeholders - the system uses metadata (docker_image) to find/build assets.
	// Consider making the API clearer by:
	// 1. Making paths optional and auto-generating them
	// 2. Using a different field name like "asset_reference" instead of "path"
	// 3. Adding clear documentation that these are placeholder values
	// 4. Providing a higher-level API that doesn't expose paths at all
	rootfsPath := "/opt/vm-assets/rootfs.ext4"

	// Replace any existing root storage with Docker-specific one
	newStorage := []*vmprovisionerv1.StorageDevice{}
	for _, storage := range b.config.Storage {
		if !storage.IsRootDevice {
			newStorage = append(newStorage, storage)
		}
	}
	b.config.Storage = newStorage

	// Add Docker rootfs with metadata for automatic build system
	b.AddStorageWithOptions("rootfs", rootfsPath, false, true, "virtio-blk",
		map[string]string{
			"docker_image": imageName,
			"auto_build":   "true",
		})

	return b
}

// sanitizeImageName converts a Docker image name to a safe filename
func sanitizeImageName(imageName string) string {
	// Replace special characters with underscores
	safe := imageName
	replacements := map[string]string{
		"/": "_",
		":": "_",
		"@": "_",
		"+": "_",
		" ": "_",
	}

	for old, new := range replacements {
		safe = fmt.Sprintf("%s", safe)
		// Simple replacement without complex string manipulation
		result := ""
		for _, char := range safe {
			if string(char) == old {
				result += new
			} else {
				result += string(char)
			}
		}
		safe = result
	}
	return safe
}

// ForceBuild configures the VM to force rebuild assets even if cached versions exist
func (b *VMConfigBuilder) ForceBuild(force bool) *VMConfigBuilder {
	// Add force build metadata that will be picked up by the asset management system
	if force {
		b.AddMetadata("force_rebuild", "true")
	} else {
		// Remove force rebuild metadata if it exists
		if b.config.Metadata != nil {
			delete(b.config.Metadata, "force_rebuild")
		}
	}
	return b
}
