# Metald Client

A Go client library for the metald VM provisioning service with built-in SPIFFE/SPIRE socket integration and tenant isolation.

## Features

- **SPIFFE/SPIRE Integration**: Automatic mTLS authentication using SPIFFE workload API
- **Tenant Isolation**: Customer ID authentication for multi-tenant environments
- **Complete VM Lifecycle**: Create, boot, pause, resume, reboot, shutdown, delete operations
- **TLS Modes**: Support for SPIFFE, file-based, and disabled TLS modes
- **High-Level Interface**: Clean Go API wrapping ConnectRPC/protobuf internals
- **Connection Management**: Automatic certificate rotation and connection pooling

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/unkeyed/unkey/go/deploy/metald/client"
)

func main() {
    ctx := context.Background()

    // Create client with SPIFFE authentication
    config := client.Config{
        ServerAddress:    "https://metald:8080",
        UserID:           "my-user-123",
        TenantID:         "my-tenant-456",
        TLSMode:          "spiffe",
        SPIFFESocketPath: "/var/lib/spire/agent/agent.sock",
        Timeout:          30 * time.Second,
    }

    metaldClient, err := client.New(ctx, config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer metaldClient.Close()

    // Create VM using a template
    vmConfig := client.NewVMConfigFromTemplate(client.TemplateStandard).
        WithCPU(4, 8).
        WithMemoryGB(4, 16, true).
        AddRootStorage("/opt/vm-assets/rootfs.ext4").
        AddDefaultNetwork().
        AddMetadata("purpose", "web-server").
        Build()

    createResp, err := metaldClient.CreateVM(ctx, &client.CreateVMRequest{
        Config: vmConfig,
    })
    if err != nil {
        log.Fatalf("Failed to create VM: %v", err)
    }

    bootResp, err := metaldClient.BootVM(ctx, createResp.VMID)
    if err != nil {
        log.Fatalf("Failed to boot VM: %v", err)
    }

    log.Printf("VM %s is now %s", createResp.VMID, bootResp.State)
}
```

### Using VM Configuration Builder

```go
// Build a custom VM configuration
vmConfig := client.NewVMConfigBuilder().
    WithCPU(8, 16).                                    // 8 vCPUs, max 16
    WithMemoryGB(16, 64, true).                        // 16GB RAM, max 64GB, hotplug enabled
    WithDefaultBoot("console=ttyS0 reboot=k panic=1"). // Standard boot config
    AddRootStorage("/opt/vm-assets/ubuntu-rootfs.ext4"). // Root filesystem
    AddDataStorage("data", "/opt/vm-assets/data.ext4", false). // Additional storage
    AddDefaultNetwork().                               // Standard dual-stack network
    WithDefaultConsole("/var/log/vm-console.log").     // Console logging
    AddMetadata("environment", "production").          // Custom metadata
    AddMetadata("owner", "platform-team").
    Build()

// Validate configuration before use
builder := client.NewVMConfigBuilder()
builder.config = vmConfig
if err := builder.Validate(); err != nil {
    log.Fatalf("Invalid VM configuration: %v", err)
}
```

## Configuration

### TLS Modes

#### SPIFFE Mode (Recommended for Production)
```go
config := client.Config{
    ServerAddress:    "https://metald:8080",
    CustomerID:       "customer-123",
    TLSMode:          "spiffe",
    SPIFFESocketPath: "/var/lib/spire/agent/agent.sock",
}
```

#### File-based TLS Mode
```go
config := client.Config{
    ServerAddress: "https://metald:8080",
    CustomerID:    "customer-123",
    TLSMode:       "file",
    TLSCertFile:   "/etc/ssl/certs/client.crt",
    TLSKeyFile:    "/etc/ssl/private/client.key",
    TLSCAFile:     "/etc/ssl/certs/ca.crt",
}
```

#### Disabled TLS Mode (Development Only)
```go
config := client.Config{
    ServerAddress: "http://localhost:8080",
    CustomerID:    "dev-customer",
    TLSMode:       "disabled",
}
```

### Environment Variables

The client respects standard environment variable patterns:

```bash
# SPIFFE socket path (if not specified in config)
export UNKEY_METALD_SPIFFE_SOCKET="/run/spire/sockets/agent.sock"

# Server address
export UNKEY_METALD_SERVER_ADDRESS="https://metald.internal:8080"

# User and tenant IDs for authentication
export UNKEY_METALD_USER_ID="user-123"
export UNKEY_METALD_TENANT_ID="tenant-456"
```

## VM Configuration

### Built-in Templates

The client provides several built-in VM templates for common use cases:

```go
// Minimal VM (512MB RAM, 1 vCPU)
config := client.NewVMConfigFromTemplate(client.TemplateMinimal).Build()

// Standard VM (2GB RAM, 2 vCPUs) 
config := client.NewVMConfigFromTemplate(client.TemplateStandard).Build()

// High-CPU VM (4GB RAM, 8 vCPUs)
config := client.NewVMConfigFromTemplate(client.TemplateHighCPU).Build()

// High-Memory VM (16GB RAM, 4 vCPUs)
config := client.NewVMConfigFromTemplate(client.TemplateHighMemory).Build()

// Development VM (8GB RAM, 4 vCPUs, extra storage)
config := client.NewVMConfigFromTemplate(client.TemplateDevelopment).Build()
```

### Configuration Builder Methods

#### CPU Configuration
```go
builder.WithCPU(vcpuCount, maxVcpuCount uint32)
```

#### Memory Configuration
```go
// Set memory in bytes
builder.WithMemory(sizeBytes, maxSizeBytes uint64, hotplugEnabled bool)

// Set memory in MB (convenience method)
builder.WithMemoryMB(sizeMB, maxSizeMB uint64, hotplugEnabled bool)

// Set memory in GB (convenience method)  
builder.WithMemoryGB(sizeGB, maxSizeGB uint64, hotplugEnabled bool)
```

#### Boot Configuration
```go
// Full boot configuration
builder.WithBoot(kernelPath, initrdPath, kernelArgs string)

// Default boot with custom kernel args
builder.WithDefaultBoot(kernelArgs string)
```

#### Storage Configuration
```go
// Add storage device
builder.AddStorage(id, path string, readOnly, isRoot bool, interfaceType string)

// Add root filesystem (convenience method)
builder.AddRootStorage(path string)

// Add data storage (convenience method)
builder.AddDataStorage(id, path string, readOnly bool)

// Add storage with custom options
builder.AddStorageWithOptions(id, path string, readOnly, isRoot bool, 
    interfaceType string, options map[string]string)
```

#### Network Configuration
```go
// Add network interface
builder.AddNetwork(id, interfaceType string, mode vmprovisionerv1.NetworkMode)

// Add default dual-stack network
builder.AddDefaultNetwork()

// Add IPv4-only network
builder.AddIPv4OnlyNetwork(id string)

// Add IPv6-only network  
builder.AddIPv6OnlyNetwork(id string)

// Add network with custom IPv4/IPv6 configuration
builder.AddNetworkWithCustomConfig(id, interfaceType string, mode vmprovisionerv1.NetworkMode,
    ipv4Config *vmprovisionerv1.IPv4Config, ipv6Config *vmprovisionerv1.IPv6Config)
```

#### Console Configuration
```go
// Configure console
builder.WithConsole(enabled bool, output, consoleType string)

// Default console configuration
builder.WithDefaultConsole(output string)

// Disable console
builder.DisableConsole()
```

#### Metadata
```go
// Add single metadata entry
builder.AddMetadata(key, value string)

// Set all metadata at once
builder.WithMetadata(metadata map[string]string)
```

#### Docker Integration
```go
// Configure VM for Docker image
builder.ForDockerImage(imageName string)
```

### Configuration Files

#### Creating Configuration Files

You can save VM configurations as JSON files for reuse:

```go
// Create configuration
config := client.NewVMConfigFromTemplate(client.TemplateStandard).
    WithCPU(4, 8).
    WithMemoryGB(8, 32, true).
    Build()

// Convert to file format
configFile := client.FromVMConfig(config, "web-server", "Configuration for web server VMs")

// Save to file
err := client.SaveVMConfigToFile(configFile, "configs/web-server.json")
```

#### Loading Configuration Files

```go
// Load configuration from file
configFile, err := client.LoadVMConfigFromFile("configs/web-server.json")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Convert to VM configuration
vmConfig, err := configFile.ToVMConfig()
if err != nil {
    log.Fatalf("Failed to convert config: %v", err)
}

// Use in VM creation
resp, err := client.CreateVM(ctx, &client.CreateVMRequest{
    Config: vmConfig,
})
```

#### Configuration File Format

```json
{
  "name": "web-server",
  "description": "Configuration for web server VMs",
  "template": "standard",
  "cpu": {
    "vcpu_count": 4,
    "max_vcpu_count": 8
  },
  "memory": {
    "size_mb": 8192,
    "max_size_mb": 32768,
    "hotplug_enabled": true
  },
  "boot": {
    "kernel_path": "/opt/vm-assets/vmlinux",
    "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
  },
  "storage": [
    {
      "id": "rootfs",
      "path": "/opt/vm-assets/rootfs.ext4",
      "read_only": false,
      "is_root_device": true,
      "interface_type": "virtio-blk"
    }
  ],
  "network": [
    {
      "id": "eth0", 
      "interface_type": "virtio-net",
      "mode": "dual_stack",
      "ipv4": {
        "dhcp": true
      },
      "ipv6": {
        "slaac": true,
        "privacy_extensions": true
      }
    }
  ],
  "console": {
    "enabled": true,
    "output": "/tmp/vm-console.log",
    "console_type": "serial"
  },
  "metadata": {
    "purpose": "web-server",
    "environment": "production"
  }
}
```

## API Reference

### VM Lifecycle Operations

#### CreateVM
```go
resp, err := client.CreateVM(ctx, &client.CreateVMRequest{
    VMID:   "optional-vm-id", // Auto-generated if empty
    Config: vmConfig,
})
```

#### BootVM
```go
resp, err := client.BootVM(ctx, vmID)
```

#### ShutdownVM
```go
resp, err := client.ShutdownVM(ctx, &client.ShutdownVMRequest{
    VMID:           vmID,
    Force:          false,
    TimeoutSeconds: 30,
})
```

#### DeleteVM
```go
resp, err := client.DeleteVM(ctx, &client.DeleteVMRequest{
    VMID:  vmID,
    Force: false,
})
```

### VM Information Operations

#### GetVMInfo
```go
vmInfo, err := client.GetVMInfo(ctx, vmID)
// Returns detailed VM info including config, metrics, and network info
```

#### ListVMs
```go
resp, err := client.ListVMs(ctx, &client.ListVMsRequest{
    PageSize:  50,
    PageToken: "", // Empty for first page
})
// Returns paginated list of VMs for the authenticated customer
```

### VM Control Operations

#### PauseVM / ResumeVM
```go
pauseResp, err := client.PauseVM(ctx, vmID)
resumeResp, err := client.ResumeVM(ctx, vmID)
```

#### RebootVM
```go
resp, err := client.RebootVM(ctx, &client.RebootVMRequest{
    VMID:  vmID,
    Force: false, // Graceful vs forced reboot
})
```

## Authentication & Tenant Isolation

### Customer ID Authentication

The client automatically adds the appropriate `Authorization` header to all requests:

- **Development Mode**: `Bearer dev_customer_<customer_id>`
- **Production Mode**: Would use real JWT tokens or API keys

### SPIFFE Workload Identity

When using SPIFFE mode, the client:

1. Connects to the SPIFFE agent socket (default: `/var/lib/spire/agent/agent.sock`)
2. Retrieves X.509 SVIDs for mTLS authentication
3. Automatically rotates certificates as they expire
4. Validates server certificates against the same trust domain

### Tenant Isolation

All VM operations are automatically scoped to the authenticated customer:

- VMs are only visible to their owning customer
- Customer ID is extracted from authentication tokens
- Database queries include customer-scoped filtering

## Error Handling

The client provides structured error handling:

```go
vmInfo, err := client.GetVMInfo(ctx, "non-existent-vm")
if err != nil {
    // Error includes details about the failure
    log.Printf("Failed to get VM info: %v", err)
    
    // ConnectRPC errors can be inspected for status codes
    if connectErr := new(connect.Error); errors.As(err, &connectErr) {
        switch connectErr.Code() {
        case connect.CodeNotFound:
            log.Println("VM not found")
        case connect.CodePermissionDenied:
            log.Println("Access denied - check customer ID")
        }
    }
}
```

## Performance Considerations

### Connection Reuse
- HTTP/2 connection pooling is handled automatically
- Single client instance can handle multiple concurrent requests
- TLS handshakes are minimized through connection reuse

### Certificate Caching
```go
config := client.Config{
    // ... other config
    EnableCertCaching: true,
    CertCacheTTL:      5 * time.Second,
}
```

### Timeouts
```go
config := client.Config{
    // ... other config
    Timeout: 30 * time.Second, // HTTP client timeout
}

// Per-request timeouts
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
resp, err := client.CreateVM(ctx, req)
```

## Command Line Interface

The metald-cli tool provides a command-line interface for VM operations:

### Basic VM Operations

```bash
# Create and boot a VM with default settings
metald-cli create-and-boot

# Create VM with specific template
metald-cli -template=high-cpu create-and-boot

# Create VM with custom resources
metald-cli -template=standard -cpu=8 -memory=16384 create-and-boot

# Create VM for Docker image
metald-cli -docker-image=nginx:alpine create-and-boot
```

### Using Configuration Files

```bash
# Generate a configuration file
metald-cli -template=development config-gen > my-vm.json

# Edit the configuration file as needed...

# Create VM from configuration file
metald-cli -config=my-vm.json create-and-boot

# Validate configuration file
metald-cli config-validate my-vm.json
```

### VM Management

```bash
# List all VMs
metald-cli list

# Get detailed VM information
metald-cli info vm-12345

# Control VM state
metald-cli pause vm-12345
metald-cli resume vm-12345
metald-cli reboot vm-12345
metald-cli shutdown vm-12345
metald-cli delete vm-12345
```

### Authentication and TLS

```bash
# Use SPIFFE authentication (default)
metald-cli -user=my-user -tenant=my-tenant list

# Use disabled TLS for development
metald-cli -tls-mode=disabled -server=http://localhost:8080 list

# Use file-based TLS
metald-cli -tls-mode=file -tls-cert=client.crt -tls-key=client.key list
```

### Output Formats

```bash
# Human-readable output (default)
metald-cli list

# JSON output for scripting
metald-cli list -json

# Generate configuration with JSON output
metald-cli -template=high-memory config-gen -json
```

### Environment Variables

Set environment variables to avoid repeating common options:

```bash
export UNKEY_METALD_SERVER_ADDRESS="https://metald.prod:8080"
export UNKEY_METALD_USER_ID="production-user"
export UNKEY_METALD_TENANT_ID="production-tenant"
export UNKEY_METALD_TLS_MODE="spiffe"

# Now you can use the CLI without specifying these options
metald-cli create-and-boot
metald-cli list
```

### Configuration Examples

#### High-Performance Web Server
```bash
# Generate config for high-performance web server
metald-cli -template=high-cpu -cpu=16 -memory=32768 config-gen > web-server.json

# Customize the configuration file...
# Add additional storage, network interfaces, etc.

# Create the VM
metald-cli -config=web-server.json create-and-boot
```

#### Database Server
```bash
# High-memory configuration for database
metald-cli -template=high-memory -memory=65536 config-gen > database.json

# Create with specific VM ID
metald-cli -config=database.json create-and-boot db-primary-01
```

#### Development Environment
```bash
# Development VM with Docker support
metald-cli -docker-image=ubuntu:22.04 -template=development create-and-boot dev-env
```

## Testing

The client includes comprehensive examples and can be tested against a local metald instance:

```bash
# Run examples (requires running metald)
go test -v ./client -run Example

# Integration tests
go test -v ./client -tags=integration

# Test CLI tool
cd client/cmd/metald-cli
go build
./metald-cli -help
```

## Security Best Practices

1. **Use SPIFFE in Production**: Always use SPIFFE mode in production environments
2. **Validate Customer IDs**: Ensure customer IDs come from authenticated sources
3. **Network Security**: Deploy metald behind appropriate network security controls
4. **Certificate Management**: Let SPIFFE handle certificate lifecycle automatically
5. **Audit Logging**: All operations are logged with customer context for audit trails

## Troubleshooting

### SPIFFE Connection Issues
```bash
# Check SPIFFE agent status
systemctl status spire-agent

# Test SPIFFE socket connectivity
ls -la /var/lib/spire/agent/agent.sock

# Check SPIFFE ID assignment
/opt/spire/bin/spire-agent api fetch -socketPath /var/lib/spire/agent/agent.sock
```

### TLS Verification Errors
- Ensure trust domain configuration matches between client and server
- Verify SPIFFE agent has proper workload attestation
- Check that certificates are not expired

### Authentication Failures
- Verify customer ID format and validity
- Check that metald has proper authentication configuration
- Ensure customer exists in the system