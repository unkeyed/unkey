# API Documentation

This document provides comprehensive reference for the metald VmService API, which offers unified virtual machine management across multiple hypervisor backends with multi-tenant security and comprehensive resource management.

## Service Definition

The VmService provides VM lifecycle management through ConnectRPC/gRPC endpoints defined in [vm.proto](../../proto/vmprovisioner/v1/vm.proto).

### Service Methods

| Method | Purpose | Authentication |
|--------|---------|----------------|
| [`CreateVm`](#createvm) | Create new VM instance | Required |
| [`DeleteVm`](#deletevm) | Remove VM instance | Required |
| [`BootVm`](#bootvm) | Start created VM | Required |
| [`ShutdownVm`](#shutdownvm) | Stop running VM | Required |
| [`PauseVm`](#pausevm) | Pause running VM | Required |
| [`ResumeVm`](#resumevm) | Resume paused VM | Required |
| [`RebootVm`](#rebootvm) | Restart running VM | Required |
| [`GetVmInfo`](#getvminfo) | Get VM status and configuration | Required |
| [`ListVms`](#listvms) | List customer VMs | Required |

## Authentication

All API methods require Bearer token authentication via the `Authorization` header:

```
Authorization: Bearer <customer-token>
```

Customer tokens are validated by [auth.go](../../internal/service/auth.go#L47) and provide multi-tenant isolation. VMs are scoped to the authenticated customer.

## VM States

VMs transition through the following states as defined in [vm.proto](../../proto/vmprovisioner/v1/vm.proto#L38):

```
VM_STATE_CREATED → VM_STATE_RUNNING → VM_STATE_PAUSED
                                   ↓
VM_STATE_SHUTDOWN ← ← ← ← ← ← ← ← ← ←
```

- `VM_STATE_CREATED` - VM created but not started
- `VM_STATE_RUNNING` - VM is actively running 
- `VM_STATE_PAUSED` - VM execution paused
- `VM_STATE_SHUTDOWN` - VM stopped gracefully

## API Methods

### CreateVm

Creates a new virtual machine instance with specified configuration.

**Request**: [`CreateVmRequest`](../../proto/vmprovisioner/v1/vm.proto#L218)
```protobuf
message CreateVmRequest {
  string vm_id = 1;           // Optional: auto-generated if not provided
  VmConfig config = 2;        // Required: VM configuration
  string customer_id = 3;     // Optional: must match authenticated customer
}
```

**Response**: [`CreateVmResponse`](../../proto/vmprovisioner/v1/vm.proto#L229)
```protobuf
message CreateVmResponse {
  string vm_id = 1;          // Assigned VM identifier
  VmState state = 2;         // Current state (VM_STATE_CREATED)
}
```

**Implementation**: [vm.go:49](../../internal/service/vm.go#L49)

**Key Features**:
- Automatic VM ID generation if not provided
- Customer authentication validation via [auth.go:79](../../internal/service/vm.go#L79)
- VM configuration validation via [vm.go:98](../../internal/service/vm.go#L98)
- Database persistence via [vm.go:135](../../internal/service/vm.go#L135)
- Asset preparation via assetmanager integration
- Automatic cleanup on failure with retry logic via [vm.go:143](../../internal/service/vm.go#L143)

**Example**:
```bash
curl -X POST https://metald:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 2},
      "memory": {"size_bytes": 1073741824},
      "boot": {"kernel_path": "/opt/kernels/vmlinux.bin"},
      "storage": [{"path": "/opt/rootfs/ubuntu.ext4", "is_root_device": true}]
    }
  }'
```

**Error Conditions**:
- `INVALID_ARGUMENT` - Missing or invalid VM configuration
- `UNAUTHENTICATED` - Missing or invalid customer authentication  
- `PERMISSION_DENIED` - Customer ID mismatch in request
- `INTERNAL` - Backend or database operation failed

---

### BootVm

Starts a created virtual machine and begins billing metrics collection.

**Request**: [`BootVmRequest`](../../proto/vmprovisioner/v1/vm.proto#L246)
```protobuf
message BootVmRequest {
  string vm_id = 1;          // Required: VM to boot
}
```

**Response**: [`BootVmResponse`](../../proto/vmprovisioner/v1/vm.proto#L248)
```protobuf
message BootVmResponse {
  bool success = 1;          // Operation success
  VmState state = 2;         // Current state (VM_STATE_RUNNING)
}
```

**Implementation**: [vm.go:274](../../internal/service/vm.go#L274)

**Key Features**:
- Customer ownership validation via [vm.go:305](../../internal/service/vm.go#L305)
- Backend VM boot operation via [vm.go:313](../../internal/service/vm.go#L313)
- Database state update via [vm.go:332](../../internal/service/vm.go#L332)
- Billing metrics collection start via [vm.go:349](../../internal/service/vm.go#L349)
- OpenTelemetry tracing with performance metrics

**Example**:
```bash
curl -X POST https://metald:8080/vmprovisioner.v1.VmService/BootVm \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"vm_id": "vm-abc123"}'
```

**Error Conditions**:
- `INVALID_ARGUMENT` - Missing VM ID
- `UNAUTHENTICATED` - Missing customer authentication
- `PERMISSION_DENIED` - VM not owned by authenticated customer
- `INTERNAL` - Backend boot operation failed

---

### ShutdownVm

Gracefully stops a running virtual machine with optional force and timeout.

**Request**: [`ShutdownVmRequest`](../../proto/vmprovisioner/v1/vm.proto#L253)
```protobuf
message ShutdownVmRequest {
  string vm_id = 1;          // Required: VM to shutdown
  bool force = 2;            // Optional: force shutdown vs graceful
  int32 timeout_seconds = 3; // Optional: graceful shutdown timeout
}
```

**Response**: [`ShutdownVmResponse`](../../proto/vmprovisioner/v1/vm.proto#L263)
```protobuf
message ShutdownVmResponse {
  bool success = 1;          // Operation success
  VmState state = 2;         // Current state (VM_STATE_SHUTDOWN)
}
```

**Implementation**: [vm.go:390](../../internal/service/vm.go#L390)

**Key Features**:
- Graceful vs force shutdown options
- Configurable timeout for graceful shutdown
- Billing metrics collection stop via [vm.go:426](../../internal/service/vm.go#L426)
- Database state update with consistency validation
- Customer ownership validation

**Example**:
```bash
# Graceful shutdown with 30 second timeout
curl -X POST https://metald:8080/vmprovisioner.v1.VmService/ShutdownVm \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"vm_id": "vm-abc123", "force": false, "timeout_seconds": 30}'
```

---

### DeleteVm

Removes a virtual machine instance and cleans up all associated resources.

**Request**: [`DeleteVmRequest`](../../proto/vmprovisioner/v1/vm.proto#L237)
```protobuf
message DeleteVmRequest {
  string vm_id = 1;          // Required: VM to delete
  bool force = 2;            // Optional: force delete even if running
}
```

**Response**: [`DeleteVmResponse`](../../proto/vmprovisioner/v1/vm.proto#L244)
```protobuf
message DeleteVmResponse {
  bool success = 1;          // Operation success
}
```

**Implementation**: [vm.go:184](../../internal/service/vm.go#L184)

**Key Features**:
- Customer ownership validation via [vm.go:206](../../internal/service/vm.go#L206)
- Billing metrics collection stop via [vm.go:215](../../internal/service/vm.go#L215)
- Backend resource cleanup via [vm.go:223](../../internal/service/vm.go#L223)
- Database soft delete via [vm.go:237](../../internal/service/vm.go#L237)
- Asset lease release (automatic)

**Example**:
```bash
curl -X POST https://metald:8080/vmprovisioner/v1.VmService/DeleteVm \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"vm_id": "vm-abc123", "force": true}'
```

---

### GetVmInfo

Retrieves comprehensive information about a virtual machine including configuration, state, and metrics.

**Request**: [`GetVmInfoRequest`](../../proto/vmprovisioner/v1/vm.proto#L294)
```protobuf
message GetVmInfoRequest {
  string vm_id = 1;          // Required: VM to query
}
```

**Response**: [`GetVmInfoResponse`](../../proto/vmprovisioner/v1/vm.proto#L296)
```protobuf
message GetVmInfoResponse {
  string vm_id = 1;               // VM identifier
  VmConfig config = 2;            // VM configuration
  VmState state = 3;              // Current state
  VmMetrics metrics = 4;          // Performance metrics
  map<string, string> backend_info = 5;  // Backend-specific info
  VmNetworkInfo network_info = 6; // Network configuration
}
```

**Implementation**: [vm.go:591](../../internal/service/vm.go#L591)

**Key Features**:
- Customer ownership validation
- Real-time state and metrics from backend
- Network information including IP addresses and port mappings
- Backend-specific information for debugging

**Example**:
```bash
curl -X POST https://metald:8080/vmprovisioner.v1.VmService/GetVmInfo \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"vm_id": "vm-abc123"}'
```

---

### ListVms

Lists all virtual machines owned by the authenticated customer.

**Request**: [`ListVmsRequest`](../../proto/vmprovisioner/v1/vm.proto#L358)
```protobuf
message ListVmsRequest {
  repeated VmState state_filter = 1; // Optional: filter by states
  int32 page_size = 2;               // Optional: pagination size
  string page_token = 3;             // Optional: pagination token
}
```

**Response**: [`ListVmsResponse`](../../proto/vmprovisioner/v1/vm.proto#L367)
```protobuf
message ListVmsResponse {
  repeated VmInfo vms = 1;     // VM information list
  string next_page_token = 2;  // Pagination token
  int32 total_count = 3;       // Total VM count
}
```

**Implementation**: [vm.go:637](../../internal/service/vm.go#L637)

**Key Features**:
- Automatic customer filtering via [vm.go:648](../../internal/service/vm.go#L648)
- State-based filtering support
- Pagination support (token-based)
- Database-backed queries via [vm.go:655](../../internal/service/vm.go#L655)

**Example**:
```bash
# List all VMs
curl -X POST https://metald:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{}'

# List only running VMs
curl -X POST https://metald:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"state_filter": ["VM_STATE_RUNNING"]}'
```

## VM Configuration

### VmConfig Structure

Defined in [vm.proto:47](../../proto/vmprovisioner/v1/vm.proto#L47):

```protobuf
message VmConfig {
  CpuConfig cpu = 1;                    // CPU configuration
  MemoryConfig memory = 2;              // Memory configuration  
  BootConfig boot = 3;                  // Boot configuration
  repeated StorageDevice storage = 4;   // Storage devices
  repeated NetworkInterface network = 5; // Network interfaces
  ConsoleConfig console = 6;            // Console configuration
  map<string, string> metadata = 7;    // Custom metadata
}
```

### CPU Configuration

```protobuf
message CpuConfig {
  int32 vcpu_count = 1;           // Number of vCPUs (required)
  int32 max_vcpu_count = 2;       // Max vCPUs for hotplug
  CpuTopology topology = 3;       // CPU topology
  map<string, string> features = 4; // CPU features
}
```

### Memory Configuration

```protobuf
message MemoryConfig {
  int64 size_bytes = 1;           // Memory size in bytes (required)
  bool hotplug_enabled = 2;       // Enable memory hotplug
  int64 max_size_bytes = 3;       // Max memory for hotplug
  map<string, string> backing = 4; // Memory backing options
}
```

### Storage Configuration

```protobuf
message StorageDevice {
  string id = 1;                  // Device identifier
  string path = 2;                // Path to backing file (required)
  bool read_only = 3;             // Read-only flag
  bool is_root_device = 4;        // Root device flag
  string interface_type = 5;      // Interface type (virtio-blk, nvme)
  map<string, string> options = 6; // Device options
}
```

### Network Configuration  

```protobuf
message NetworkInterface {
  string id = 1;                  // Interface identifier
  string mac_address = 2;         // MAC address (auto-generated if empty)
  string tap_device = 3;          // TAP device name
  string interface_type = 4;      // Interface type (virtio-net, e1000)
  map<string, string> options = 5; // Interface options
  IPv4Config ipv4_config = 6;     // IPv4 configuration
  IPv6Config ipv6_config = 7;     // IPv6 configuration
  NetworkMode mode = 8;           // Network mode (dual-stack, IPv4-only, IPv6-only)
  RateLimit rx_rate_limit = 10;   // Receive rate limit
  RateLimit tx_rate_limit = 11;   // Transmit rate limit
}
```

## Error Handling

### Standard Error Codes

| Code | Description | Retry Recommended |
|------|-------------|-------------------|
| `INVALID_ARGUMENT` | Invalid request parameters | No |
| `UNAUTHENTICATED` | Missing or invalid authentication | No |
| `PERMISSION_DENIED` | Customer authorization failed | No |
| `NOT_FOUND` | VM not found | No |
| `RESOURCE_EXHAUSTED` | System resource limits exceeded | Yes (with backoff) |
| `INTERNAL` | Internal server error | Yes (with backoff) |
| `UNAVAILABLE` | Service temporarily unavailable | Yes (with backoff) |

### Error Response Format

ConnectRPC errors include structured information:

```json
{
  "code": "INVALID_ARGUMENT",
  "message": "vm config is required",
  "details": []
}
```

## Downstream Service Calls

Metald integrates with other Unkey Deploy services:

### AssetManager Integration

- **Query Assets**: Automatic asset lookup via [assetmanager/client.go:160](../../internal/assetmanager/client.go#L160)
- **Prepare Assets**: Asset staging for VM creation via [assetmanager/client.go:220](../../internal/assetmanager/client.go#L220)
- **Acquire Assets**: Lease management via [assetmanager/client.go:262](../../internal/assetmanager/client.go#L262)
- **Release Assets**: Cleanup on VM deletion

### Billing Integration

- **SendMetricsBatch**: VM usage metrics via [billing/client.go:167](../../internal/billing/client.go#L167)
- **SendHeartbeat**: Service health monitoring via [billing/client.go:228](../../internal/billing/client.go#L228)
- **NotifyVmStarted**: VM lifecycle events via [billing/client.go:265](../../internal/billing/client.go#L265)
- **NotifyVmStopped**: VM lifecycle events via [billing/client.go:305](../../internal/billing/client.go#L305)

## Client Libraries

### Go Client

```go
import "github.com/unkeyed/unkey/go/deploy/metald/client"

// Create client with SPIFFE authentication
client, err := client.New(ctx, client.Config{
    ServerAddress: "https://metald:8080",
    CustomerID:    "customer-123",
    TLSMode:      "spiffe",
})

// Create VM
vm, err := client.CreateVM(ctx, &client.CreateVMRequest{
    Config: &client.VMConfig{
        CPU:    &client.CPUConfig{VCPUCount: 2},
        Memory: &client.MemoryConfig{SizeBytes: 1 << 30}, // 1GB
        Boot:   &client.BootConfig{KernelPath: "/opt/kernels/vmlinux"},
        Storage: []*client.StorageDevice{{
            Path:         "/opt/rootfs/ubuntu.ext4",
            IsRootDevice: true,
        }},
    },
})
```

See [client documentation](../../client/README.md) for complete reference.

### CLI Tool

```bash
# Install CLI
cd client/cmd/metald-cli
go install

# Create VM with configuration file
metald-cli create-vm --config vm-config.json --server https://metald:8080

# List VMs
metald-cli list-vms --server https://metald:8080

# Get VM info
metald-cli get-vm --vm-id vm-123 --server https://metald:8080
```

## Rate Limiting

API requests are subject to rate limiting based on customer authentication:

- **Authenticated requests**: 1000 requests/minute per customer
- **VM operations**: 100 concurrent operations per customer
- **List operations**: 10 requests/minute per customer

Rate limits are enforced by the tenant authentication interceptor.

## Monitoring

Key API metrics available via OpenTelemetry:

- `metald_vm_operations_total{method, result, customer_id}` - Operation counts
- `metald_vm_operation_duration_seconds{method}` - Operation latency histograms
- `metald_api_requests_total{method, code}` - Request counts by status code
- `metald_api_request_duration_seconds{method}` - Request duration histograms

Metrics configuration: [observability/metrics.go](../../internal/observability/metrics.go)