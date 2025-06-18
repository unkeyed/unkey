# Service Interaction Diagrams

## Overview

This document illustrates the interactions between gatewayd, metald, builderd, assetmanagerd, and billaged services in the Unkey infrastructure.

## 1. Complete System Architecture Flow

```mermaid
flowchart TB
    subgraph "External Layer"
        USER[Customer/User]
        REGISTRY[Container Registry]
    end

    subgraph "API Gateway Layer"
        GATEWAYD[Gatewayd<br/>API Gateway & Orchestrator]
    end

    subgraph "Build Layer"
        BUILDERD[Builderd<br/>Multi-tenant Build Sandbox]
    end

    subgraph "VM Management Layer"
        METALD[Metald<br/>VM Lifecycle Manager]
        ASSETMGR[AssetManagerd<br/>VM Asset Registry]
    end

    subgraph "Billing Layer"
        BILLAGED[Billaged<br/>Usage Tracking & Billing]
    end

    subgraph "Storage Layer"
        ASSETS[(VM Assets<br/>Kernels, RootFS)]
        METRICS[(Metrics DB<br/>Time Series)]
    end

    subgraph "Compute Layer"
        FC1[Firecracker VM 1]
        FC2[Firecracker VM 2]
        FCN[Firecracker VM N]
    end

    %% User flows
    USER -->|API Requests| GATEWAYD
    
    %% Gatewayd orchestration
    GATEWAYD -->|Build Request| BUILDERD
    GATEWAYD -->|VM Create/Boot| METALD
    
    %% Build flow
    BUILDERD -->|Pull Image| REGISTRY
    BUILDERD -->|Register Built Asset| ASSETMGR
    BUILDERD -->|Store RootFS| ASSETS
    
    %% VM Management flow
    METALD -->|List/Prepare Assets| ASSETMGR
    METALD -->|VM Lifecycle Events| BILLAGED
    METALD -->|Metrics (100ms)| BILLAGED
    METALD -->|Manage| FC1
    METALD -->|Manage| FC2
    METALD -->|Manage| FCN
    
    %% Asset management
    ASSETMGR -->|Read/Write| ASSETS
    
    %% Billing flow
    BILLAGED -->|Store| METRICS
    
    %% Styling
    classDef gateway fill:#f9a825,stroke:#f57c00,stroke-width:2px
    classDef build fill:#7b1fa2,stroke:#4a148c,stroke-width:2px
    classDef vm fill:#1976d2,stroke:#0d47a1,stroke-width:2px
    classDef billing fill:#388e3c,stroke:#1b5e20,stroke-width:2px
    classDef storage fill:#455a64,stroke:#263238,stroke-width:2px
    
    class GATEWAYD gateway
    class BUILDERD build
    class METALD,ASSETMGR vm
    class BILLAGED billing
    class ASSETS,METRICS storage
```

## 2. VM Creation and Build Sequence

```mermaid
sequenceDiagram
    participant User
    participant Gatewayd
    participant Builderd
    participant AssetManagerd
    participant Metald
    participant Billaged
    participant Firecracker

    Note over User,Firecracker: User wants to deploy custom API in a MicroVM
    
    %% Build Phase
    User->>Gatewayd: Deploy API (code/image)
    activate Gatewayd
    
    Gatewayd->>Builderd: BuildFromImage(image, tenant_id)
    activate Builderd
    
    Builderd->>Builderd: Create tenant-isolated build
    Builderd->>Builderd: Pull image from registry
    Builderd->>Builderd: Convert to rootfs
    Builderd->>Builderd: Apply security policies
    
    Builderd->>AssetManagerd: RegisterAsset(rootfs, metadata)
    activate AssetManagerd
    AssetManagerd->>AssetManagerd: Store asset with ID
    AssetManagerd->>AssetManagerd: Calculate checksum
    AssetManagerd-->>Builderd: asset_id
    deactivate AssetManagerd
    
    Builderd-->>Gatewayd: build_id, asset_id
    deactivate Builderd
    
    %% VM Creation Phase
    Gatewayd->>Metald: CreateVm(config, asset_id)
    activate Metald
    
    Metald->>AssetManagerd: ListAssets(type=ROOTFS)
    activate AssetManagerd
    AssetManagerd-->>Metald: available_assets[]
    deactivate AssetManagerd
    
    Metald->>AssetManagerd: PrepareAssets([kernel_id, rootfs_id], vm_id)
    activate AssetManagerd
    AssetManagerd->>AssetManagerd: Stage assets in jailer chroot
    AssetManagerd-->>Metald: asset_paths{}
    deactivate AssetManagerd
    
    Metald->>Firecracker: Create VM Process
    activate Firecracker
    Metald->>Metald: Configure VM resources
    Metald-->>Gatewayd: vm_id
    
    %% Boot Phase
    Gatewayd->>Metald: BootVm(vm_id)
    Metald->>Firecracker: Start VM
    Firecracker-->>Metald: VM Running
    
    Metald->>Billaged: NotifyVmStarted(vm_id, customer_id, timestamp)
    activate Billaged
    Billaged->>Billaged: Initialize billing record
    Billaged-->>Metald: ack
    deactivate Billaged
    
    Metald-->>Gatewayd: VM Running
    deactivate Metald
    
    Gatewayd-->>User: API Deployed (endpoint)
    deactivate Gatewayd
    
    %% Metrics Collection Loop
    loop Every 100ms
        Firecracker->>Metald: Resource Metrics
        Metald->>Metald: Buffer metrics
    end
    
    loop Every 60s
        Metald->>Billaged: SendMetricsBatch(vm_id, metrics[600])
        Billaged->>Billaged: Aggregate & store
    end
    
    deactivate Firecracker
```

## 3. Asset Lifecycle Management

```mermaid
sequenceDiagram
    participant Builderd
    participant AssetManagerd
    participant Metald
    participant Storage

    Note over Builderd,Storage: Asset Registration Flow
    
    %% Asset Registration
    Builderd->>AssetManagerd: RegisterAsset(name, type, location, metadata)
    activate AssetManagerd
    AssetManagerd->>AssetManagerd: Generate asset_id (UUID)
    AssetManagerd->>AssetManagerd: Validate checksum
    AssetManagerd->>Storage: Store metadata in DB
    AssetManagerd->>Storage: Create sharded path (/assets/{first-2-chars}/{id})
    AssetManagerd-->>Builderd: asset{id, status=AVAILABLE}
    deactivate AssetManagerd
    
    Note over Builderd,Storage: Asset Usage Flow
    
    %% Asset Discovery
    Metald->>AssetManagerd: ListAssets(type=KERNEL, labels={})
    activate AssetManagerd
    AssetManagerd->>Storage: Query available assets
    AssetManagerd-->>Metald: assets[]
    deactivate AssetManagerd
    
    %% Asset Preparation
    Metald->>AssetManagerd: PrepareAssets(asset_ids[], target_path, vm_id)
    activate AssetManagerd
    
    loop For each asset
        AssetManagerd->>Storage: Read asset location
        AssetManagerd->>AssetManagerd: Create hard link/copy to target
        AssetManagerd->>AssetManagerd: Update access timestamp
    end
    
    AssetManagerd->>AssetManagerd: AcquireAsset (increment ref count)
    AssetManagerd-->>Metald: asset_paths{id: local_path}
    deactivate AssetManagerd
    
    Note over Builderd,Storage: Asset Cleanup Flow
    
    %% VM Shutdown
    Metald->>AssetManagerd: ReleaseAsset(lease_id)
    activate AssetManagerd
    AssetManagerd->>AssetManagerd: Decrement reference count
    AssetManagerd-->>Metald: ack
    deactivate AssetManagerd
    
    %% Garbage Collection
    AssetManagerd->>AssetManagerd: GarbageCollect(max_age, delete_unreferenced)
    activate AssetManagerd
    AssetManagerd->>Storage: Find unused assets
    AssetManagerd->>Storage: Delete if ref_count=0 & age>max
    AssetManagerd-->>AssetManagerd: freed_bytes
    deactivate AssetManagerd
```

## 4. Billing Integration Flow

```mermaid
flowchart LR
    subgraph "VM Layer"
        VM1[VM 1<br/>customer: A]
        VM2[VM 2<br/>customer: B]
        VM3[VM 3<br/>customer: A]
    end
    
    subgraph "Metald"
        COLLECTOR[Metrics Collector<br/>100ms intervals]
        BUFFER[Ring Buffer<br/>600 samples/VM]
        SENDER[Batch Sender<br/>60s intervals]
        LIFECYCLE[VM Manager]
    end
    
    subgraph "Billaged"
        API[Billing API]
        AGG[Aggregator]
        VALIDATOR[Data Validator]
        STORAGE[ClickHouse]
    end
    
    subgraph "Monitoring"
        HEARTBEAT[Health Monitor]
        GAPS[Gap Detector]
        ALERTS[Alerting]
    end
    
    VM1 -->|metrics| COLLECTOR
    VM2 -->|metrics| COLLECTOR
    VM3 -->|metrics| COLLECTOR
    
    COLLECTOR -->|100ms| BUFFER
    BUFFER -->|60s batch| SENDER
    SENDER -->|SendMetricsBatch| API
    
    LIFECYCLE -->|NotifyVmStarted| API
    LIFECYCLE -->|NotifyVmStopped| API
    
    API --> VALIDATOR
    VALIDATOR --> AGG
    AGG --> STORAGE
    
    SENDER -->|SendHeartbeat| HEARTBEAT
    HEARTBEAT --> GAPS
    GAPS --> ALERTS
    
    style VM1 fill:#e1f5fe
    style VM3 fill:#e1f5fe
    style VM2 fill:#fff3e0
```

## 5. Complete VM Lifecycle with All Services

```mermaid
stateDiagram-v2
    [*] --> BuildRequested: User submits code/image
    
    BuildRequested --> Building: Gatewayd → Builderd
    Building --> AssetRegistered: Builderd → AssetManagerd
    
    AssetRegistered --> VMCreating: Gatewayd → Metald
    VMCreating --> AssetsStaged: Metald → AssetManagerd
    
    AssetsStaged --> VMCreated: Assets prepared in jailer
    VMCreated --> VMBooting: Boot command
    
    VMBooting --> VMRunning: Firecracker started
    VMRunning --> BillingActive: Metald → Billaged
    
    BillingActive --> MetricsFlowing: 100ms collection
    MetricsFlowing --> MetricsFlowing: Continuous monitoring
    
    MetricsFlowing --> VMStopping: Shutdown command
    VMStopping --> VMStopped: Firecracker terminated
    
    VMStopped --> BillingFinalized: Final metrics batch
    BillingFinalized --> AssetsReleased: Metald → AssetManagerd
    
    AssetsReleased --> [*]: Cleanup complete
    
    note right of Building
        Builderd:
        - Pulls image
        - Converts to rootfs
        - Applies policies
    end note
    
    note right of AssetsStaged
        AssetManagerd:
        - Prepares kernel
        - Prepares rootfs
        - Updates ref counts
    end note
    
    note right of MetricsFlowing
        Billaged:
        - Collects every 100ms
        - Batches every 60s
        - Stores in ClickHouse
    end note
```

## 6. Error Handling and Recovery Flows

```mermaid
sequenceDiagram
    participant Gatewayd
    participant Metald
    participant AssetManagerd
    participant Billaged
    
    Note over Gatewayd,Billaged: Asset Unavailable Scenario
    
    Gatewayd->>Metald: CreateVm(config)
    Metald->>AssetManagerd: PrepareAssets(asset_ids[])
    AssetManagerd-->>Metald: Error: Asset not found
    
    alt Fallback to hardcoded assets
        Metald->>Metald: Use hardcoded asset list
        Metald-->>Gatewayd: VM created with defaults
    else No fallback available
        Metald-->>Gatewayd: Error: Required assets unavailable
    end
    
    Note over Gatewayd,Billaged: Billing Service Unavailable
    
    Gatewayd->>Metald: BootVm(vm_id)
    Metald->>Metald: Boot VM successfully
    Metald->>Billaged: NotifyVmStarted(vm_id)
    Billaged-->>Metald: Error: Connection refused
    
    Metald->>Metald: Log error, continue operation
    Metald->>Metald: Buffer metrics locally
    Metald-->>Gatewayd: VM booted (billing degraded)
    
    loop Retry with backoff
        Metald->>Billaged: SendMetricsBatch(buffered_data)
        alt Success
            Billaged-->>Metald: Batch accepted
        else Still failing
            Metald->>Metald: Write to WAL for recovery
        end
    end
```

## Key Integration Points

### 1. **Gatewayd → Builderd**
- Initiates builds for customer code/images
- Tracks build status and completion
- Receives asset IDs for VM creation

### 2. **Builderd → AssetManagerd**
- Registers built rootfs images
- Provides metadata (size, checksum, labels)
- Receives unique asset IDs

### 3. **Gatewayd → Metald**
- Creates and manages VM lifecycle
- Provides VM configuration
- Receives VM status updates

### 4. **Metald → AssetManagerd**
- Queries available assets (kernels, rootfs)
- Prepares assets in jailer chroot
- Manages asset reference counting

### 5. **Metald → Billaged**
- Sends VM lifecycle events (start/stop)
- Streams resource metrics (100ms intervals)
- Handles billing data reliability

## Service Responsibilities

### **Gatewayd**
- API gateway and request routing
- Orchestrates build and VM workflows
- Manages customer authentication

### **Builderd**
- Multi-tenant build isolation
- Container → rootfs conversion
- Security policy enforcement

### **AssetManagerd**
- VM asset registry and storage
- Reference counting and GC
- Asset staging for jailer

### **Metald**
- VM lifecycle management
- Firecracker process management
- Metrics collection

### **Billaged**
- Usage tracking and aggregation
- Billing data persistence
- Cost calculation

## Data Flows

### Build Flow
`Customer Code → Gatewayd → Builderd → RootFS → AssetManagerd → Asset ID`

### VM Creation Flow
`Asset ID → Gatewayd → Metald → AssetManagerd → Firecracker → Running VM`

### Billing Flow
`VM Metrics → Metald → Billaged → ClickHouse → Usage Reports`