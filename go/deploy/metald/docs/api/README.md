# Metald API Documentation

Metald exposes a ConnectRPC API for virtual machine lifecycle management. This document provides complete API reference with examples.

## Table of Contents

- [Service Overview](#service-overview)
- [Authentication](#authentication)
- [API Reference](#api-reference)
  - [CreateVm](#createvm)
  - [BootVm](#bootvm)
  - [ShutdownVm](#shutdownvm)
  - [DeleteVm](#deletevm)
  - [GetVmInfo](#getvminfo)
  - [ListVms](#listvms)
  - [PauseVm](#pausevm)
  - [ResumeVm](#resumevm)
  - [RebootVm](#rebootvm)
- [Data Types](#data-types)
- [Error Handling](#error-handling)

## Service Overview

**Service**: `vmprovisioner.v1.VmService`  
**Protocol**: ConnectRPC over HTTP/2  
**Default Port**: 8080  
**Content-Type**: `application/json` or `application/proto`

## Authentication and Tenant Isolation

Metald uses a two-header authentication system that separates user identity from tenant data scoping:

### Authentication Header
Identifies and authorizes the user making the request:

```
Authorization: Bearer <secure_token>
```

**Purpose**: Validates WHO you are and WHAT you're allowed to do
- In production: JWT tokens, API keys, or OAuth tokens
- In development: `dev_customer_{user_id}` format for simplicity

### Tenant Isolation Header
Specifies which tenant's data to access:

```
X-Tenant-ID: <tenant_identifier>
```

**Purpose**: Scopes all operations to a specific tenant's resources
- Enables multi-tenant applications where one user can access multiple tenants
- Provides clear data isolation and audit trails
- Keeps tenant information separate from secure authentication credentials

### Security Model

1. **Authentication**: Bearer token validates user identity and permissions
2. **Authorization**: System checks if authenticated user can access specified tenant
3. **Data Scoping**: All operations are scoped to the specified tenant's resources

### Example Request

```bash
curl -X POST https://metald:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Authorization: Bearer <secure_token>" \
  -H "X-Tenant-ID: tenant-123" \
  -H "Content-Type: application/json" \
  -d '{"config": {...}}'
```

The separation ensures security (tokens don't leak tenant info) and flexibility (one user can manage multiple tenants).

## API Reference

### CreateVm

Creates a new virtual machine instance with the specified configuration.

**RPC**: `CreateVm(CreateVmRequest) returns (CreateVmResponse)`  
**Implementation**: [vm.go:48-159](../../../metald/internal/service/vm.go#L48-L159)

#### Request

```protobuf
message CreateVmRequest {
  string vm_id = 1;        // Optional, auto-generated if empty
  VmConfig config = 2;     // Required VM configuration
  string tenant_id = 3;    // Optional, must match X-Tenant-ID header
}
```

#### Response

```protobuf
message CreateVmResponse {
  string vm_id = 1;    // Assigned VM identifier
  VmState state = 2;   // Initial state (VM_STATE_CREATED)
}
```

#### Example

```bash
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <secure_token>" \
  -H "X-Tenant-ID: test123" \
  -d '{
    "config": {
      "cpu": {
        "vcpu_count": 2,
        "max_vcpu_count": 4
      },
      "memory": {
        "size_bytes": 1073741824,
        "hotplug_enabled": true,
        "max_size_bytes": 4294967296
      },
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
      },
      "storage": [{
        "id": "rootfs",
        "path": "/opt/vm-assets/rootfs.ext4",
        "read_only": false,
        "is_root_device": true,
        "interface_type": "virtio-blk"
      }],
      "network": [{
        "id": "eth0",
        "interface_type": "virtio-net",
        "mode": "NETWORK_MODE_DUAL_STACK",
        "ipv4_config": {
          "dhcp": true
        },
        "ipv6_config": {
          "slaac": true,
          "privacy_extensions": true
        }
      }],
      "console": {
        "enabled": true,
        "output": "/tmp/vm-console.log",
        "console_type": "serial"
      },
      "metadata": {
        "app": "web-server",
        "env": "production"
      }
    }
  }'
```

#### Downstream Calls

- Stores VM configuration in SQLite database ([repository.go:51-99](../../../metald/internal/database/repository.go#L51-L99))
- Allocates network resources via network manager
- Prepares VM assets via AssetManagerd (if enabled)

#### Errors

- `InvalidArgument` (3): Missing or invalid configuration
- `Unauthenticated` (16): Missing customer authentication
- `PermissionDenied` (7): Customer ID mismatch
- `ResourceExhausted` (8): Quota exceeded

### BootVm

Starts a previously created virtual machine.

**RPC**: `BootVm(BootVmRequest) returns (BootVmResponse)`  
**Implementation**: [vm.go:161-219](../../../metald/internal/service/vm.go#L161-L219)

#### Request

```protobuf
message BootVmRequest {
  string vm_id = 1;  // Required VM identifier
}
```

#### Response

```protobuf
message BootVmResponse {
  bool success = 1;   // Operation success
  VmState state = 2;  // New state (VM_STATE_RUNNING)
}
```

#### Example

```bash
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/BootVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_test123" \
  -d '{"vm_id": "vm-01HQKP3X5V2Q8Z9R1N4M7BHCFD"}'
```

#### Downstream Calls

- Notifies billaged service of VM start ([collector.go:96](../../../metald/internal/billing/collector.go#L96))
- Starts metrics collection for billing

### ShutdownVm

Gracefully stops a running virtual machine.

**RPC**: `ShutdownVm(ShutdownVmRequest) returns (ShutdownVmResponse)`  
**Implementation**: [vm.go:221-295](../../../metald/internal/service/vm.go#L221-L295)

#### Request

```protobuf
message ShutdownVmRequest {
  string vm_id = 1;           // Required VM identifier
  bool force = 2;             // Force immediate shutdown
  int32 timeout_seconds = 3;  // Graceful shutdown timeout
}
```

#### Response

```protobuf
message ShutdownVmResponse {
  bool success = 1;   // Operation success
  VmState state = 2;  // New state (VM_STATE_SHUTDOWN)
}
```

#### Downstream Calls

- Notifies billaged service of VM stop ([collector.go:165](../../../metald/internal/billing/collector.go#L165))
- Stops metrics collection

### DeleteVm

Removes a virtual machine and cleans up all associated resources.

**RPC**: `DeleteVm(DeleteVmRequest) returns (DeleteVmResponse)`  
**Implementation**: [vm.go:297-361](../../../metald/internal/service/vm.go#L297-L361)

#### Request

```protobuf
message DeleteVmRequest {
  string vm_id = 1;  // Required VM identifier
  bool force = 2;    // Force deletion of running VM
}
```

#### Response

```protobuf
message DeleteVmResponse {
  bool success = 1;  // Operation success
}
```

#### Downstream Calls

- Removes VM from database ([repository.go:372-424](../../../metald/internal/database/repository.go#L372-L424))
- Deallocates network resources
- Cleans up jailer chroot directory

### GetVmInfo

Retrieves detailed information about a virtual machine.

**RPC**: `GetVmInfo(GetVmInfoRequest) returns (GetVmInfoResponse)`  
**Implementation**: [vm.go:430-515](../../../metald/internal/service/vm.go#L430-L515)

#### Request

```protobuf
message GetVmInfoRequest {
  string vm_id = 1;  // Required VM identifier
}
```

#### Response

```protobuf
message GetVmInfoResponse {
  string vm_id = 1;                   // VM identifier
  VmConfig config = 2;                // Full configuration
  VmState state = 3;                  // Current state
  VmMetrics metrics = 4;              // Resource usage metrics
  map<string, string> backend_info = 5; // Backend-specific info
  VmNetworkInfo network_info = 6;     // Network configuration
}
```

#### Example Response

```json
{
  "vm_id": "vm-01HQKP3X5V2Q8Z9R1N4M7BHCFD",
  "state": "VM_STATE_RUNNING",
  "config": { /* full config */ },
  "metrics": {
    "cpu_usage_percent": 45.2,
    "memory_usage_bytes": 536870912,
    "network_stats": {
      "bytes_received": 1048576,
      "bytes_transmitted": 2097152
    },
    "uptime_seconds": 3600
  },
  "network_info": {
    "ip_address": "10.100.1.2",
    "mac_address": "52:54:00:12:34:56",
    "tap_device": "tap-vm-01HQKP3X",
    "gateway": "10.100.0.1"
  }
}
```

### ListVms

Lists virtual machines with optional filtering and pagination.

**RPC**: `ListVms(ListVmsRequest) returns (ListVmsResponse)`  
**Implementation**: [vm.go:517-616](../../../metald/internal/service/vm.go#L517-L616)

#### Request

```protobuf
message ListVmsRequest {
  repeated VmState state_filter = 1;  // Optional state filter
  int32 page_size = 2;                // Results per page (max 100)
  string page_token = 3;              // Pagination token
}
```

#### Response

```protobuf
message ListVmsResponse {
  repeated VmInfo vms = 1;       // VM summaries
  string next_page_token = 2;    // Next page token
  int32 total_count = 3;         // Total matching VMs
}
```

#### Example

```bash
# List all running VMs
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_test123" \
  -d '{
    "state_filter": ["VM_STATE_RUNNING"],
    "page_size": 50
  }'
```

### PauseVm

Pauses execution of a running virtual machine.

**RPC**: `PauseVm(PauseVmRequest) returns (PauseVmResponse)`  
**Implementation**: [vm.go:363-391](../../../metald/internal/service/vm.go#L363-L391)

#### Request

```protobuf
message PauseVmRequest {
  string vm_id = 1;  // Required VM identifier
}
```

### ResumeVm

Resumes execution of a paused virtual machine.

**RPC**: `ResumeVm(ResumeVmRequest) returns (ResumeVmResponse)`  
**Implementation**: [vm.go:393-421](../../../metald/internal/service/vm.go#L393-L421)

### RebootVm

Restarts a running virtual machine.

**RPC**: `RebootVm(RebootVmRequest) returns (RebootVmResponse)`  
**Implementation**: [vm.go:618-676](../../../metald/internal/service/vm.go#L618-L676)

#### Request

```protobuf
message RebootVmRequest {
  string vm_id = 1;  // Required VM identifier
  bool force = 2;    // Force immediate reboot
}
```

## Data Types

### VmState Enum

```protobuf
enum VmState {
  VM_STATE_UNSPECIFIED = 0;
  VM_STATE_CREATED = 1;     // VM created but not running
  VM_STATE_RUNNING = 2;     // VM is actively running
  VM_STATE_PAUSED = 3;      // VM execution paused
  VM_STATE_SHUTDOWN = 4;    // VM has been shut down
}
```

### VmConfig Message

Complete VM configuration structure defined in [vm.proto:47-68](../../../metald/proto/vmprovisioner/v1/vm.proto#L47-L68).

Key components:
- `CpuConfig` - Virtual CPU configuration
- `MemoryConfig` - Memory allocation settings
- `BootConfig` - Kernel and boot parameters
- `StorageDevice[]` - Block storage devices
- `NetworkInterface[]` - Network interfaces
- `ConsoleConfig` - Console settings

### NetworkMode Enum

```protobuf
enum NetworkMode {
  NETWORK_MODE_UNSPECIFIED = 0;
  NETWORK_MODE_DUAL_STACK = 1;  // Both IPv4 and IPv6
  NETWORK_MODE_IPV4_ONLY = 2;   // IPv4 only
  NETWORK_MODE_IPV6_ONLY = 3;   // IPv6 only
}
```

## Error Handling

The API uses standard ConnectRPC error codes:

| Code | Name | Usage |
|------|------|-------|
| 3 | InvalidArgument | Invalid request parameters |
| 5 | NotFound | VM or resource not found |
| 7 | PermissionDenied | Authorization failure |
| 8 | ResourceExhausted | Quota exceeded |
| 9 | FailedPrecondition | Invalid state transition |
| 13 | Internal | Unexpected server error |
| 16 | Unauthenticated | Missing authentication |

### Error Response Format

```json
{
  "code": "invalid_argument",
  "message": "vm config is required",
  "details": []
}
```

## Rate Limiting

Currently, no rate limiting is implemented at the API level. Consider implementing rate limits in production deployments.

## API Versioning

The API uses protobuf package versioning (`vmprovisioner.v1`). Breaking changes will result in a new major version (`v2`).