# Unkey Deploy Service Architecture Overview

## Table of Contents
1. [System Overview](#system-overview)
2. [Service Components](#service-components)
3. [Infrastructure Architecture](#infrastructure-architecture)
4. [Application Flow](#application-flow)
5. [Sequence Diagrams](#sequence-diagrams)
6. [Data Flow](#data-flow)
7. [Service Communication](#service-communication)

## System Overview

The Unkey Deploy service is a comprehensive VM management platform that handles the entire lifecycle of microVMs from build to billing. The system is designed with microservices architecture for scalability, security, and maintainability.

```mermaid
graph TB
    subgraph "Client Layer"
        CLI[User CLI]
    end
    
    subgraph "API Gateway Layer"
        GATEWAY[Gatewayd<br/>API Gateway & Router]
    end
    
    subgraph "Core Services"
        METALD[Metald<br/>VM Lifecycle Manager]
        BUILDERD[Builderd<br/>Container Builder]
        ASSETMGR[AssetManagerd<br/>Asset Registry]
    end
    
    subgraph "Billing Pipeline"
        BILLAGED[Billaged<br/>Metrics Collector]
        BILLAGG[BillingAggregator<br/>Data Processor]
    end
    
    subgraph "Storage Layer"
        CLICKHOUSE[(ClickHouse<br/>Time Series DB)]
        ASSETS[(VM Assets<br/>Storage)]
        SQLITE[(Service DBs<br/>SQLite)]
    end
    
    subgraph "Compute Layer"
        FC[Firecracker VMs]
    end
    
    CLI --> GATEWAY
    GATEWAY --> METALD
    GATEWAY --> BUILDERD
    
    BUILDERD --> ASSETMGR
    METALD --> ASSETMGR
    METALD --> BILLAGED
    METALD --> FC
    
    BILLAGED --> BILLAGG
    BILLAGG --> CLICKHOUSE
    
    ASSETMGR --> ASSETS
    METALD --> SQLITE
    ASSETMGR --> SQLITE
    BILLAGED --> SQLITE
    
    style GATEWAY fill:#f9a825
    style METALD fill:#1976d2
    style BUILDERD fill:#7b1fa2
    style ASSETMGR fill:#00897b
    style BILLAGED fill:#388e3c
    style BILLAGG fill:#d32f2f
    style CLICKHOUSE fill:#ff6f00
```

## Service Components

### 1. **Gatewayd** (To Be Developed)
- **Purpose**: API Gateway and request router
- **Responsibilities**:
  - Authentication and authorization
  - Request routing to appropriate services
  - Rate limiting and quota management
  - API versioning
  - Request/response transformation
  - Circuit breaking and retries

### 2. **Metald** (VM Lifecycle Manager)
- **Purpose**: Core VM management service
- **Port**: 8080
- **Responsibilities**:
  - VM lifecycle management (create, boot, pause, resume, shutdown, delete)
  - Resource allocation and scheduling
  - Integration with Firecracker/Cloud Hypervisor
  - Metrics collection (100ms intervals)
  - Multi-tenant isolation
  - Jailer integration for security

### 3. **Builderd** (Container Builder)
- **Purpose**: Multi-tenant build service
- **Port**: 8082
- **Responsibilities**:
  - Container image to rootfs conversion
  - Build isolation and security
  - Build caching and optimization
  - Asset registration with AssetManagerd
  - Build provenance tracking

### 4. **AssetManagerd** (Asset Registry)
- **Purpose**: Centralized VM asset management
- **Port**: 8083
- **Responsibilities**:
  - Asset registration and metadata management
  - Reference counting for garbage collection
  - Asset distribution to jailer chroots
  - Storage backend abstraction (local, S3, etc.)
  - Asset versioning and deduplication

### 5. **Billaged** (Metrics Collector)
- **Purpose**: High-frequency metrics collection
- **Port**: 8081
- **Responsibilities**:
  - 100ms metrics collection from VMs
  - Data buffering and batching
  - Reliability and gap detection
  - Initial aggregation
  - Forwarding to BillingAggregator

### 6. **BillingAggregator** (To Be Developed)
- **Purpose**: Billing data processing pipeline
- **Responsibilities**:
  - Advanced aggregation and rollups
  - Data validation and enrichment
  - Multi-dimensional analysis
  - ClickHouse schema management
  - Cost calculation rules engine

## Infrastructure Architecture

```mermaid
graph TB
    subgraph "Network Zones"
        subgraph "DMZ"
            LB[Load Balancer]
            WAF[WAF/DDoS Protection]
        end
        
        subgraph "Application Zone"
            GATEWAY_CLUSTER[Gatewayd Cluster<br/>3+ instances]
            
            subgraph "Service Mesh"
                METALD_CLUSTER[Metald Cluster]
                BUILDERD_CLUSTER[Builderd Cluster]
                ASSETMGR_CLUSTER[AssetManagerd Cluster]
                BILLAGED_CLUSTER[Billaged Cluster]
            end
        end
        
        subgraph "Data Zone"
            BILLAGG_CLUSTER[BillingAggregator<br/>Stream Processing]
            CH_CLUSTER[ClickHouse Cluster<br/>Sharded + Replicated]
            S3[S3-Compatible<br/>Object Storage]
        end
        
        subgraph "Compute Zone"
            subgraph "Host 1"
                FC1[Firecracker VMs]
                JAILER1[Jailer Isolation]
            end
            subgraph "Host N"
                FCN[Firecracker VMs]
                JAILERN[Jailer Isolation]
            end
        end
    end
    
    LB --> WAF
    WAF --> GATEWAY_CLUSTER
    GATEWAY_CLUSTER --> Service Mesh
    
    METALD_CLUSTER --> BILLAGED_CLUSTER
    BILLAGED_CLUSTER --> BILLAGG_CLUSTER
    BILLAGG_CLUSTER --> CH_CLUSTER
    
    ASSETMGR_CLUSTER --> S3
    METALD_CLUSTER --> Compute Zone
```

## Application Flow

### 1. VM Deployment Flow

```mermaid
flowchart LR
    subgraph "User Journey"
        A[User CLI] -->|Deploy Command| B[Gatewayd]
        B -->|Route Request| C{Request Type}
        
        C -->|Build Request| D[Builderd]
        C -->|VM Request| E[Metald]
        
        D -->|1. Build Image| D1[Create Rootfs]
        D1 -->|2. Register| F[AssetManagerd]
        
        E -->|3. Query Assets| F
        F -->|4. Asset List| E
        E -->|5. Create VM| G[Firecracker]
        
        G -->|6. Metrics| H[Billaged]
        H -->|7. Batch| I[BillingAggregator]
        I -->|8. Store| J[(ClickHouse)]
        
        E -->|Success| B
        B -->|Response| A
    end
    
    style A fill:#e1f5fe
    style B fill:#f9a825
    style D fill:#7b1fa2
    style E fill:#1976d2
    style F fill:#00897b
    style H fill:#388e3c
    style I fill:#d32f2f
    style J fill:#ff6f00
```

### 2. Billing Pipeline Flow

```mermaid
flowchart TB
    subgraph "Data Collection"
        VM1[VM 1] -->|100ms| COL1[Collector]
        VM2[VM 2] -->|100ms| COL2[Collector]
        VMN[VM N] -->|100ms| COLN[Collector]
    end
    
    subgraph "Billaged Service"
        COL1 --> BUFFER[Ring Buffer<br/>600 samples/VM]
        COL2 --> BUFFER
        COLN --> BUFFER
        
        BUFFER -->|60s batch| SENDER[Batch Sender]
        SENDER -->|gRPC Stream| QUEUE[Message Queue]
    end
    
    subgraph "BillingAggregator"
        QUEUE --> VALIDATOR[Data Validator]
        VALIDATOR --> ENRICHER[Enrichment<br/>Customer/Pricing]
        ENRICHER --> AGG[Aggregator<br/>1m, 5m, 1h]
        AGG --> CALC[Cost Calculator]
    end
    
    subgraph "ClickHouse"
        CALC --> RAW[Raw Metrics Table<br/>Short TTL]
        CALC --> AGG_TABLE[Aggregated Table<br/>Long TTL]
        CALC --> BILLING[Billing Table<br/>Permanent]
    end
    
    style BUFFER fill:#4caf50
    style AGG fill:#ff9800
    style CALC fill:#f44336
```

## Sequence Diagrams

### 1. Complete VM Deployment Sequence

```mermaid
sequenceDiagram
    actor User
    participant CLI
    participant Gatewayd
    participant Builderd
    participant AssetManagerd
    participant Metald
    participant Firecracker
    participant Billaged
    participant BillingAgg
    participant ClickHouse

    User->>CLI: deploy my-app:latest
    CLI->>Gatewayd: POST /deploy {image: "my-app:latest"}
    
    Note over Gatewayd: Authenticate & Route
    
    %% Build Phase
    Gatewayd->>Builderd: BuildImage(image, customer_id)
    activate Builderd
    Builderd->>Builderd: Pull image
    Builderd->>Builderd: Extract rootfs
    Builderd->>Builderd: Apply security policies
    
    Builderd->>AssetManagerd: RegisterAsset(rootfs, metadata)
    AssetManagerd->>AssetManagerd: Store asset
    AssetManagerd-->>Builderd: asset_id: "abc123"
    deactivate Builderd
    
    Builderd-->>Gatewayd: build_complete(asset_id)
    
    %% VM Creation Phase
    Gatewayd->>Metald: CreateVM(config, asset_id)
    activate Metald
    
    Metald->>AssetManagerd: PrepareAssets([kernel, rootfs], target_path)
    AssetManagerd->>AssetManagerd: Copy/link to jailer chroot
    AssetManagerd-->>Metald: assets_ready
    
    Metald->>Firecracker: Create VM Process (via Jailer)
    Firecracker-->>Metald: process_ready
    
    Metald->>Firecracker: Configure VM
    Metald->>Firecracker: Boot VM
    Firecracker-->>Metald: vm_running
    
    %% Billing Phase
    Metald->>Billaged: NotifyVMStarted(vm_id, customer_id)
    activate Billaged
    Billaged-->>Metald: ack
    
    loop Every 100ms
        Firecracker->>Metald: Metrics
        Metald->>Billaged: ForwardMetrics(vm_id, metrics)
    end
    
    loop Every 60s
        Billaged->>BillingAgg: SendBatch(metrics[])
        BillingAgg->>BillingAgg: Aggregate
        BillingAgg->>ClickHouse: INSERT
    end
    deactivate Billaged
    
    Metald-->>Gatewayd: vm_created(vm_id, endpoint)
    deactivate Metald
    
    Gatewayd-->>CLI: deployment_success(endpoint)
    CLI-->>User: Application deployed at https://...
```

### 2. Asset Management Sequence

```mermaid
sequenceDiagram
    participant Builderd
    participant AssetManagerd
    participant Storage
    participant Metald
    participant Jailer

    Note over AssetManagerd: Asset Registration
    
    Builderd->>Storage: Upload rootfs.ext4
    Storage-->>Builderd: location: "/assets/abc/abc123..."
    
    Builderd->>AssetManagerd: RegisterAsset(name, type, location, size, checksum)
    AssetManagerd->>AssetManagerd: Validate metadata
    AssetManagerd->>AssetManagerd: Create DB entry
    AssetManagerd-->>Builderd: asset{id: "01ABC...", status: AVAILABLE}
    
    Note over AssetManagerd: Asset Preparation for VM
    
    Metald->>AssetManagerd: ListAssets(type: KERNEL)
    AssetManagerd-->>Metald: [kernel1, kernel2, ...]
    
    Metald->>AssetManagerd: ListAssets(type: ROOTFS, labels: {app: "my-app"})
    AssetManagerd-->>Metald: [rootfs1, rootfs2, ...]
    
    Metald->>AssetManagerd: PrepareAssets([kernel_id, rootfs_id], "/srv/jailer/vm-123/root/assets")
    
    AssetManagerd->>Storage: Get asset locations
    Storage-->>AssetManagerd: paths
    
    AssetManagerd->>Jailer: Create target directory
    AssetManagerd->>Jailer: Hard link assets
    
    alt Hard link fails
        AssetManagerd->>Jailer: Copy assets
    end
    
    AssetManagerd->>AssetManagerd: Update access time
    AssetManagerd->>AssetManagerd: Increment ref count
    
    AssetManagerd-->>Metald: asset_paths{kernel: "/path", rootfs: "/path"}
    
    Note over AssetManagerd: Cleanup
    
    Metald->>AssetManagerd: ReleaseAsset(lease_id)
    AssetManagerd->>AssetManagerd: Decrement ref count
    AssetManagerd-->>Metald: released
```

### 3. Billing Data Flow Sequence

```mermaid
sequenceDiagram
    participant VM as Firecracker VM
    participant Metald
    participant Billaged
    participant Queue as Message Queue
    participant BillingAgg as BillingAggregator
    participant ClickHouse
    participant API as Billing API

    Note over VM,Billaged: High-Frequency Collection (100ms)
    
    loop Every 100ms
        VM->>Metald: CPU, Memory, Network, Disk metrics
        Metald->>Metald: Add timestamp & metadata
        Metald->>Billaged: StreamMetrics(vm_id, customer_id, metrics)
        Billaged->>Billaged: Store in ring buffer
    end
    
    Note over Billaged,Queue: Batching & Reliability
    
    loop Every 60s
        Billaged->>Billaged: Prepare batch (600 samples)
        Billaged->>Queue: PublishBatch(customer_id, vm_id, metrics[])
        Queue-->>Billaged: ack
        
        alt Publish fails
            Billaged->>Billaged: Write to WAL
            Billaged->>Billaged: Retry with exponential backoff
        end
    end
    
    Note over Queue,ClickHouse: Processing Pipeline
    
    Queue->>BillingAgg: ConsumeBatch
    activate BillingAgg
    
    BillingAgg->>BillingAgg: Validate data integrity
    BillingAgg->>BillingAgg: Enrich with pricing rules
    BillingAgg->>BillingAgg: Calculate aggregates
    
    par Parallel Writes
        BillingAgg->>ClickHouse: INSERT raw_metrics
    and
        BillingAgg->>ClickHouse: INSERT hourly_aggregates
    and
        BillingAgg->>ClickHouse: INSERT daily_aggregates
    and
        BillingAgg->>ClickHouse: INSERT billing_records
    end
    
    deactivate BillingAgg
    
    Note over ClickHouse,API: Query Path
    
    API->>ClickHouse: SELECT usage WHERE customer_id = ?
    ClickHouse-->>API: Aggregated usage data
    API-->>User: Usage report
```

## Data Flow

### 1. Metrics Pipeline

```mermaid
graph LR
    subgraph "Collection"
        VM[VM Metrics] -->|100ms| FIFO[FIFO/Files]
        FIFO --> Metald
    end
    
    subgraph "Buffering"
        Metald -->|Stream| Billaged
        Billaged --> RingBuf[Ring Buffer<br/>10min retention]
    end
    
    subgraph "Transport"
        RingBuf -->|60s batch| Proto[Protobuf]
        Proto --> Compress[Zstd Compression]
        Compress --> Queue[Kafka/NATS]
    end
    
    subgraph "Processing"
        Queue --> BillingAgg
        BillingAgg --> Validate[Validation]
        Validate --> Enrich[Enrichment]
        Enrich --> Aggregate[Aggregation]
    end
    
    subgraph "Storage"
        Aggregate --> CH[(ClickHouse)]
        CH --> Partition[Date Partitions]
        Partition --> TTL[TTL Policies]
    end
    
    style VM fill:#2196f3
    style RingBuf fill:#4caf50
    style Queue fill:#ff9800
    style CH fill:#ff6f00
```

### 2. Request Flow

```mermaid
graph TB
    subgraph "Request Path"
        CLI[CLI Request] --> TLS[TLS Termination]
        TLS --> Auth[Authentication]
        Auth --> RateLimit[Rate Limiting]
        RateLimit --> Route[Route Decision]
        
        Route --> SvcA[Service A]
        Route --> SvcB[Service B]
        Route --> SvcN[Service N]
    end
    
    subgraph "Response Path"
        SvcA --> Transform[Response Transform]
        SvcB --> Transform
        SvcN --> Transform
        
        Transform --> Cache[Cache Layer]
        Cache --> Compress[Compression]
        Compress --> Response[CLI Response]
    end
    
    subgraph "Observability"
        Route --> Trace[Distributed Tracing]
        Transform --> Trace
        Trace --> OTel[OpenTelemetry]
        OTel --> Jaeger[Jaeger/Tempo]
    end
```

## Service Communication

### 1. Protocol Stack

```yaml
External Communication:
  CLI -> Gatewayd: HTTPS/REST or gRPC-Web
  
Internal Communication:
  Gatewayd -> Services: ConnectRPC (gRPC + HTTP semantics)
  Service -> Service: ConnectRPC
  Billaged -> BillingAgg: gRPC streaming
  BillingAgg -> ClickHouse: Native TCP protocol

Message Formats:
  API: JSON or Protobuf
  Internal: Protobuf exclusively
  Metrics: Protobuf with compression
  
Security:
  External: TLS 1.3, mTLS optional
  Internal: mTLS required
  Service Mesh: Istio/Linkerd for zero-trust
```

### 2. Service Discovery

```mermaid
graph LR
    subgraph "Service Registry"
        Consul[Consul/etcd]
        
        subgraph "Registered Services"
            GW[gatewayd: 192.168.1.10:8000]
            MT[metald: 192.168.1.11:8080]
            BD[builderd: 192.168.1.12:8082]
            AM[assetmanagerd: 192.168.1.13:8083]
            BG[billaged: 192.168.1.14:8081]
        end
    end
    
    subgraph "Health Checking"
        HC[Health Checker] -->|/health| GW
        HC -->|/health| MT
        HC -->|/health| BD
        HC -->|/health| AM
        HC -->|/health| BG
    end
    
    subgraph "Load Balancing"
        LB[Client-side LB]
        LB --> Consul
        LB --> RoundRobin[Round Robin]
        LB --> Weighted[Weighted]
        LB --> Consistent[Consistent Hash]
    end
```

### 3. Failure Handling

```mermaid
stateDiagram-v2
    [*] --> Healthy
    
    Healthy --> Degraded: Service Timeout
    Healthy --> Failed: Service Crash
    
    Degraded --> Healthy: Recovery
    Degraded --> Failed: Continued Failures
    
    Failed --> CircuitOpen: Threshold Exceeded
    CircuitOpen --> HalfOpen: After Timeout
    
    HalfOpen --> Healthy: Test Success
    HalfOpen --> CircuitOpen: Test Failed
    
    Failed --> Healthy: Service Restart
    
    note right of CircuitOpen
        No requests sent
        Fast fail responses
        Prevents cascading failures
    end note
    
    note right of HalfOpen
        Limited test traffic
        Gradual recovery
    end note
```

## Deployment Architecture

### 1. Container Organization

```yaml
services:
  gatewayd:
    image: unkey/gatewayd:latest
    replicas: 3
    resources:
      cpu: 2
      memory: 4Gi
    
  metald:
    image: unkey/metald:latest
    replicas: 3
    resources:
      cpu: 4
      memory: 8Gi
    capabilities:
      - CAP_SYS_ADMIN  # For jailer
      - CAP_NET_ADMIN  # For network namespaces
    volumes:
      - /srv/jailer:/srv/jailer
      - /opt/vm-assets:/opt/vm-assets:ro
    
  builderd:
    image: unkey/builderd:latest
    replicas: 2
    resources:
      cpu: 8
      memory: 16Gi
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    
  assetmanagerd:
    image: unkey/assetmanagerd:latest
    replicas: 2
    resources:
      cpu: 2
      memory: 4Gi
    volumes:
      - /opt/vm-assets:/opt/vm-assets
    
  billaged:
    image: unkey/billaged:latest
    replicas: 3
    resources:
      cpu: 2
      memory: 8Gi  # High memory for buffering
```

### 2. Network Architecture

```mermaid
graph TB
    subgraph "Internet"
        Users[Users]
    end
    
    subgraph "Edge"
        CDN[CDN/WAF]
        LB[Load Balancers]
    end
    
    subgraph "DMZ"
        GW[Gatewayd Cluster]
    end
    
    subgraph "App Network"
        subgraph "Control Plane"
            METALD[Metald]
            BUILDERD[Builderd]
            ASSETMGR[AssetManagerd]
        end
        
        subgraph "Data Plane"
            BILLAGED[Billaged]
            BILLAGG[BillingAggregator]
        end
    end
    
    subgraph "Data Network"
        CLICKHOUSE[(ClickHouse)]
        S3[(Object Storage)]
    end
    
    subgraph "Compute Network"
        FC[Firecracker Hosts]
    end
    
    Users --> CDN
    CDN --> LB
    LB --> GW
    
    GW -.->|Control| Control Plane
    Control Plane -.->|Metrics| Data Plane
    Data Plane -.->|Store| Data Network
    METALD -.->|Manage| FC
    
    style GW fill:#f9a825
    style METALD fill:#1976d2
    style BILLAGED fill:#388e3c
    style CLICKHOUSE fill:#ff6f00
```

## Security Architecture

```mermaid
graph TB
    subgraph "Security Layers"
        subgraph "Network Security"
            FW[Firewalls]
            IDS[IDS/IPS]
            VLAN[VLAN Segmentation]
        end
        
        subgraph "Application Security"
            WAF[Web Application Firewall]
            AUTHZ[Authorization Service]
            SECRETS[Secrets Management]
        end
        
        subgraph "VM Security"
            JAILER[Jailer Isolation]
            SECCOMP[Seccomp Filters]
            CGROUPS[Resource Limits]
            NETNS[Network Namespaces]
        end
        
        subgraph "Data Security"
            ENCRYPT[Encryption at Rest]
            TLS[TLS in Transit]
            AUDIT[Audit Logging]
        end
    end
    
    FW --> WAF
    WAF --> AUTHZ
    AUTHZ --> JAILER
    JAILER --> ENCRYPT
```

This architecture provides:
- **Scalability**: Horizontal scaling of all components
- **Security**: Multiple isolation layers and zero-trust networking
- **Reliability**: Redundancy and failure handling at every level
- **Observability**: Comprehensive metrics, logs, and traces
- **Flexibility**: Pluggable components and storage backends