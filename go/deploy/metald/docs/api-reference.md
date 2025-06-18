# Metald API Reference

## Overview

Metald provides a ConnectRPC (gRPC-compatible) API for virtual machine lifecycle management. The API supports gRPC, gRPC-Web, and Connect protocols, making it accessible from various client environments.

## Base Configuration

- **Protocol**: ConnectRPC (gRPC-compatible)
- **Default Port**: 8080
- **Base Path**: `/`
- **Service Name**: `vmprovisioner.v1.VmService`
- **Content Type**: `application/json` or `application/proto`

## Authentication

### Requirements

All API endpoints (except health checks) require authentication via Bearer token:

```
Authorization: Bearer <token>
```

### Development Authentication

For development, use tokens in the format:
```
Bearer dev_customer_<customer_id>
```

Example:
```bash
curl -H "Authorization: Bearer dev_customer_123" http://localhost:8080/v1/vms
```

### Production Authentication

In production, integrate with your authentication provider. See [Authentication Guide](development/authentication.md) for implementation details.

### Customer Isolation

- Each request is isolated by customer ID extracted from the token
- VMs are strictly isolated between customers
- List operations only return resources owned by the authenticated customer

## API Endpoints

### VM Management

#### CreateVm

Creates a new virtual machine in the CREATED state.

**Endpoint**: `/vmprovisioner.v1.VmService/CreateVm`

**Request**:
```json
{
  "vm_id": "string",              // Optional, auto-generated if not provided
  "config": {
    "cpu": {
      "vcpu_count": 2,            // Number of vCPUs (1-32)
      "cpu_template": "T2"        // Optional: T2, T2S, T2CL, T2A, V1N1
    },
    "memory": {
      "size_bytes": 1073741824    // Memory in bytes (min: 128MB)
    },
    "boot": {
      "kernel_path": "/assets/vmlinux-5.10",
      "kernel_args": "console=ttyS0 reboot=k panic=1",
      "initrd_path": "/assets/initrd"  // Optional
    },
    "storage": [{
      "device_id": "rootfs",
      "path_on_host": "/assets/rootfs.ext4",
      "is_read_only": false,
      "is_root_device": true
    }],
    "network": [{
      "interface_id": "eth0",
      "type": "TAP",
      "guest_mac": "auto",        // Or specific MAC address
      "rate_limit": {
        "bandwidth_bytes_per_second": 104857600,  // 100 Mbps
        "burst_bytes": 1048576,
        "refill_time_millis": 100
      }
    }],
    "console": {
      "type": "VIRTIO_CONSOLE",
      "output_file": "/tmp/vm-console.log"
    },
    "metadata": {
      "name": "my-vm",
      "project": "test"
    }
  },
  "customer_id": "customer-123"
}
```

**Response**:
```json
{
  "vm_id": "vm_1234567890",
  "state": "VM_STATE_CREATED"
}
```

**Errors**:
- `INVALID_ARGUMENT`: Invalid configuration
- `RESOURCE_EXHAUSTED`: Resource limits exceeded
- `INTERNAL`: Backend failure

#### BootVm

Starts a created virtual machine.

**Endpoint**: `/vmprovisioner.v1.VmService/BootVm`

**Request**:
```json
{
  "vm_id": "vm_1234567890"
}
```

**Response**:
```json
{
  "success": true,
  "state": "VM_STATE_RUNNING"
}
```

**Errors**:
- `NOT_FOUND`: VM not found
- `PERMISSION_DENIED`: VM not owned by customer
- `FAILED_PRECONDITION`: VM not in CREATED state

#### GetVmInfo

Retrieves detailed information about a VM.

**Endpoint**: `/vmprovisioner.v1.VmService/GetVmInfo`

**Request**:
```json
{
  "vm_id": "vm_1234567890"
}
```

**Response**:
```json
{
  "vm_id": "vm_1234567890",
  "config": { /* Full VmConfig */ },
  "state": "VM_STATE_RUNNING",
  "metrics": {
    "cpu_time_nanos": 1234567890,
    "memory_usage_bytes": 536870912,
    "disk_read_bytes": 1048576,
    "disk_write_bytes": 2097152,
    "network_rx_bytes": 1024,
    "network_tx_bytes": 2048
  },
  "backend_info": {
    "backend_type": "firecracker",
    "process_id": "12345",
    "socket_path": "/tmp/firecracker.sock"
  },
  "network_info": {
    "interfaces": [{
      "interface_id": "eth0",
      "tap_device": "tap_12345678",
      "guest_mac": "aa:bb:cc:dd:ee:ff",
      "host_device_name": "vh_12345678"
    }],
    "namespace": "ns_vm_12345678"
  }
}
```

#### ListVms

Lists all VMs owned by the authenticated customer.

**Endpoint**: `/vmprovisioner.v1.VmService/ListVms`

**Request**:
```json
{
  "state_filter": ["VM_STATE_RUNNING", "VM_STATE_PAUSED"],
  "page_size": 50,
  "page_token": ""
}
```

**Response**:
```json
{
  "vms": [
    {
      "vm_id": "vm_1234567890",
      "config": { /* VmConfig */ },
      "state": "VM_STATE_RUNNING",
      "created_at": "2025-06-18T12:00:00Z"
    }
  ],
  "next_page_token": "eyJvZmZzZXQiOjUwfQ==",
  "total_count": 150
}
```

#### ShutdownVm

Gracefully shuts down a running VM.

**Endpoint**: `/vmprovisioner.v1.VmService/ShutdownVm`

**Request**:
```json
{
  "vm_id": "vm_1234567890",
  "force": false,
  "timeout_seconds": 30
}
```

**Response**:
```json
{
  "success": true,
  "state": "VM_STATE_SHUTDOWN"
}
```

#### DeleteVm

Removes a VM and cleans up all resources.

**Endpoint**: `/vmprovisioner.v1.VmService/DeleteVm`

**Request**:
```json
{
  "vm_id": "vm_1234567890",
  "force": false
}
```

**Response**:
```json
{
  "success": true
}
```

**Errors**:
- `FAILED_PRECONDITION`: VM still running (use force=true)

#### PauseVm

Pauses a running VM.

**Endpoint**: `/vmprovisioner.v1.VmService/PauseVm`

**Request**:
```json
{
  "vm_id": "vm_1234567890"
}
```

**Response**:
```json
{
  "success": true,
  "state": "VM_STATE_PAUSED"
}
```

#### ResumeVm

Resumes a paused VM.

**Endpoint**: `/vmprovisioner.v1.VmService/ResumeVm`

**Request**:
```json
{
  "vm_id": "vm_1234567890"
}
```

**Response**:
```json
{
  "success": true,
  "state": "VM_STATE_RUNNING"
}
```

#### RebootVm

Reboots a running VM.

**Endpoint**: `/vmprovisioner.v1.VmService/RebootVm`

**Request**:
```json
{
  "vm_id": "vm_1234567890",
  "force": false
}
```

**Response**:
```json
{
  "success": true,
  "state": "VM_STATE_RUNNING"
}
```

### Health & Monitoring

#### Health Check

Returns service health status.

**Endpoint**: `GET /health`

**Authentication**: Not required

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-06-18T12:00:00Z",
  "version": "0.2.0",
  "backend": {
    "type": "firecracker",
    "status": "healthy"
  },
  "system": {
    "uptime_seconds": 3600,
    "memory_usage_mb": 512,
    "cpu_usage_percent": 25.5
  },
  "checks": {
    "backend_ping": {
      "status": "healthy",
      "duration_ms": 5,
      "timestamp": "2025-06-18T12:00:00Z"
    }
  }
}
```

**Status Values**:
- `healthy`: All systems operational
- `degraded`: Partial functionality available
- `unhealthy`: Service unavailable

#### Metrics

Prometheus metrics endpoint.

**Endpoint**: `GET /metrics`

**Authentication**: Not required

**Response**: Prometheus text format

Key metrics:
- `unkey_metald_vm_create_requests_total`
- `unkey_metald_vm_boot_duration_seconds`
- `unkey_metald_active_vms`
- `unkey_metald_backend_errors_total`

## Data Types

### VmState Enum

| Value | Name | Description |
|-------|------|-------------|
| 0 | VM_STATE_UNSPECIFIED | Unknown state |
| 1 | VM_STATE_CREATED | VM created but not started |
| 2 | VM_STATE_RUNNING | VM is running |
| 3 | VM_STATE_PAUSED | VM is paused |
| 4 | VM_STATE_SHUTDOWN | VM is stopped |

### VmConfig Structure

Complete VM configuration specification:

```protobuf
message VmConfig {
  CpuConfig cpu = 1;
  MemoryConfig memory = 2;
  BootConfig boot = 3;
  repeated StorageDevice storage = 4;
  repeated NetworkInterface network = 5;
  ConsoleConfig console = 6;
  map<string, string> metadata = 7;
}
```

### Network Configuration

#### IPv4 Example
```json
{
  "network": [{
    "interface_id": "eth0",
    "type": "TAP",
    "guest_mac": "auto",
    "host_ipv4": {
      "address": "192.168.1.1",
      "prefix_length": 24,
      "gateway": "192.168.1.254"
    },
    "guest_ipv4": {
      "address": "192.168.1.100",
      "prefix_length": 24
    }
  }]
}
```

#### IPv6 Example
```json
{
  "network": [{
    "interface_id": "eth0",
    "type": "TAP",
    "guest_mac": "auto",
    "host_ipv6": {
      "address": "2001:db8:1::1",
      "prefix_length": 64,
      "gateway": "2001:db8:1::ffff"
    },
    "guest_ipv6": {
      "address": "2001:db8:1::100",
      "prefix_length": 64
    }
  }]
}
```

## Error Handling

### Standard Error Codes

| Code | Name | HTTP Status | Description |
|------|------|-------------|-------------|
| 0 | OK | 200 | Success |
| 3 | INVALID_ARGUMENT | 400 | Invalid request parameters |
| 5 | NOT_FOUND | 404 | Resource not found |
| 6 | ALREADY_EXISTS | 409 | Resource already exists |
| 7 | PERMISSION_DENIED | 403 | Not authorized for resource |
| 8 | RESOURCE_EXHAUSTED | 429 | Limits exceeded |
| 9 | FAILED_PRECONDITION | 412 | Invalid state for operation |
| 13 | INTERNAL | 500 | Internal server error |
| 16 | UNAUTHENTICATED | 401 | Missing/invalid auth |

### Error Response Format

```json
{
  "code": "permission_denied",
  "message": "VM vm_1234567890 not found or not owned by customer",
  "details": [
    {
      "@type": "type.googleapis.com/google.rpc.ErrorInfo",
      "reason": "VM_NOT_OWNED",
      "domain": "metald.unkey.com",
      "metadata": {
        "vm_id": "vm_1234567890",
        "customer_id": "customer-123"
      }
    }
  ]
}
```

## Rate Limiting

### API Level
Currently no API-level rate limiting is implemented. This is typically handled by an API gateway in production.

### Network Level
Individual VM network interfaces support rate limiting:

```json
{
  "rate_limit": {
    "bandwidth_bytes_per_second": 104857600,  // 100 Mbps
    "burst_bytes": 1048576,                   // 1 MB burst
    "refill_time_millis": 100                 // Token bucket refill
  }
}
```

## Client Libraries

### Go Client (Recommended)

```go
import (
    "connectrpc.com/connect"
    vmprovisionerv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
    "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
)

client := vmprovisionerv1connect.NewVmServiceClient(
    http.DefaultClient,
    "http://localhost:8080",
)

req := &vmprovisionerv1.CreateVmRequest{
    Config: &vmprovisionerv1.VmConfig{
        Cpu: &vmprovisionerv1.CpuConfig{VcpuCount: 2},
        Memory: &vmprovisionerv1.MemoryConfig{SizeBytes: 1073741824},
    },
}

resp, err := client.CreateVm(ctx, connect.NewRequest(req))
```

### gRPC Clients

Any standard gRPC client in any language can connect to the service.

### HTTP/JSON

For simple REST-like access:

```bash
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_123" \
  -d '{"config": {...}}'
```

## SDK References

- **Go SDK**: See generated code in `gen/vmprovisioner/v1/`
- **Proto Files**: Located in `proto/vmprovisioner/v1/`
- **ConnectRPC**: [connectrpc.com](https://connectrpc.com)

## OpenAPI/Swagger

OpenAPI specification is not currently available as the service uses Protocol Buffers. Use the proto files for schema definitions.

## Versioning

- **API Version**: v1
- **Protocol**: ConnectRPC/gRPC
- **Backward Compatibility**: Maintained within major versions
- **Deprecation Policy**: 6 months notice for breaking changes