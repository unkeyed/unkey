# Detailed Service Interactions and Flow

## Complete System Flow with All Services

```mermaid
graph TB
    subgraph "Client Layer"
        CLI[User CLI]
        SDK[SDKs]
        WEBAPP[Web App]
    end
    
    subgraph "API Gateway (To Be Developed)"
        GATEWAYD[Gatewayd<br/>- Auth/AuthZ<br/>- Rate Limiting<br/>- Request Routing<br/>- API Versioning]
    end
    
    subgraph "Core Services"
        subgraph "Build Service"
            BUILDERD[Builderd :8082<br/>- Image → RootFS<br/>- Security Policies<br/>- Multi-tenant Isolation]
        end
        
        subgraph "Asset Management"
            ASSETMGR[AssetManagerd :8083<br/>- Asset Registry<br/>- Reference Counting<br/>- Storage Backend<br/>- Garbage Collection]
        end
        
        subgraph "VM Management"
            METALD[Metald :8080<br/>- VM Lifecycle<br/>- Resource Scheduling<br/>- Jailer Integration<br/>- Metrics Collection]
        end
    end
    
    subgraph "Billing Pipeline"
        subgraph "Collection"
            BILLAGED[Billaged :8081<br/>- 100ms Collection<br/>- Ring Buffers<br/>- Batching<br/>- Reliability]
        end
        
        subgraph "Processing (To Be Developed)"
            BILLAGG[BillingAggregator<br/>- Stream Processing<br/>- Aggregation<br/>- Enrichment<br/>- Cost Calculation]
        end
    end
    
    subgraph "Storage Systems"
        S3[(S3/Object Storage<br/>VM Assets)]
        SQLITE[(SQLite DBs<br/>Service Metadata)]
        CLICKHOUSE[(ClickHouse<br/>Time Series Data)]
    end
    
    subgraph "Compute Infrastructure"
        subgraph "VM Host"
            JAILER[Jailer Process]
            FC[Firecracker VM]
            METRICS[Metrics FIFO]
        end
    end
    
    %% Client connections
    CLI --> GATEWAYD
    SDK --> GATEWAYD
    WEBAPP --> GATEWAYD
    
    %% Gateway routing
    GATEWAYD -->|Build Request| BUILDERD
    GATEWAYD -->|VM Operations| METALD
    GATEWAYD -->|Asset Queries| ASSETMGR
    
    %% Build flow
    BUILDERD -->|Register Asset| ASSETMGR
    BUILDERD -->|Store RootFS| S3
    
    %% VM creation flow
    METALD -->|List/Prepare Assets| ASSETMGR
    ASSETMGR -->|Copy to Jailer| JAILER
    METALD -->|Create VM| JAILER
    JAILER -->|Spawn| FC
    
    %% Metrics flow
    FC -->|100ms| METRICS
    METRICS -->|Read| METALD
    METALD -->|Stream| BILLAGED
    BILLAGED -->|60s Batch| BILLAGG
    BILLAGG -->|Store| CLICKHOUSE
    
    %% Storage connections
    ASSETMGR --> S3
    ASSETMGR --> SQLITE
    METALD --> SQLITE
    BILLAGED --> SQLITE
    
    style GATEWAYD fill:#f9a825,stroke:#f57c00,stroke-width:3px
    style METALD fill:#1976d2,stroke:#0d47a1,stroke-width:2px
    style BUILDERD fill:#7b1fa2,stroke:#4a148c,stroke-width:2px
    style ASSETMGR fill:#00897b,stroke:#00695c,stroke-width:2px
    style BILLAGED fill:#388e3c,stroke:#1b5e20,stroke-width:2px
    style BILLAGG fill:#d32f2f,stroke:#b71c1c,stroke-width:3px
    style CLICKHOUSE fill:#ff6f00,stroke:#e65100,stroke-width:2px
```

## Service Communication Patterns

### 1. Synchronous Request/Response

```mermaid
sequenceDiagram
    participant Client
    participant Gatewayd
    participant Service
    participant Database

    Client->>Gatewayd: HTTPS Request
    activate Gatewayd
    
    Note over Gatewayd: Authenticate & Authorize
    
    Gatewayd->>Service: ConnectRPC Call
    activate Service
    
    Service->>Database: Query
    Database-->>Service: Result
    
    Service-->>Gatewayd: Response
    deactivate Service
    
    Gatewayd-->>Client: JSON/Protobuf Response
    deactivate Gatewayd
```

### 2. Asynchronous Processing

```mermaid
sequenceDiagram
    participant Client
    participant Gatewayd
    participant Builderd
    participant Queue
    participant Worker

    Client->>Gatewayd: Start Build
    Gatewayd->>Builderd: BuildImage()
    
    Builderd->>Queue: Enqueue Job
    Builderd-->>Gatewayd: job_id: "123"
    Gatewayd-->>Client: 202 Accepted {job_id}
    
    Note over Client: Poll for status
    
    Queue->>Worker: Dequeue Job
    activate Worker
    Worker->>Worker: Process Build
    Worker->>Queue: Update Status
    deactivate Worker
    
    Client->>Gatewayd: GET /jobs/123
    Gatewayd->>Queue: Check Status
    Queue-->>Gatewayd: Status: Complete
    Gatewayd-->>Client: 200 OK {result}
```

### 3. Streaming Communication

```mermaid
sequenceDiagram
    participant VM
    participant Metald
    participant Billaged
    participant BillingAgg

    Note over VM,BillingAgg: Continuous Metrics Stream
    
    VM->>Metald: Metrics FIFO
    
    loop Every 100ms
        Metald->>Metald: Read Metrics
        Metald->>Billaged: StreamMetrics()
        Note right of Billaged: Buffer in memory
    end
    
    loop Every 60s
        Billaged->>BillingAgg: SendBatch()
        BillingAgg-->>Billaged: ACK
        Note right of BillingAgg: Process async
    end
```

## Detailed Service Flows

### 1. Complete Application Deployment Flow

```mermaid
flowchart TB
    Start([User: Deploy App])
    
    subgraph "Phase 1: Authentication"
        Auth{Authenticate}
        AuthFail[Return 401]
    end
    
    subgraph "Phase 2: Build"
        BuildCheck{Image Exists?}
        BuildStart[Start Build]
        BuildProcess[Extract RootFS]
        BuildSecure[Apply Policies]
        RegisterAsset[Register with AssetManagerd]
    end
    
    subgraph "Phase 3: VM Creation"
        SelectAssets[Query Available Assets]
        AllocateResources[Allocate CPU/Memory]
        PrepareJailer[Setup Jailer Chroot]
        CopyAssets[Copy Assets to Chroot]
        CreateVM[Create Firecracker Process]
        ConfigureVM[Configure Resources]
        BootVM[Boot VM]
    end
    
    subgraph "Phase 4: Billing"
        StartBilling[Initialize Billing]
        CollectMetrics[Start Metrics Collection]
        StreamToBillaged[Stream to Billaged]
    end
    
    subgraph "Phase 5: Response"
        Success[Return Endpoint]
        Failure[Return Error]
    end
    
    Start --> Auth
    Auth -->|Success| BuildCheck
    Auth -->|Fail| AuthFail
    
    BuildCheck -->|No| BuildStart
    BuildCheck -->|Yes| SelectAssets
    
    BuildStart --> BuildProcess
    BuildProcess --> BuildSecure
    BuildSecure --> RegisterAsset
    RegisterAsset --> SelectAssets
    
    SelectAssets --> AllocateResources
    AllocateResources --> PrepareJailer
    PrepareJailer --> CopyAssets
    CopyAssets --> CreateVM
    CreateVM --> ConfigureVM
    ConfigureVM --> BootVM
    
    BootVM -->|Success| StartBilling
    BootVM -->|Fail| Failure
    
    StartBilling --> CollectMetrics
    CollectMetrics --> StreamToBillaged
    StreamToBillaged --> Success
```

### 2. Asset Lifecycle Management

```mermaid
stateDiagram-v2
    [*] --> Registered: Asset Created
    
    Registered --> Available: Validation Pass
    Registered --> Failed: Validation Fail
    
    Available --> InUse: VM References
    Available --> Archived: Manual Archive
    
    InUse --> Available: References = 0
    InUse --> InUse: More References
    
    Available --> Deleted: GC After TTL
    Archived --> Deleted: Manual Delete
    
    Failed --> Deleted: Cleanup
    
    Deleted --> [*]
    
    note right of InUse
        Reference count > 0
        Cannot be deleted
    end note
    
    note right of Available
        Reference count = 0
        Eligible for GC
    end note
```

### 3. Billing Data Pipeline

```mermaid
flowchart LR
    subgraph "Collection Layer"
        VM1[VM 1<br/>CPU: 45%<br/>RAM: 2GB]
        VM2[VM 2<br/>CPU: 80%<br/>RAM: 4GB]
        VMN[VM N<br/>CPU: 20%<br/>RAM: 1GB]
    end
    
    subgraph "Aggregation Layer"
        Buffer1[Buffer VM1<br/>600 samples]
        Buffer2[Buffer VM2<br/>600 samples]
        BufferN[Buffer VMN<br/>600 samples]
        
        Batch[Batch Processor<br/>60s window]
    end
    
    subgraph "Processing Layer"
        Validate[Validate<br/>- Completeness<br/>- Accuracy]
        Enrich[Enrich<br/>- Customer Info<br/>- Pricing Rules]
        Calculate[Calculate<br/>- Usage Cost<br/>- Discounts]
    end
    
    subgraph "Storage Layer"
        Raw[(Raw Metrics<br/>7 day TTL)]
        Hour[(Hourly Agg<br/>90 day TTL)]
        Day[(Daily Agg<br/>2 year TTL)]
        Bill[(Billing Records<br/>Permanent)]
    end
    
    VM1 -->|100ms| Buffer1
    VM2 -->|100ms| Buffer2
    VMN -->|100ms| BufferN
    
    Buffer1 --> Batch
    Buffer2 --> Batch
    BufferN --> Batch
    
    Batch --> Validate
    Validate --> Enrich
    Enrich --> Calculate
    
    Calculate --> Raw
    Calculate --> Hour
    Calculate --> Day
    Calculate --> Bill
```

## Network Communication Details

### 1. Service Mesh Architecture

```mermaid
graph TB
    subgraph "Service A Pod"
        AppA[Application]
        ProxyA[Envoy Sidecar]
    end
    
    subgraph "Service B Pod"
        AppB[Application]
        ProxyB[Envoy Sidecar]
    end
    
    subgraph "Control Plane"
        Pilot[Istio Pilot]
        Citadel[Istio Citadel]
        Galley[Istio Galley]
    end
    
    subgraph "Observability"
        Prometheus[Prometheus]
        Jaeger[Jaeger]
        Grafana[Grafana]
    end
    
    AppA -->|HTTP| ProxyA
    ProxyA -->|mTLS| ProxyB
    ProxyB -->|HTTP| AppB
    
    ProxyA -.->|Config| Pilot
    ProxyB -.->|Config| Pilot
    
    ProxyA -.->|Certs| Citadel
    ProxyB -.->|Certs| Citadel
    
    ProxyA -.->|Metrics| Prometheus
    ProxyB -.->|Metrics| Prometheus
    
    ProxyA -.->|Traces| Jaeger
    ProxyB -.->|Traces| Jaeger
```

### 2. API Gateway Request Flow

```mermaid
flowchart TB
    subgraph "Gatewayd Components"
        LB[Load Balancer]
        
        subgraph "Request Pipeline"
            TLS[TLS Termination]
            Auth[Authentication]
            AuthZ[Authorization]
            Rate[Rate Limiter]
            Route[Router]
            Transform[Transformer]
        end
        
        subgraph "Middleware"
            Retry[Retry Logic]
            Circuit[Circuit Breaker]
            Timeout[Timeout Handler]
            Cache[Response Cache]
        end
    end
    
    subgraph "Backend Services"
        Metald[Metald]
        Builderd[Builderd]
        AssetMgr[AssetManagerd]
    end
    
    Client[Client Request] --> LB
    LB --> TLS
    TLS --> Auth
    Auth --> AuthZ
    AuthZ --> Rate
    Rate --> Route
    
    Route --> Retry
    Retry --> Circuit
    Circuit --> Timeout
    
    Timeout --> Metald
    Timeout --> Builderd
    Timeout --> AssetMgr
    
    Metald --> Transform
    Builderd --> Transform
    AssetMgr --> Transform
    
    Transform --> Cache
    Cache --> Client
```

### 3. High Availability Architecture

```mermaid
graph TB
    subgraph "Region 1"
        subgraph "AZ 1A"
            GW1A[Gatewayd]
            META1A[Metald]
            BUILD1A[Builderd]
        end
        
        subgraph "AZ 1B"
            GW1B[Gatewayd]
            META1B[Metald]
            BUILD1B[Builderd]
        end
        
        subgraph "AZ 1C"
            GW1C[Gatewayd]
            META1C[Metald]
            BUILD1C[Builderd]
        end
        
        CH1[(ClickHouse<br/>Cluster)]
    end
    
    subgraph "Region 2"
        subgraph "AZ 2A"
            GW2A[Gatewayd]
            META2A[Metald]
        end
        
        subgraph "AZ 2B"
            GW2B[Gatewayd]
            META2B[Metald]
        end
        
        CH2[(ClickHouse<br/>Replica)]
    end
    
    GLB[Global Load Balancer]
    
    GLB --> GW1A
    GLB --> GW1B
    GLB --> GW1C
    GLB --> GW2A
    GLB --> GW2B
    
    CH1 -.->|Replication| CH2
    
    style GLB fill:#ff9800
    style CH1 fill:#ff6f00
    style CH2 fill:#ff6f00
```

## Error Handling and Recovery

### 1. Circuit Breaker Pattern

```mermaid
stateDiagram-v2
    [*] --> Closed: Initial State
    
    Closed --> Open: Failure Threshold
    Closed --> Closed: Success
    
    Open --> HalfOpen: Timeout Expires
    
    HalfOpen --> Closed: Success
    HalfOpen --> Open: Failure
    
    note right of Closed
        All requests pass through
        Count failures
    end note
    
    note right of Open
        All requests fail fast
        No load on failing service
    end note
    
    note right of HalfOpen
        Limited requests allowed
        Test if service recovered
    end note
```

### 2. Retry and Backoff Strategy

```mermaid
graph LR
    Request[Initial Request] --> Attempt1{Attempt 1}
    
    Attempt1 -->|Fail| Wait1[Wait 100ms]
    Wait1 --> Attempt2{Attempt 2}
    
    Attempt2 -->|Fail| Wait2[Wait 200ms]
    Wait2 --> Attempt3{Attempt 3}
    
    Attempt3 -->|Fail| Wait3[Wait 400ms]
    Wait3 --> Attempt4{Attempt 4}
    
    Attempt4 -->|Fail| CircuitOpen[Open Circuit]
    
    Attempt1 -->|Success| Success[Return Response]
    Attempt2 -->|Success| Success
    Attempt3 -->|Success| Success
    Attempt4 -->|Success| Success
    
    style CircuitOpen fill:#f44336
    style Success fill:#4caf50
```

This comprehensive documentation provides:
- Complete service interaction flows
- Detailed sequence diagrams for all major operations
- Network architecture and communication patterns
- Error handling and recovery mechanisms
- High availability design
- Security considerations at each layer