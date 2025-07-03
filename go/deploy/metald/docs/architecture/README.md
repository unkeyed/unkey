# Architecture Guide

This document provides a comprehensive overview of metald's architecture, covering service design, component interactions, and integration patterns with other Unkey Deploy services.

## Service Overview

Metald is the Virtual Machine Manager (VMM) control plane that provides unified VM lifecycle management across multiple hypervisor backends. It serves as the central orchestrator for VM operations in the Unkey Deploy infrastructure.

**Purpose**: Unified VM management with multi-tenant security, resource tracking, and comprehensive observability.

## Core Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        metald Service                           │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐│
│  │  API Gateway    │    │ Authentication  │    │  Authorization  ││
│  │  (ConnectRPC)   │    │  (SPIFFE/mTLS)  │    │  (Customer)     ││
│  └─────────────────┘    └─────────────────┘    └─────────────────┘│
│           │                       │                       │       │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐│
│  │   VM Service    │    │ Billing Client  │    │ Asset Client    ││
│  │ [vm.go:25]      │    │ [billing/*.go]  │    │ [assetmgr/*.go] ││
│  └─────────────────┘    └─────────────────┘    └─────────────────┘│
│           │                       │                       │       │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐│
│  │ VM Repository   │    │ Network Manager │    │ State Monitor   ││
│  │ [database/*.go] │    │ [network/*.go]  │    │ [reconciler/]   ││
│  └─────────────────┘    └─────────────────┘    └─────────────────┘│
│           │                       │                       │       │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐│
│  │ Backend Layer   │    │ Observability   │    │ Configuration   ││
│  │ [backend/*.go]  │    │ [observ/*.go]   │    │ [config/*.go]   ││
│  └─────────────────┘    └─────────────────┘    └─────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

1. **API Layer**: ConnectRPC endpoints with authentication and authorization
2. **Service Layer**: Business logic for VM lifecycle management
3. **Integration Layer**: External service communication (billing, assets)
4. **Persistence Layer**: VM state management and database operations
5. **Backend Layer**: Hypervisor abstraction and VM execution
6. **Infrastructure Layer**: Networking, observability, and configuration

## Service Dependencies

### Primary Dependencies

#### AssetManagerd Integration
- **Source**: [assetmanager/client.go](../../internal/assetmanager/client.go)
- **Purpose**: VM asset management (kernels, rootfs images)
- **Key Operations**:
  - `QueryAssets` - Asset discovery with automatic building via [client.go:160](../../internal/assetmanager/client.go#L160)
  - `PrepareAssets` - Asset staging for VM creation via [client.go:220](../../internal/assetmanager/client.go#L220)
  - `AcquireAsset` - Reference counting via [client.go:262](../../internal/assetmanager/client.go#L262)
  - `ReleaseAsset` - Cleanup on VM deletion via [client.go:303](../../internal/assetmanager/client.go#L303)

#### Billaged Integration
- **Source**: [billing/client.go](../../internal/billing/client.go)
- **Purpose**: VM usage metrics and billing aggregation
- **Key Operations**:
  - `SendMetricsBatch` - Real-time VM metrics via [client.go:167](../../internal/billing/client.go#L167)
  - `SendHeartbeat` - Service health monitoring via [client.go:228](../../internal/billing/client.go#L228)
  - `NotifyVmStarted` - Lifecycle events via [client.go:265](../../internal/billing/client.go#L265)
  - `NotifyVmStopped` - Lifecycle events via [client.go:305](../../internal/billing/client.go#L305)

#### SPIFFE/Spire Integration
- **Source**: [TLS configuration](../../internal/config/config.go#L372)
- **Purpose**: Service-to-service mTLS authentication
- **Features**:
  - Automatic certificate rotation
  - Service identity verification
  - Secure inter-service communication

### Optional Dependencies

#### Builderd Integration
- **Integration**: Via assetmanagerd automatic build triggering
- **Purpose**: Automatic rootfs creation when assets don't exist
- **Flow**: metald → assetmanagerd → builderd (when assets missing)

## VM Lifecycle Management

### State Machine

VMs transition through well-defined states with database persistence:

```
┌─────────────┐    CreateVm    ┌─────────────┐    BootVm     ┌─────────────┐
│             │ ──────────────▶│             │ ─────────────▶│             │
│ NOT_CREATED │                │   CREATED   │               │   RUNNING   │
│             │                │             │               │             │
└─────────────┘                └─────────────┘               └─────────────┘
                                       │                             │
                                DeleteVm│                             │PauseVm
                                       │                             ▼
                               ┌─────────────┐                ┌─────────────┐
                               │             │                │             │
                               │   DELETED   │                │   PAUSED    │
                               │             │                │             │
                               └─────────────┘                └─────────────┘
                                       ▲                             │
                                       │                             │ResumeVm
                ┌─────────────┐        │ShutdownVm            ┌─────────────┐
                │             │ ───────────────────────────────│             │
                │  SHUTDOWN   │◀───────────────────────────────│   RUNNING   │
                │             │                               │             │
                └─────────────┘                               └─────────────┘
```

**Implementation**: [service/vm.go](../../internal/service/vm.go)

### Database State Consistency

VM state is persisted in SQLite with automatic reconciliation:

- **Repository**: [database/repository.go](../../internal/database/repository.go)
- **Schema**: [database/schema.sql](../../internal/database/schema.sql)
- **Reconciler**: [reconciler/vm_reconciler.go](../../internal/reconciler/vm_reconciler.go)

**Key Features**:
- Transaction-based state updates
- Automatic orphaned VM cleanup via [reconciler/vm_reconciler.go](../../internal/reconciler/vm_reconciler.go)
- Customer-scoped queries for multi-tenancy
- Soft deletes for audit trails

## Backend Abstraction

### Backend Interface

Defined in [backend/types/backend.go](../../internal/backend/types/backend.go):

```go
type Backend interface {
    CreateVM(ctx context.Context, config *VmConfig) (string, error)
    DeleteVM(ctx context.Context, vmID string) error
    BootVM(ctx context.Context, vmID string) error
    ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeout int32) error
    PauseVM(ctx context.Context, vmID string) error
    ResumeVM(ctx context.Context, vmID string) error
    RebootVM(ctx context.Context, vmID string) error
    GetVMInfo(ctx context.Context, vmID string) (*VMInfo, error)
}
```

### Firecracker Backend

Primary implementation using Firecracker SDK:

- **Source**: [backend/firecracker/sdk_client_v4.go](../../internal/backend/firecracker/sdk_client_v4.go)
- **Features**:
  - Integrated jailer for security isolation via [jailer/jailer.go](../../internal/jailer/jailer.go)
  - Network management with TAP devices
  - Asset preparation and mounting
  - Process lifecycle management
  - Metrics collection via stats sockets

**Security Features**:
- Chroot isolation via [config.go:67](../../internal/config/config.go#L67)
- Dedicated UID/GID per VM
- Network namespace isolation
- Resource limits and quotas

### Cloud Hypervisor Backend

**Status**: Planned but not implemented
- **Interface**: Ready via backend abstraction
- **Implementation**: [backend/cloudhypervisor/client.go](../../internal/backend/cloudhypervisor/client.go) (placeholder)

## Network Management

### Network Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Host Network                               │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │     VM1     │    │     VM2     │    │     VM3     │         │
│  │ 172.31.0.10 │    │ 172.31.0.11 │    │ 172.31.0.12 │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│         │                   │                   │              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │    tap1     │    │    tap2     │    │    tap3     │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│         │                   │                   │              │
│         └───────────────────┼───────────────────┘              │
│                             │                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   br-vms                                │   │
│  │               172.31.0.1/19                             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                             │                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                Host Interface                           │   │
│  │                 (NAT/Forwarding)                        │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Network Components

#### Network Manager
- **Source**: [network/implementation.go](../../internal/network/implementation.go)
- **Features**:
  - Bridge creation and management
  - IP address allocation via [network/allocator.go](../../internal/network/allocator.go)
  - TAP device creation and configuration
  - IPv4/IPv6 dual-stack support
  - Rate limiting and QoS

#### Port Allocation
- **Source**: [network/port_allocator.go](../../internal/network/port_allocator.go)
- **Features**:
  - Dynamic port assignment for VM services
  - Port range management (32768-65535)
  - Conflict detection and resolution
  - Automatic cleanup on VM deletion

#### Network Protection
- **Source**: [network/protection.go](../../internal/network/protection.go)
- **Features**:
  - Host route protection from VM access
  - Primary interface detection and isolation
  - Firewall rule management
  - Network namespace isolation

## Multi-Tenant Security

### Authentication Layer

**SPIFFE/SPIRE Integration**:
- Service identity verification via [config.go:372](../../internal/config/config.go#L372)
- Automatic certificate rotation
- mTLS for all service communications

**Customer Authentication**:
- Bearer token validation via [service/auth.go:22](../../internal/service/auth.go#L22)
- Customer context extraction via [auth.go:47](../../internal/service/auth.go#L47)
- Request-response baggage propagation

### Authorization Model

**Customer Isolation**:
- Database-level customer scoping via [repository.go](../../internal/database/repository.go)
- VM ownership validation via [vm.go:206](../../internal/service/vm.go#L206)
- API request filtering by authenticated customer

**Resource Quotas**:
- Per-customer VM limits (configurable)
- Resource allocation controls (CPU, memory, storage)
- Network bandwidth limiting

## Observability Architecture

### Metrics Collection

**VM Metrics**: [observability/metrics.go](../../internal/observability/metrics.go)
- `metald_vm_operations_total` - Operation counts by type and result
- `metald_vm_operation_duration_seconds` - Operation latency histograms
- `metald_active_vms` - Current VM count by state and customer
- `metald_backend_errors_total` - Backend failure rates

**Billing Metrics**: [observability/billing_metrics.go](../../internal/observability/billing_metrics.go)
- `metald_billing_metrics_collected_total` - Metrics collection rate
- `metald_billing_errors_total` - Billing integration failures
- `metald_vm_resource_usage` - Real-time resource consumption

### Distributed Tracing

**OpenTelemetry Integration**:
- Trace propagation across service boundaries
- Operation-level span creation via [vm.go:50](../../internal/service/vm.go#L50)
- Error attribution and correlation
- Performance monitoring and optimization

**Debug Interceptors**:
- Request/response logging via [observability/debug_interceptor.go](../../internal/observability/debug_interceptor.go)
- Service call tracing
- Error context preservation

## Service Interactions

### Outbound Service Calls

#### AssetManager Communication
```
metald ──QueryAssets──▶ assetmanagerd ──TriggerBuild──▶ builderd
       ◀──Assets────────              ◀──BuildResult───
       ──PrepareAssets─▶
       ◀──PreparedPaths─
       ──AcquireAsset──▶
       ◀──LeaseID──────
```

**Implementation**: [assetmanager/client.go](../../internal/assetmanager/client.go)

#### Billing Communication
```
metald ──SendMetricsBatch──▶ billaged
       ──SendHeartbeat────▶
       ──NotifyVmStarted──▶
       ──NotifyVmStopped──▶
       ──NotifyPossibleGap▶
```

**Implementation**: [billing/client.go](../../internal/billing/client.go)

### Inbound Service Calls

#### Client Applications
- **Source**: External applications via ConnectRPC API
- **Authentication**: Bearer token + SPIFFE validation
- **Operations**: All VM lifecycle operations

#### Monitoring Systems
- **Prometheus**: Metrics scraping via `/metrics` endpoint
- **Health Checks**: Status monitoring via `/health` endpoint
- **OTLP Export**: Trace and metric export to collectors

## Data Flow Patterns

### VM Creation Flow

```
1. Client ──CreateVm──▶ metald
2. metald ──Authenticate──▶ Customer validation
3. metald ──QueryAssets──▶ assetmanagerd
4. assetmanagerd ──TriggerBuild──▶ builderd (if needed)
5. metald ──PrepareAssets──▶ assetmanagerd
6. metald ──CreateVM──▶ Firecracker backend
7. metald ──PersistState──▶ SQLite database
8. metald ──AcquireAssets──▶ assetmanagerd
9. metald ──Response──▶ Client
```

### VM Boot Flow

```
1. Client ──BootVm──▶ metald
2. metald ──ValidateOwnership──▶ Database
3. metald ──BootVM──▶ Firecracker backend
4. metald ──UpdateState──▶ Database
5. metald ──StartCollection──▶ Billing metrics
6. metald ──NotifyVmStarted──▶ billaged
7. metald ──Response──▶ Client
```

### Metrics Collection Flow

```
1. Background ──CollectMetrics──▶ Firecracker stats
2. metald ──BatchMetrics──▶ Billing collector
3. Collector ──SendMetricsBatch──▶ billaged
4. metald ──ExportMetrics──▶ OpenTelemetry
5. OTEL ──Export──▶ Prometheus/Jaeger
```

## Error Handling Patterns

### Retry Logic

**Asset Operations**:
- Exponential backoff for asset preparation failures
- Circuit breaker for assetmanagerd communication
- Fallback to cached assets when available

**Billing Operations**:
- Buffering for temporary billaged unavailability
- Retry with jitter for metric batching
- Gap detection and reconciliation

### Resource Cleanup

**VM Cleanup**: [vm.go:811](../../internal/service/vm.go#L811)
- Multi-retry cleanup with grace periods
- Orphaned resource detection via reconciler
- Manual cleanup logging for operator intervention

**Asset Cleanup**:
- Automatic lease release on VM deletion
- Reference counting for garbage collection
- Storage cleanup via assetmanagerd

### State Consistency

**Database Reconciliation**: [reconciler/vm_reconciler.go](../../internal/reconciler/vm_reconciler.go)
- Periodic state validation (5 minute interval)
- Orphaned VM detection and cleanup
- Process state synchronization

**Network Cleanup**:
- TAP device cleanup on VM deletion
- IP address deallocation
- Bridge cleanup when no VMs remain

## Configuration Management

### Environment Variables

**Critical Settings**: [config/config.go](../../internal/config/config.go)
- `UNKEY_METALD_BACKEND` - Hypervisor backend selection
- `UNKEY_METALD_TLS_MODE` - Security mode (spiffe/file/disabled)
- `UNKEY_METALD_BILLING_ENABLED` - Billing integration toggle
- `UNKEY_METALD_ASSETMANAGER_ENABLED` - Asset integration toggle

**Security Settings**:
- `UNKEY_METALD_JAILER_UID` - VM isolation user ID
- `UNKEY_METALD_JAILER_GID` - VM isolation group ID  
- `UNKEY_METALD_JAILER_CHROOT_DIR` - Isolation directory

**Network Settings**:
- `UNKEY_METALD_NETWORK_BRIDGE_IPV4` - Bridge IP configuration
- `UNKEY_METALD_NETWORK_VM_SUBNET_IPV4` - VM subnet allocation
- `UNKEY_METALD_NETWORK_HOST_PROTECTION` - Host route protection

### Validation

**Configuration Validation**: [config.go:393](../../internal/config/config.go#L393)
- Required field validation
- Value range checking
- Cross-field consistency validation
- Environment-specific defaults

## Performance Considerations

### Concurrency

**VM Operations**:
- Per-customer operation limiting
- Backend-level concurrency control
- Database connection pooling

**Network Operations**:
- Parallel TAP device creation
- Concurrent IP allocation
- Asynchronous cleanup operations

### Resource Management

**Memory Usage**:
- VM state caching with TTL
- Metrics buffering for batch operations
- Database connection reuse

**Storage Optimization**:
- Asset deduplication via assetmanagerd
- Compressed VM state storage
- Efficient database indexing

### Scalability Patterns

**Horizontal Scaling**:
- Stateless service design (except local DB)
- Customer-based load distribution
- Service discovery via SPIFFE

**Vertical Scaling**:
- Configurable worker pools
- Memory-based caching strategies
- Database optimization for VM queries

## Security Architecture

### Process Isolation

**Jailer Integration**: [jailer/jailer.go](../../internal/jailer/jailer.go)
- Chroot environment per VM
- Dedicated UID/GID isolation
- Resource limit enforcement
- Namespace isolation

### Network Security

**Isolation Mechanisms**:
- Network namespace per VM
- Bridge-based segmentation
- Host route protection
- Firewall rule automation

### Data Protection

**Customer Data Isolation**:
- Database-level customer scoping
- Encrypted data at rest (TLS for transport)
- Audit logging for all operations
- SPIFFE identity verification

## Deployment Architecture

### Service Placement

**Requirements**:
- Root privileges for network operations
- Direct access to hypervisor binaries
- Local SQLite database storage
- Network interface management capabilities

**Recommended Setup**:
- Dedicated VM management nodes
- systemd service management
- Log aggregation for audit trails
- Monitoring integration

### High Availability

**State Management**:
- Local database for fast access
- External backup for disaster recovery
- State reconciliation on restart
- Customer data partitioning

**Service Resilience**:
- Health check endpoints
- Graceful degradation modes
- Circuit breaker patterns
- Automatic restart capabilities