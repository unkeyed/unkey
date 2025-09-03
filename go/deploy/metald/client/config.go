package client

import (
	"encoding/json"
	"fmt"
	"os"

	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
)

// VMConfigFile represents a VM configuration that can be loaded from/saved to a file
type VMConfigFile struct {
	// Name is a human-readable name for this configuration
	Name string `json:"name"`

	// Description describes the purpose of this configuration
	Description string `json:"description"`

	// Template is the base template to use (optional)
	Template string `json:"template,omitempty"`

	// CPU configuration
	CPU CPUConfig `json:"cpu"`

	// Memory configuration
	Memory MemoryConfig `json:"memory"`

	// Boot configuration
	Boot BootConfig `json:"boot"`

	// Storage devices
	Storage []StorageConfig `json:"storage"`

	// Network interfaces
	Network []NetworkConfig `json:"network"`

	// Console configuration
	Console ConsoleConfig `json:"console"`

	// Metadata key-value pairs
	Metadata map[string]string `json:"metadata"`
}

// CPUConfig represents CPU configuration in a config file
type CPUConfig struct {
	VCPUCount    uint32 `json:"vcpu_count"`
	MaxVCPUCount uint32 `json:"max_vcpu_count"`
}

// MemoryConfig represents memory configuration in a config file
type MemoryConfig struct {
	SizeMB         uint64 `json:"size_mb"`
	MaxSizeMB      uint64 `json:"max_size_mb"`
	HotplugEnabled bool   `json:"hotplug_enabled"`
}

// BootConfig represents boot configuration in a config file
type BootConfig struct {
	KernelPath string `json:"kernel_path"`
	InitrdPath string `json:"initrd_path,omitempty"`
	KernelArgs string `json:"kernel_args"`
}

// StorageConfig represents storage device configuration in a config file
type StorageConfig struct {
	ID            string            `json:"id"`
	Path          string            `json:"path"`
	ReadOnly      bool              `json:"read_only"`
	IsRootDevice  bool              `json:"is_root_device"`
	InterfaceType string            `json:"interface_type"`
	Options       map[string]string `json:"options,omitempty"`
}

// NetworkConfig represents network interface configuration in a config file
type NetworkConfig struct {
	ID            string      `json:"id"`
	InterfaceType string      `json:"interface_type"`
	Mode          string      `json:"mode"` // "dual_stack", "ipv4_only", "ipv6_only"
	IPv4          *IPv4Config `json:"ipv4,omitempty"`
	IPv6          *IPv6Config `json:"ipv6,omitempty"`
}

// IPv4Config represents IPv4 configuration in a config file
type IPv4Config struct {
	DHCP       bool     `json:"dhcp"`
	StaticIP   string   `json:"static_ip,omitempty"`
	Gateway    string   `json:"gateway,omitempty"`
	DNSServers []string `json:"dns_servers,omitempty"`
}

// IPv6Config represents IPv6 configuration in a config file
type IPv6Config struct {
	SLAAC             bool     `json:"slaac"`
	PrivacyExtensions bool     `json:"privacy_extensions"`
	StaticIP          string   `json:"static_ip,omitempty"`
	Gateway           string   `json:"gateway,omitempty"`
	DNSServers        []string `json:"dns_servers,omitempty"`
}

// ConsoleConfig represents console configuration in a config file
type ConsoleConfig struct {
	Enabled     bool   `json:"enabled"`
	Output      string `json:"output"`
	ConsoleType string `json:"console_type"`
}

// LoadVMConfigFromFile loads a VM configuration from a JSON file
func LoadVMConfigFromFile(filename string) (*VMConfigFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	var config VMConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}

	return &config, nil
}

// ToVMConfig converts a VMConfigFile to a protobuf VmConfig
func (c *VMConfigFile) ToVMConfig() (*vmprovisionerv1.VmConfig, error) {
	var builder *VMConfigBuilder

	builder = NewVMConfigBuilder()

	// Override with specific configuration
	builder.WithCPU(c.CPU.VCPUCount, c.CPU.MaxVCPUCount)
	builder.WithMemoryMB(c.Memory.SizeMB, c.Memory.MaxSizeMB, c.Memory.HotplugEnabled)
	builder.WithBoot(c.Boot.KernelPath, c.Boot.InitrdPath, c.Boot.KernelArgs)

	// Clear storage and network from template
	builder.config.Storage = []*vmprovisionerv1.StorageDevice{}
	builder.config.Network = []*vmprovisionerv1.NetworkInterface{}

	// Add storage devices
	for _, storage := range c.Storage {
		interfaceType := storage.InterfaceType
		if interfaceType == "" {
			interfaceType = "virtio-blk"
		}
		builder.AddStorageWithOptions(storage.ID, storage.Path, storage.ReadOnly,
			storage.IsRootDevice, interfaceType, storage.Options)
	}

	// Add network interfaces
	for _, network := range c.Network {
		mode := parseNetworkMode(network.Mode)
		interfaceType := network.InterfaceType
		if interfaceType == "" {
			interfaceType = "virtio-net"
		}

		var ipv4Config *vmprovisionerv1.IPv4Config
		var ipv6Config *vmprovisionerv1.IPv6Config

		if network.IPv4 != nil {
			ipv4Config = &vmprovisionerv1.IPv4Config{
				Dhcp:       network.IPv4.DHCP,
				Address:    network.IPv4.StaticIP,
				Gateway:    network.IPv4.Gateway,
				DnsServers: network.IPv4.DNSServers,
			}
		}

		if network.IPv6 != nil {
			ipv6Config = &vmprovisionerv1.IPv6Config{
				Slaac:             network.IPv6.SLAAC,
				PrivacyExtensions: network.IPv6.PrivacyExtensions,
				Address:           network.IPv6.StaticIP,
				Gateway:           network.IPv6.Gateway,
				DnsServers:        network.IPv6.DNSServers,
			}
		}

		builder.AddNetworkWithCustomConfig(network.ID, interfaceType, mode, ipv4Config, ipv6Config)
	}

	// Configure console
	builder.WithConsole(c.Console.Enabled, c.Console.Output, c.Console.ConsoleType)

	// Add metadata
	if c.Metadata != nil {
		builder.WithMetadata(c.Metadata)
	}

	// Add config file metadata
	builder.AddMetadata("config_name", c.Name)
	builder.AddMetadata("config_description", c.Description)

	return builder.Build(), nil
}

// FromVMConfig creates a VMConfigFile from a protobuf VmConfig
func FromVMConfig(config *vmprovisionerv1.VmConfig, name, description string) *VMConfigFile {
	configFile := &VMConfigFile{
		Name:        name,
		Description: description,
		CPU: CPUConfig{
			VCPUCount:    uint32(config.Cpu.VcpuCount),
			MaxVCPUCount: uint32(config.Cpu.MaxVcpuCount),
		},
		Memory: MemoryConfig{
			SizeMB:         uint64(config.Memory.SizeBytes / (1024 * 1024)),
			MaxSizeMB:      uint64(config.Memory.MaxSizeBytes / (1024 * 1024)),
			HotplugEnabled: config.Memory.HotplugEnabled,
		},
		Boot: BootConfig{
			KernelPath: config.Boot.KernelPath,
			InitrdPath: config.Boot.InitrdPath,
			KernelArgs: config.Boot.KernelArgs,
		},
		Storage: []StorageConfig{},
		Network: []NetworkConfig{},
		Console: ConsoleConfig{
			Enabled:     config.Console.Enabled,
			Output:      config.Console.Output,
			ConsoleType: config.Console.ConsoleType,
		},
		Metadata: config.Metadata,
	}

	// Convert storage devices
	for _, storage := range config.Storage {
		configFile.Storage = append(configFile.Storage, StorageConfig{
			ID:            storage.Id,
			Path:          storage.Path,
			ReadOnly:      storage.ReadOnly,
			IsRootDevice:  storage.IsRootDevice,
			InterfaceType: storage.InterfaceType,
			Options:       storage.Options,
		})
	}

	// Convert network interfaces
	for _, network := range config.Network {
		netConfig := NetworkConfig{
			ID:            network.Id,
			InterfaceType: network.InterfaceType,
			Mode:          formatNetworkMode(network.Mode),
		}

		if network.Ipv4Config != nil {
			netConfig.IPv4 = &IPv4Config{
				DHCP:       network.Ipv4Config.Dhcp,
				StaticIP:   network.Ipv4Config.Address,
				Gateway:    network.Ipv4Config.Gateway,
				DNSServers: network.Ipv4Config.DnsServers,
			}
		}

		if network.Ipv6Config != nil {
			netConfig.IPv6 = &IPv6Config{
				SLAAC:             network.Ipv6Config.Slaac,
				PrivacyExtensions: network.Ipv6Config.PrivacyExtensions,
				StaticIP:          network.Ipv6Config.Address,
				Gateway:           network.Ipv6Config.Gateway,
				DNSServers:        network.Ipv6Config.DnsServers,
			}
		}

		configFile.Network = append(configFile.Network, netConfig)
	}

	return configFile
}

// parseNetworkMode converts string to protobuf NetworkMode
func parseNetworkMode(mode string) vmprovisionerv1.NetworkMode {
	switch mode {
	case "ipv4_only":
		return vmprovisionerv1.NetworkMode_NETWORK_MODE_IPV4_ONLY
	case "ipv6_only":
		return vmprovisionerv1.NetworkMode_NETWORK_MODE_IPV6_ONLY
	case "dual_stack":
		return vmprovisionerv1.NetworkMode_NETWORK_MODE_DUAL_STACK
	default:
		return vmprovisionerv1.NetworkMode_NETWORK_MODE_DUAL_STACK
	}
}

// formatNetworkMode converts protobuf NetworkMode to string
func formatNetworkMode(mode vmprovisionerv1.NetworkMode) string {
	switch mode {
	case vmprovisionerv1.NetworkMode_NETWORK_MODE_IPV4_ONLY:
		return "ipv4_only"
	case vmprovisionerv1.NetworkMode_NETWORK_MODE_IPV6_ONLY:
		return "ipv6_only"
	case vmprovisionerv1.NetworkMode_NETWORK_MODE_DUAL_STACK:
		return "dual_stack"
	default:
		return "dual_stack"
	}
}
