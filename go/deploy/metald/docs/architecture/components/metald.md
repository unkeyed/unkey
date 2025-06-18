# Metald Core Architecture

> VM lifecycle management service with multi-VMM support and real-time billing integration

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Component Details](#component-details)
- [VM Lifecycle Management](#vm-lifecycle-management)
- [Multi-Tenant Security](#multi-tenant-security)
- [Backend Abstraction](#backend-abstraction)
- [Process Management](#process-management)
- [Billing Integration](#billing-integration)
- [Operational Considerations](#operational-considerations)
- [Cross-References](#cross-references)

---

## Overview

Metald is the core VM management service that orchestrates virtual machine lifecycles across multiple hypervisor backends while providing real-time billing metrics and multi-tenant security isolation.

### Key Responsibilities
- **VM Lifecycle Management**: Create, boot, shutdown, delete, pause, resume, reboot
- **Multi-Tenant Security**: Customer authentication, authorization, and resource isolation
- **Backend Abstraction**: Unified API across Firecracker and Cloud Hypervisor
- **Process Management**: 1:1 VM-to-process isolation model for security
- **Billing Integration**: Real-time metrics streaming to billing service
- **Observability**: OpenTelemetry tracing and Prometheus metrics

### Technology Stack
- **Language**: Go 1.21+
- **API Protocol**: ConnectRPC (gRPC-compatible)
- **Observability**: OpenTelemetry, Prometheus metrics
- **Database**: SQLite with WAL mode
- **Authentication**: JWT Bearer tokens with customer context

---

## Architecture

### High-Level Component Architecture

```mermaid
graph TB
    subgraph "API Layer"
        ConnectRPC[ConnectRPC Server<br/>- VM Service API<br/>- Health Endpoints<br/>- TLS Termination]
    end
    
    subgraph "Middleware Layer"
        Auth[Authentication Interceptor<br/>- JWT Validation<br/>- Customer Context<br/>- Development Tokens]
        Observability[Observability Interceptor<br/>- Distributed Tracing<br/>- Request Metrics<br/>- Error Tracking]
    end
    
    subgraph "Service Layer"
        VMService[VM Service<br/>- Lifecycle Operations<br/>- Ownership Validation<br/>- State Management]
        AuthService[Auth Service<br/>- Token Validation<br/>- Customer Extraction<br/>- Context Propagation]
    end
    
    subgraph "Backend Layer"
        BackendManager[Backend Manager<br/>- Backend Selection<br/>- Interface Abstraction<br/>- Configuration]
        
        subgraph "VM Backends"
            Firecracker[Firecracker Backend<br/>- Production Ready<br/>- Integrated Jailer<br/>- FIFO Streaming]
            CloudHypervisor[Cloud Hypervisor Backend<br/>- Development/Testing<br/>- API Client<br/>- Alternative Option]
        end
    end
    
    subgraph "Data Layer"
        Repository[VM Repository<br/>- SQLite Database<br/>- Customer Filtering<br/>- Concurrent Access]
        ProcessManager[Process Manager<br/>- Process Lifecycle<br/>- 1:1 VM Mapping<br/>- Cleanup Handling]
    end
    
    subgraph "Integration Layer"
        BillingClient[Billing Client<br/>- Metrics Streaming<br/>- Heartbeat Monitoring<br/>- Failure Recovery]
        HealthChecker[Health Checker<br/>- VM Status Monitoring<br/>- Process Validation<br/>- Recovery Actions]
    end
    
    ConnectRPC --> Auth
    Auth --> Observability
    Observability --> VMService
    VMService --> AuthService
    
    VMService --> BackendManager
    VMService --> Repository
    VMService --> ProcessManager
    VMService --> BillingClient
    VMService --> HealthChecker
    
    BackendManager --> Firecracker
    BackendManager --> CloudHypervisor
    
    ProcessManager --> Firecracker
    ProcessManager --> CloudHypervisor
    
    BillingClient --> BillingService[Billing Service]
```

### Request Processing Architecture

```mermaid
sequenceDiagram
    participant Client
    participant Auth as Auth Interceptor
    participant VMSvc as VM Service
    participant Backend
    participant Process as Process Manager
    participant Billing
    participant DB as Repository
    
    Client->>Auth: CreateVM Request
    Auth->>Auth: Validate JWT Token
    Auth->>Auth: Extract Customer ID
    Auth->>VMSvc: Request + Customer Context
    
    VMSvc->>VMSvc: Validate Configuration
    VMSvc->>DB: Check Customer Ownership
    VMSvc->>Backend: Create VM
    Backend->>Process: Start VM Process
    Process->>Process: Initialize FIFO Stream
    Process->>Backend: VM Process Ready
    Backend->>VMSvc: VM Created
    
    VMSvc->>DB: Store VM Metadata
    VMSvc->>Billing: Start Metrics Collection
    Billing->>Process: Begin FIFO Streaming
    
    VMSvc->>Client: Success Response
    
    loop Every 100ms
        Process->>Billing: Stream VM Metrics
    end
```

---

## Component Details

### Authentication System

```mermaid
graph LR
    subgraph "Token Types"
        DevToken[Development Token<br/>dev_customer_123]
        ProdToken[Production JWT<br/>Signed + Customer Claims]
    end
    
    subgraph "Authentication Flow"
        Extract[Token Extraction<br/>Bearer Header]
        Validate[Token Validation<br/>Format + Signature]
        Context[Customer Context<br/>OpenTelemetry Baggage]
    end
    
    subgraph "Authorization"
        Ownership[Ownership Validation<br/>Customer VM Access]
        Resources[Resource Limits<br/>Per-Customer Quotas]
    end
    
    DevToken --> Extract
    ProdToken --> Extract
    Extract --> Validate
    Validate --> Context
    Context --> Ownership
    Context --> Resources
```

**Authentication Features**:
- **Development Tokens**: `dev_customer_<id>` format for testing
- **Production JWT**: Signed tokens with customer claims
- **Customer Context**: Propagated via OpenTelemetry baggage
- **Ownership Validation**: All VM operations check customer ownership

### Backend Abstraction Layer

```mermaid
graph TB
    subgraph "Backend Interface"
        Interface[Backend Interface<br/>- Unified API<br/>- Error Handling<br/>- Configuration]
    end
    
    subgraph "Firecracker Implementation"
        FCClient[Firecracker Client<br/>- Process Management<br/>- API Communication<br/>- Integrated Jailer]
        FCProcess[Process Wrapper<br/>- FIFO Streaming<br/>- Lifecycle Management<br/>- Security Isolation]
    end
    
    subgraph "Cloud Hypervisor Implementation"
        CHClient[Cloud Hypervisor Client<br/>- REST API Client<br/>- VM Management<br/>- Development Focus]
        CHProcess[Process Manager<br/>- Alternative Backend<br/>- Testing Support]
    end
    
    Interface --> FCClient
    Interface --> CHClient
    
    FCClient --> FCProcess
    CHClient --> CHProcess
```

**Backend Features**:
- **Unified Interface**: Same API regardless of backend
- **Configuration-Driven**: Backend selection via environment variables
- **Error Mapping**: Backend-specific errors mapped to common interface
- **Resource Management**: Backend-specific resource allocation

### Process Management Architecture

```mermaid
graph TB
    subgraph "Process Model"
        OneToOne[1:1 VM-Process Model<br/>- Security Isolation<br/>- Resource Containment<br/>- Failure Isolation]
    end
    
    subgraph "Process Lifecycle"
        Create[Process Creation<br/>- Fork/Exec Model<br/>- Environment Setup<br/>- Security Context]
        Monitor[Process Monitoring<br/>- Health Checks<br/>- Resource Usage<br/>- State Tracking]
        Cleanup[Process Cleanup<br/>- Graceful Shutdown<br/>- Resource Reclamation<br/>- Error Recovery]
    end
    
    subgraph "Security Features"
        Jailer[Integrated Jailer<br/>- chroot Isolation<br/>- Network Namespace<br/>- TAP Device Control]
        Privileges[Privilege Dropping<br/>- Non-root Execution<br/>- Capability Limits<br/>- User Namespaces]
    end
    
    OneToOne --> Create
    Create --> Monitor
    Monitor --> Cleanup
    
    Create --> Jailer
    Jailer --> Privileges
```

**Process Management Features**:
- **1:1 Isolation**: Each VM runs in separate process
- **Security Isolation**: Integrated jailer for production security
- **Resource Limits**: cgroups v2 for memory/CPU constraints
- **Failure Isolation**: Process failures don't affect other VMs

---

## VM Lifecycle Management

### State Machine

```mermaid
stateDiagram-v2
    [*] --> Creating : CreateVM
    Creating --> Created : Success
    Creating --> Failed : Error
    
    Created --> Booting : BootVM
    Booting --> Running : Success
    Booting --> Failed : Error
    
    Running --> Pausing : PauseVM
    Pausing --> Paused : Success
    Pausing --> Failed : Error
    
    Paused --> Resuming : ResumeVM
    Resuming --> Running : Success
    Resuming --> Failed : Error
    
    Running --> Rebooting : RebootVM
    Rebooting --> Running : Success
    Rebooting --> Failed : Error
    
    Running --> Shutting_Down : ShutdownVM
    Paused --> Shutting_Down : ShutdownVM
    Shutting_Down --> Stopped : Success
    Shutting_Down --> Failed : Error
    
    Created --> Deleting : DeleteVM
    Stopped --> Deleting : DeleteVM
    Failed --> Deleting : DeleteVM
    Deleting --> [*] : Success
```

### Lifecycle Operations

```mermaid
sequenceDiagram
    participant Client
    participant Metald
    participant Backend
    participant Process
    participant VM
    
    Note over Client,VM: VM Creation Flow
    Client->>Metald: CreateVM(config)
    Metald->>Backend: Create VM
    Backend->>Process: Start Process
    Process->>VM: Initialize VM
    VM->>Process: Ready
    Process->>Backend: VM Created
    Backend->>Metald: Success
    Metald->>Client: VM ID
    
    Note over Client,VM: VM Boot Flow
    Client->>Metald: BootVM(vm_id)
    Metald->>Backend: Boot VM
    Backend->>VM: Start Boot
    VM->>Backend: Boot Complete
    Backend->>Metald: Running
    Metald->>Client: Success
    
    Note over Client,VM: VM Shutdown Flow
    Client->>Metald: ShutdownVM(vm_id)
    Metald->>Backend: Shutdown VM
    Backend->>VM: Graceful Shutdown
    VM->>Backend: Stopped
    Backend->>Metald: Stopped
    Metald->>Client: Success
    
    Note over Client,VM: VM Deletion Flow
    Client->>Metald: DeleteVM(vm_id)
    Metald->>Backend: Delete VM
    Backend->>Process: Terminate Process
    Process->>VM: Cleanup
    VM->>Process: Destroyed
    Process->>Backend: Cleaned Up
    Backend->>Metald: Deleted
    Metald->>Client: Success
```

---

## Multi-Tenant Security

### Customer Isolation Model

```mermaid
graph TB
    subgraph "Request Context"
        CustomerToken[Customer Token<br/>Bearer JWT]
        CustomerID[Customer ID<br/>Extracted from Token]
        Context[Request Context<br/>OpenTelemetry Baggage]
    end
    
    subgraph "Authorization Layer"
        Ownership[Ownership Validation<br/>VM belongs to Customer]
        ResourceLimits[Resource Limits<br/>Per-Customer Quotas]
        DataIsolation[Data Isolation<br/>Customer-filtered Queries]
    end
    
    subgraph "Runtime Isolation"
        ProcessIsolation[Process Isolation<br/>1:1 VM-Process Model]
        NetworkIsolation[Network Isolation<br/>Customer VLANs/Namespaces]
        StorageIsolation[Storage Isolation<br/>Customer-specific Paths]
    end
    
    CustomerToken --> CustomerID
    CustomerID --> Context
    Context --> Ownership
    Context --> ResourceLimits
    Context --> DataIsolation
    
    DataIsolation --> ProcessIsolation
    ProcessIsolation --> NetworkIsolation
    NetworkIsolation --> StorageIsolation
```

### Security Boundaries

```mermaid
graph LR
    subgraph "Authentication Boundary"
        JWT[JWT Validation<br/>- Token Signature<br/>- Claims Validation<br/>- Expiry Check]
    end
    
    subgraph "Authorization Boundary"
        RBAC[Resource Access<br/>- Customer Ownership<br/>- Operation Permissions<br/>- Resource Quotas]
    end
    
    subgraph "Runtime Boundary"
        Process[Process Isolation<br/>- Separate Processes<br/>- User Namespaces<br/>- Capability Limits]
        VM[VM Isolation<br/>- Memory Isolation<br/>- Network Namespaces<br/>- Storage Isolation]
    end
    
    JWT --> RBAC
    RBAC --> Process
    Process --> VM
```

---

## Backend Abstraction

### Interface Design

```go
type Backend interface {
    // VM Lifecycle
    CreateVM(ctx context.Context, config *VMConfig) (*VM, error)
    BootVM(ctx context.Context, vmID string) error
    ShutdownVM(ctx context.Context, vmID string) error
    DeleteVM(ctx context.Context, vmID string) error
    
    // VM State Management
    PauseVM(ctx context.Context, vmID string) error
    ResumeVM(ctx context.Context, vmID string) error
    RebootVM(ctx context.Context, vmID string) error
    
    // VM Information
    GetVMInfo(ctx context.Context, vmID string) (*VMInfo, error)
    ListVMs(ctx context.Context) ([]*VM, error)
    
    // Health & Monitoring
    HealthCheck(ctx context.Context) error
    GetMetrics(ctx context.Context, vmID string) (*Metrics, error)
}
```

### Backend Comparison

| Feature | Firecracker | Cloud Hypervisor |
|---------|-------------|------------------|
| **Maturity** | Production Ready | Development/Testing |
| **Security** | Integrated Jailer | Basic Isolation |
| **Performance** | Optimized | Good |
| **Billing** | FIFO Streaming | Polling-based |
| **Networking** | IPv6 Support | IPv6 Support |
| **Process Model** | 1:1 VM-Process | 1:1 VM-Process |
| **Resource Limits** | cgroups v2 | cgroups v2 |
| **Use Case** | Production | Development |

---

## Process Management

### Process Architecture

```mermaid
graph TB
    subgraph "Process Manager"
        Manager[Process Manager<br/>- Lifecycle Control<br/>- Resource Monitoring<br/>- Cleanup Handling]
        Registry[Process Registry<br/>- Active Processes<br/>- State Tracking<br/>- Resource Usage]
    end
    
    subgraph "Process Instances"
        P1[VM Process 1<br/>- Customer A<br/>- VM-001<br/>- Firecracker]
        P2[VM Process 2<br/>- Customer B<br/>- VM-002<br/>- Cloud Hypervisor]
        Pn[VM Process N<br/>- Customer N<br/>- VM-nnn<br/>- Backend Type]
    end
    
    subgraph "Security Context"
        Jailer[Integrated Jailer<br/>- chroot Isolation<br/>- Network Control<br/>- Privilege Dropping]
        User[User Context<br/>- Non-root User<br/>- Dropped Privileges<br/>- Limited Capabilities]
    end
    
    Manager --> Registry
    Manager --> P1
    Manager --> P2
    Manager --> Pn
    
    P1 --> Jailer
    P2 --> Jailer
    Pn --> Jailer
    
    Jailer --> User
```

### Resource Management

```mermaid
graph LR
    subgraph "Resource Limits"
        Memory[Memory Limits<br/>- cgroups v2<br/>- Per-VM Allocation<br/>- OOM Protection]
        CPU[CPU Limits<br/>- CPU Shares<br/>- Process Affinity<br/>- Throttling]
        IO[I/O Limits<br/>- Disk Bandwidth<br/>- IOPS Limits<br/>- Priority Classes]
    end
    
    subgraph "Monitoring"
        Usage[Usage Tracking<br/>- Real-time Metrics<br/>- Historical Data<br/>- Trend Analysis]
        Alerts[Alert Thresholds<br/>- Resource Exhaustion<br/>- Performance Degradation<br/>- Capacity Planning]
    end
    
    Memory --> Usage
    CPU --> Usage
    IO --> Usage
    
    Usage --> Alerts
```

---

## Billing Integration

### Metrics Collection Architecture

```mermaid
graph TB
    subgraph "VM Runtime"
        VM[VM Instance<br/>- Running Workload<br/>- Resource Usage<br/>- Performance Metrics]
    end
    
    subgraph "Collection Layer"
        FIFO[FIFO Stream<br/>- 100ms Precision<br/>- JSON Format<br/>- Real-time Data]
        Parser[Metrics Parser<br/>- JSON Validation<br/>- Data Enrichment<br/>- Error Handling]
    end
    
    subgraph "Transmission Layer"
        Buffer[Metrics Buffer<br/>- Batch Aggregation<br/>- Retry Logic<br/>- Failure Recovery]
        Client[Billing Client<br/>- HTTP Transport<br/>- Authentication<br/>- Heartbeat]
    end
    
    VM --> FIFO
    FIFO --> Parser
    Parser --> Buffer
    Buffer --> Client
    Client --> BillingService[Billing Service]
```

### Billing Data Flow

```mermaid
sequenceDiagram
    participant VM
    participant FIFO
    participant Parser
    participant Client
    participant Billing
    
    Note over VM,Billing: Metrics Collection Setup
    VM->>FIFO: Initialize Stream
    FIFO->>Parser: Start Parsing
    Parser->>Client: Begin Collection
    Client->>Billing: Register VM
    
    loop Every 100ms
        VM->>FIFO: Write Metrics JSON
        FIFO->>Parser: Parse JSON
        Parser->>Parser: Validate & Enrich
        Parser->>Client: Add to Batch
        
        alt Batch Full or Timeout
            Client->>Billing: Send Batch
            Billing->>Client: Acknowledge
        end
    end
    
    Note over VM,Billing: Error Handling
    alt Collection Error
        Parser->>Client: Report Error
        Client->>Billing: Send Error Report
        Client->>Client: Retry Logic
    end
    
    Note over VM,Billing: VM Shutdown
    VM->>FIFO: Close Stream
    FIFO->>Parser: End of Stream
    Parser->>Client: Final Batch
    Client->>Billing: Final Metrics
    Billing->>Client: Acknowledge
```

---

## Operational Considerations

### Performance Characteristics
- **VM Creation Time**: ~2-5 seconds for Firecracker, ~5-10 seconds for Cloud Hypervisor
- **Memory Overhead**: ~50MB per VM process
- **CPU Overhead**: ~2-5% per VM for management
- **Network Latency**: <1ms additional latency for API calls
- **Billing Precision**: 100ms metrics collection interval

### Scalability Limits
- **Maximum VMs per Instance**: 1000 (configurable)
- **Memory Requirements**: 8GB + (VM count × 50MB)
- **CPU Requirements**: 4 cores minimum, scales with VM count
- **Network Bandwidth**: 1Gbps recommended for 100+ VMs
- **Storage IOPS**: 1000 IOPS minimum for SQLite WAL

### High Availability
- **Process Isolation**: VM failures don't affect other VMs
- **Backend Abstraction**: Can switch backends during maintenance
- **Database Reliability**: SQLite WAL mode for crash recovery
- **Graceful Shutdown**: Clean VM termination on service restart
- **Health Monitoring**: Continuous VM and process health checks

### Monitoring & Alerting
- **VM Lifecycle Metrics**: Creation, boot, shutdown, deletion times
- **Authentication Metrics**: Success/failure rates, customer access patterns
- **Backend Performance**: Operation latencies, error rates
- **Process Health**: Memory usage, CPU usage, process counts
- **Billing Integration**: Metrics collection rates, transmission success

---

## Cross-References

### Architecture Documentation
- **[System Architecture Overview](../overview.md)** - Complete system design
- **[Gateway Architecture](gateway.md)** - API gateway integration
- **[Billing Architecture](billing.md)** - Billing system integration
- **[Security Architecture](../security/overview.md)** - Security design

### API Documentation
- **[API Reference](../../api/reference.md)** - Complete API documentation
- **[Configuration Guide](../../api/configuration.md)** - VM configuration options

### Operational Documentation
- **[Production Deployment](../../deployment/production.md)** - Deployment procedures
- **[Reliability Guide](../../operations/reliability.md)** - Operational procedures
- **[Troubleshooting](../../operations/troubleshooting.md)** - Problem resolution

### Development Documentation
- **[Testing Guide](../../development/testing/stress-testing.md)** - Load testing procedures
- **[Contribution Guide](../../development/contribution-guide.md)** - Development setup

---

*Last updated: 2025-06-12 | Next review: Metald Architecture Review*