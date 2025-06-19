# Metald Client Guide

This guide explains how to communicate with metald to create and manage VMs.

## Overview

Metald provides a gRPC API (using ConnectRPC) for VM lifecycle management. The API is defined in protobuf at `/metald/proto/vmprovisioner/v1/vm.proto`.

## Key Components

### 1. API Endpoints

The main service is `VmService` with the following RPCs:
- `CreateVm` - Create a new VM instance
- `BootVm` - Start a created VM
- `ShutdownVm` - Gracefully stop a VM
- `DeleteVm` - Remove a VM
- `PauseVm` / `ResumeVm` - Pause/resume execution
- `RebootVm` - Restart a VM
- `GetVmInfo` - Get detailed VM information
- `ListVms` - List all VMs

### 2. Authentication

Metald uses two authentication mechanisms:

1. **Customer ID Header**: Required `X-Customer-ID` header for tenant isolation
2. **mTLS via SPIFFE/SPIRE**: Optional but recommended for production

### 3. TLS/SPIFFE Configuration

Metald supports three TLS modes:

#### Disabled (Development)
```go
tlsConfig := tlspkg.Config{
    Mode: tlspkg.ModeDisabled,
}
```

#### File-based TLS
```go
tlsConfig := tlspkg.Config{
    Mode:     tlspkg.ModeFile,
    CertFile: "/path/to/cert.pem",
    KeyFile:  "/path/to/key.pem",
    CAFile:   "/path/to/ca.pem",
}
```

#### SPIFFE (Production)
```go
// Using pkg/spiffe directly
spiffeClient, err := spiffe.New(ctx)
httpClient := spiffeClient.HTTPClient()

// Or using pkg/tls provider
tlsConfig := tlspkg.Config{
    Mode:             tlspkg.ModeSPIFFE,
    SPIFFESocketPath: "unix:///run/spire/sockets/agent.sock",
}
```

## Client Examples

### Basic Client Setup

```go
import (
    "connectrpc.com/connect"
    "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
    tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
)

// Create TLS provider
tlsProvider, err := tlspkg.NewProvider(ctx, tlsConfig)
httpClient := tlsProvider.HTTPClient()

// Create metald client
client := vmprovisionerv1connect.NewVmServiceClient(
    httpClient,
    "https://localhost:8080",
    connect.WithInterceptors(
        authInterceptor("customer-123"),
    ),
)
```

### Creating and Booting a VM

```go
// Define VM configuration
vmConfig := &vmprovisionerv1.VmConfig{
    Cpu: &vmprovisionerv1.CpuConfig{
        VcpuCount: 2,
    },
    Memory: &vmprovisionerv1.MemoryConfig{
        SizeBytes: 1024 * 1024 * 1024, // 1GB
    },
    Boot: &vmprovisionerv1.BootConfig{
        KernelPath: "/assets/vmlinux",
        InitrdPath: "/assets/initrd.img",
        KernelArgs: "console=ttyS0 reboot=k panic=1",
    },
    Storage: []*vmprovisionerv1.StorageDevice{
        {
            Id:           "rootfs",
            Path:         "/assets/rootfs.ext4",
            IsRootDevice: true,
        },
    },
    Network: []*vmprovisionerv1.NetworkInterface{
        {
            Id:   "eth0",
            Mode: vmprovisionerv1.NetworkMode_NETWORK_MODE_IPV4_ONLY,
            Ipv4Config: &vmprovisionerv1.IPv4Config{
                Dhcp: true,
            },
        },
    },
}

// Create VM
createResp, err := client.CreateVm(ctx, connect.NewRequest(&vmprovisionerv1.CreateVmRequest{
    Config:     vmConfig,
    CustomerId: "customer-123",
}))

// Boot VM
bootResp, err := client.BootVm(ctx, connect.NewRequest(&vmprovisionerv1.BootVmRequest{
    VmId: createResp.Msg.VmId,
}))
```

## Environment Variables

Metald server configuration via environment variables:

```bash
# Server configuration
UNKEY_METALD_PORT=8080
UNKEY_METALD_ADDRESS=0.0.0.0

# TLS configuration
UNKEY_METALD_TLS_MODE=spiffe
UNKEY_METALD_TLS_SPIFFE_SOCKET_PATH=unix:///run/spire/sockets/agent.sock

# For file-based TLS
UNKEY_METALD_TLS_MODE=file
UNKEY_METALD_TLS_CERT_FILE=/etc/unkey/certs/server.crt
UNKEY_METALD_TLS_KEY_FILE=/etc/unkey/certs/server.key
UNKEY_METALD_TLS_CA_FILE=/etc/unkey/certs/ca.crt
```

## Running the Examples

1. **Basic example with all options**:
   ```bash
   go run examples/metald-client-example.go \
     -endpoint https://localhost:8080 \
     -tls-mode spiffe \
     -action create-and-boot
   ```

2. **SPIFFE-only example**:
   ```bash
   go run examples/metald-spiffe-client.go
   ```

## File Paths

- **Protobuf Definition**: `/metald/proto/vmprovisioner/v1/vm.proto`
- **Generated Go Code**: `/metald/gen/vmprovisioner/v1/`
- **SPIFFE Client**: `/pkg/spiffe/client.go`
- **TLS Provider**: `/pkg/tls/provider.go`
- **Example Clients**: `/examples/metald-*.go`

## Key Insights

1. **Opt-in Security**: TLS/SPIFFE is optional, allowing gradual migration from development to production
2. **Tenant Isolation**: Customer ID header provides basic multi-tenancy
3. **Backend Agnostic**: Same API works for both Firecracker and Cloud Hypervisor backends
4. **Asset Integration**: Metald integrates with assetmanagerd for kernel/rootfs management
5. **Network Management**: Automatic TAP device and network namespace creation for VMs
6. **Billing Integration**: Automatic usage tracking via billaged service