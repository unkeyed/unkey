package docker

import (
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

// dockerVM represents a "VM" managed as a Docker container
type dockerVM struct {
	ID           string
	ContainerID  string
	Config       *metaldv1.VmConfig
	State        metaldv1.VmState
	PortMappings []PortMapping
	CreatedAt    time.Time
}

// PortMapping represents a port mapping between container and host
type PortMapping struct {
	ContainerPort int
	HostPort      int
	Protocol      string
}

// ContainerSpec represents the Docker container specification
type ContainerSpec struct {
	Image        string
	Cmd          []string
	Env          []string
	ExposedPorts []string
	PortMappings []PortMapping
	Labels       map[string]string
	Memory       int64
	CPUs         float64
	WorkingDir   string
}

// DockerBackendConfig represents configuration for the Docker backend
type DockerBackendConfig struct {
	// DockerHost is the Docker daemon socket (defaults to unix:///var/run/docker.sock)
	DockerHost string `json:"docker_host,omitempty"`

	// NetworkName is the Docker network to use for containers (defaults to bridge)
	NetworkName string `json:"network_name,omitempty"`

	// ContainerPrefix is the prefix for container names (defaults to unkey-vm-)
	ContainerPrefix string `json:"container_prefix,omitempty"`

	// PortRange defines the range of host ports to allocate
	PortRange struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"port_range,omitempty"`

	// AutoRemove determines if containers should be automatically removed on exit
	AutoRemove bool `json:"auto_remove,omitempty"`

	// Privileged determines if containers run in privileged mode
	Privileged bool `json:"privileged,omitempty"`
}

// DefaultDockerBackendConfig returns default configuration for Docker backend
func DefaultDockerBackendConfig() *DockerBackendConfig {
	return &DockerBackendConfig{
		DockerHost:      "", // Use default Docker socket
		NetworkName:     "bridge",
		ContainerPrefix: "unkey-vm-",
		PortRange: struct {
			Min int `json:"min"`
			Max int `json:"max"`
		}{
			Min: 30000,
			Max: 40000,
		},
		AutoRemove: true,
		Privileged: false,
	}
}

// DockerMetrics represents metrics collected from Docker stats API
type DockerMetrics struct {
	Timestamp        time.Time
	CPUUsagePercent  float64
	MemoryUsageBytes int64
	MemoryLimitBytes int64
	NetworkRxBytes   int64
	NetworkTxBytes   int64
	BlockReadBytes   int64
	BlockWriteBytes  int64
	PIDs             int64
}

// ToVMMetrics converts DockerMetrics to types.VMMetrics
func (dm *DockerMetrics) ToVMMetrics() *types.VMMetrics {
	return &types.VMMetrics{
		Timestamp:        dm.Timestamp,
		CpuTimeNanos:     int64(dm.CPUUsagePercent * 1e9), // Convert percentage to nanoseconds approximation
		MemoryUsageBytes: dm.MemoryUsageBytes,
		DiskReadBytes:    dm.BlockReadBytes,
		DiskWriteBytes:   dm.BlockWriteBytes,
		NetworkRxBytes:   dm.NetworkRxBytes,
		NetworkTxBytes:   dm.NetworkTxBytes,
	}
}

// ContainerCreateOptions represents options for creating a Docker container
type ContainerCreateOptions struct {
	Name         string
	Image        string
	Cmd          []string
	Env          []string
	Labels       map[string]string
	ExposedPorts map[string]struct{}
	PortBindings map[string][]nat.PortBinding
	Memory       int64
	CPUs         float64
	WorkingDir   string
	AutoRemove   bool
	Privileged   bool
	NetworkMode  string
}

// NetworkCreateOptions represents options for creating a Docker network
type NetworkCreateOptions struct {
	Name     string
	Driver   string
	Internal bool
	Labels   map[string]string
}

// AIDEV-NOTE: Docker backend types provide a clean abstraction layer
// between the VM concepts used by metald and Docker container operations.
// This allows metald to treat Docker containers as VMs while maintaining
// the same interface contract as the Firecracker backend.
