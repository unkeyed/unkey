# VmService Interface Documentation

The VMM Control Plane provides a unified API for managing virtual machines across different hypervisor backends through the `VmService` interface.

## Architecture Overview

The service follows a layered architecture:

```
┌─────────────────────────────────────────┐
│           ConnectRPC API                │  ← Unified gRPC/HTTP API
├─────────────────────────────────────────┤
│           Service Layer                 │  ← Business logic & validation
├─────────────────────────────────────────┤
│          Backend Interface              │  ← Abstract VM operations
├─────────────────────────────────────────┤
│  Cloud Hypervisor │ Firecracker │ ...  │  ← Hypervisor implementations
└─────────────────────────────────────────┘
```

## VmService API

The unified API is defined in `proto/vmm/v1/vm.proto` and provides these operations:

### Core VM Operations

- **CreateVm**: Create a new virtual machine instance
- **DeleteVm**: Remove a virtual machine instance
- **BootVm**: Start a created virtual machine
- **ShutdownVm**: Gracefully stop a running virtual machine
- **PauseVm**: Pause a running virtual machine
- **ResumeVm**: Resume a paused virtual machine
- **RebootVm**: Restart a running virtual machine

### Information & Management

- **GetVmInfo**: Retrieve VM status and configuration
- **ListVms**: List all VMs managed by this service

### VM Configuration

The unified `VmConfig` message supports:

```protobuf
message VmConfig {
  CpuConfig cpu = 1;              // CPU allocation and topology
  MemoryConfig memory = 2;        // Memory size and backing options
  BootConfig boot = 3;            // Kernel, initrd, boot arguments
  repeated StorageDevice storage = 4;  // Disk devices
  repeated NetworkInterface network = 5; // Network interfaces
  ConsoleConfig console = 6;      // Console/serial configuration
  map<string, string> metadata = 7; // Custom metadata
}
```

### VM States

```protobuf
enum VmState {
  VM_STATE_UNSPECIFIED = 0;
  VM_STATE_CREATED = 1;     // VM created but not started
  VM_STATE_RUNNING = 2;     // VM is running
  VM_STATE_PAUSED = 3;      // VM is paused
  VM_STATE_SHUTDOWN = 4;    // VM is stopped
}
```

## Backend Interface

All hypervisor backends must implement the `Backend` interface defined in `internal/backend/types/backend.go`:

```go
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
```

## Adding a New Backend

To add support for a new hypervisor (e.g., "QEMU"), follow these steps:

### 1. Create Backend Package Structure

```
internal/backend/qemu/
├── client.go     # Main backend implementation
├── types.go      # QEMU-specific types and conversions
└── config.go     # Configuration mapping (optional)
```

### 2. Implement the Backend Interface

Create `internal/backend/qemu/client.go`:

```go
package qemu

import (
    "context"
    "log/slog"

    "vmm-controlplane/internal/backend/types"
    vmmv1 "vmm-controlplane/gen/vmm/v1"
)

// Client implements the Backend interface for QEMU
type Client struct {
    endpoint string
    logger   *slog.Logger
    // Add QEMU-specific fields (QMP socket, libvirt connection, etc.)
}

// NewClient creates a new QEMU backend client
func NewClient(endpoint string, logger *slog.Logger) *Client {
    return &Client{
        endpoint: endpoint,
        logger:   logger.With("backend", "qemu"),
    }
}

// CreateVM creates a new VM instance using QEMU
func (c *Client) CreateVM(ctx context.Context, config *vmmv1.VmConfig) (string, error) {
    c.logger.LogAttrs(ctx, slog.LevelInfo, "creating qemu vm",
        slog.Int("vcpus", int(config.Cpu.VcpuCount)),
        slog.Int64("memory_bytes", config.Memory.SizeBytes),
    )

    // Convert unified config to QEMU command line or QMP commands
    qemuArgs := c.convertVMMConfigToQEMU(config)

    // Launch QEMU process or send QMP commands
    vmID, err := c.launchQEMUVM(ctx, qemuArgs)
    if err != nil {
        return "", fmt.Errorf("failed to create qemu vm: %w", err)
    }

    return vmID, nil
}

// Implement remaining Backend interface methods...
// DeleteVM, BootVM, ShutdownVM, PauseVM, ResumeVM, RebootVM, GetVMInfo, Ping

// Helper methods for QEMU-specific operations
func (c *Client) convertVMMConfigToQEMU(config *vmmv1.VmConfig) []string {
    // Convert unified config to QEMU command line arguments
    args := []string{"qemu-system-x86_64"}

    // CPU configuration
    if config.Cpu != nil {
        args = append(args, "-smp", fmt.Sprintf("%d", config.Cpu.VcpuCount))
    }

    // Memory configuration
    if config.Memory != nil {
        memMB := config.Memory.SizeBytes / (1024 * 1024)
        args = append(args, "-m", fmt.Sprintf("%d", memMB))
    }

    // Boot configuration
    if config.Boot != nil {
        if config.Boot.KernelPath != "" {
            args = append(args, "-kernel", config.Boot.KernelPath)
        }
        if config.Boot.InitrdPath != "" {
            args = append(args, "-initrd", config.Boot.InitrdPath)
        }
        if config.Boot.KernelArgs != "" {
            args = append(args, "-append", config.Boot.KernelArgs)
        }
    }

    // Storage devices
    for i, storage := range config.Storage {
        if storage.IsRootDevice {
            args = append(args, "-drive",
                fmt.Sprintf("file=%s,format=raw,if=virtio", storage.Path))
        }
    }

    // Network interfaces
    for _, network := range config.Network {
        args = append(args, "-netdev",
            fmt.Sprintf("tap,id=%s,ifname=%s", network.Id, network.TapDevice))
        args = append(args, "-device",
            fmt.Sprintf("virtio-net-pci,netdev=%s,mac=%s", network.Id, network.MacAddress))
    }

    return args
}

func (c *Client) launchQEMUVM(ctx context.Context, args []string) (string, error) {
    // Implementation depends on how you want to manage QEMU:
    // - Direct process execution
    // - QMP (QEMU Machine Protocol) over Unix socket
    // - Libvirt integration
    // - QEMU Guest Agent communication

    // Example: Direct process execution
    vmID := fmt.Sprintf("qemu-vm-%d", time.Now().Unix())

    cmd := exec.CommandContext(ctx, args[0], args[1:]...)
    if err := cmd.Start(); err != nil {
        return "", fmt.Errorf("failed to start qemu process: %w", err)
    }

    // Store process reference for management
    c.storeVMProcess(vmID, cmd.Process)

    return vmID, nil
}

// Ensure Client implements Backend interface
var _ types.Backend = (*Client)(nil)
```

### 3. Add Backend Type Constant

Update `internal/backend/types/backend.go`:

```go
const (
    BackendTypeCloudHypervisor BackendType = "cloudhypervisor"
    BackendTypeFirecracker     BackendType = "firecracker"
    BackendTypeQEMU           BackendType = "qemu"  // Add your backend
)
```

### 4. Add Configuration Support

Update `internal/config/config.go`:

```go
type BackendConfig struct {
    Type            types.BackendType
    CloudHypervisor CloudHypervisorConfig
    Firecracker     FirecrackerConfig
    QEMU           QEMUConfig  // Add config struct
}

type QEMUConfig struct {
    // QMP socket path or libvirt URI
    Endpoint string
    // QEMU binary path (optional)
    BinaryPath string
    // Additional QEMU-specific settings
    EnableKVM bool
}
```

Add environment variable support:

```go
func LoadConfigWithSocketPath(socketPath string) (*Config, error) {
    chEndpoint := getEnvOrDefault("VMCP_CH_ENDPOINT", "unix:///tmp/ch.sock")
    fcEndpoint := getEnvOrDefault("VMCP_FC_ENDPOINT", "unix:///tmp/firecracker.sock")
    qemuEndpoint := getEnvOrDefault("VMCP_QEMU_ENDPOINT", "qemu:///system")  // Add QEMU

    // ... rest of configuration

    Backend: BackendConfig{
        Type: types.BackendType(getEnvOrDefault("VMCP_BACKEND", string(types.BackendTypeCloudHypervisor))),
        CloudHypervisor: CloudHypervisorConfig{
            Endpoint: chEndpoint,
        },
        Firecracker: FirecrackerConfig{
            Endpoint: fcEndpoint,
        },
        QEMU: QEMUConfig{  // Add QEMU config
            Endpoint: qemuEndpoint,
            BinaryPath: getEnvOrDefault("VMCP_QEMU_BINARY", "qemu-system-x86_64"),
            EnableKVM: getEnvOrDefault("VMCP_QEMU_KVM", "true") == "true",
        },
    },
}
```

### 5. Register Backend in Main

Update `cmd/api/main.go`:

```go
import (
    "vmm-controlplane/internal/backend/cloudhypervisor"
    "vmm-controlplane/internal/backend/firecracker"
    "vmm-controlplane/internal/backend/qemu"  // Add import
)

// Initialize backend based on configuration
var backend types.Backend
switch cfg.Backend.Type {
case types.BackendTypeCloudHypervisor:
    backend = cloudhypervisor.NewClient(cfg.Backend.CloudHypervisor.Endpoint, logger)
case types.BackendTypeFirecracker:
    backend = firecracker.NewClient(cfg.Backend.Firecracker.Endpoint, logger)
case types.BackendTypeQEMU:  // Add QEMU case
    backend = qemu.NewClient(cfg.Backend.QEMU.Endpoint, logger)
default:
    logger.Error("unsupported backend type", slog.String("backend", string(cfg.Backend.Type)))
    os.Exit(1)
}
```

### 6. Add Validation

Update the validation in `config.go`:

```go
func (c *Config) Validate() error {
    switch c.Backend.Type {
    case types.BackendTypeCloudHypervisor:
        if c.Backend.CloudHypervisor.Endpoint == "" {
            return fmt.Errorf("cloud hypervisor endpoint is required")
        }
    case types.BackendTypeFirecracker:
        if c.Backend.Firecracker.Endpoint == "" {
            return fmt.Errorf("firecracker endpoint is required")
        }
    case types.BackendTypeQEMU:  // Add QEMU validation
        if c.Backend.QEMU.Endpoint == "" {
            return fmt.Errorf("qemu endpoint is required")
        }
    default:
        return fmt.Errorf("unsupported backend type: %s", c.Backend.Type)
    }
    return nil
}
```

## Backend Implementation Guidelines

### Configuration Mapping

Each backend needs to map the unified `VmConfig` to its native format:

- **Resource allocation**: Map CPU count, memory size, and topology
- **Boot configuration**: Handle kernel, initrd, and boot arguments appropriately
- **Storage devices**: Convert to backend-specific disk attachment format
- **Network interfaces**: Map to backend's networking model
- **Console/serial**: Configure according to backend capabilities

### State Management

Map backend-specific VM states to the unified `VmState` enum:

```go
func (c *Client) backendStateToUnified(state string) vmmv1.VmState {
    switch state {
    case "created", "stopped":
        return vmmv1.VmState_VM_STATE_CREATED
    case "running":
        return vmmv1.VmState_VM_STATE_RUNNING
    case "paused":
        return vmmv1.VmState_VM_STATE_PAUSED
    case "shutdown", "halted":
        return vmmv1.VmState_VM_STATE_SHUTDOWN
    default:
        return vmmv1.VmState_VM_STATE_UNSPECIFIED
    }
}
```

### Error Handling

Use structured logging and wrap errors with context:

```go
if err := c.doSomething(); err != nil {
    c.logger.LogAttrs(ctx, slog.LevelError, "operation failed",
        slog.String("vm_id", vmID),
        slog.String("error", err.Error()),
    )
    return fmt.Errorf("operation failed: %w", err)
}
```

### Health Checks

Implement meaningful health checks for your backend:

```go
func (c *Client) Ping(ctx context.Context) error {
    // Check if backend daemon is responsive
    // Verify API connectivity
    // Test basic operations if needed
    return nil
}
```

## Testing Your Backend

1. **Unit tests**: Test configuration conversion and state mapping
2. **Integration tests**: Test against real backend instances
3. **Health checks**: Verify `/_/health` endpoint works correctly

Example usage:

```bash
# Set backend type
export VMCP_BACKEND=qemu
export VMCP_QEMU_ENDPOINT=qemu:///system

# Start the service
go run ./cmd/api

# Test health endpoint
curl http://localhost:8080/_/health
```

## Best Practices

1. **Use structured logging** with consistent field names
2. **Handle backend-specific errors** gracefully
3. **Map configurations** accurately between unified and native formats
4. **Implement proper cleanup** in DeleteVM
5. **Support graceful shutdown** where possible
6. **Add comprehensive validation** for backend-specific requirements
7. **Document backend-specific behavior** and limitations
