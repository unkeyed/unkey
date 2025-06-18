# Metald Visual Documentation

This document contains all architectural and flow diagrams for the Metald service. Diagrams are provided in Mermaid format for easy maintenance and version control.

## System Architecture Diagrams

### High-Level System Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        API[API Clients]
        SDK[Go/gRPC SDKs]
        HTTP[HTTP/JSON Clients]
    end
    
    subgraph "Metald Service"
        subgraph "API Layer"
            GRPC[ConnectRPC Server<br/>:8080]
            Auth[Auth Interceptor]
            Health[Health Endpoint<br/>/health]
            Metrics[Metrics Endpoint<br/>/metrics]
        end
        
        subgraph "Business Logic"
            VMService[VM Service<br/>- Lifecycle Management<br/>- State Tracking<br/>- Customer Isolation]
            Registry[VM Registry<br/>- In-Memory Cache<br/>- Process Tracking]
        end
        
        subgraph "Backend Layer"
            FCBackend[Firecracker Backend<br/>- SDK v4 Integration<br/>- Process Management]
            Jailer[Integrated Jailer<br/>- Security Isolation<br/>- Namespace Control]
        end
        
        subgraph "Infrastructure"
            Network[Network Manager<br/>- TAP Devices<br/>- Namespaces<br/>- IPv6/IPv4]
            Storage[Storage Layer<br/>- SQLite DB<br/>- Asset Cache]
            Billing[Billing Collector<br/>- FIFO Reader<br/>- Metrics Export]
        end
    end
    
    subgraph "External Services"
        AssetMgr[AssetManager<br/>:8083]
        Billaged[Billaged<br/>:8081]
        OTEL[OpenTelemetry<br/>:4318]
    end
    
    API --> GRPC
    SDK --> GRPC
    HTTP --> GRPC
    
    GRPC --> Auth
    Auth --> VMService
    VMService --> Registry
    VMService --> FCBackend
    FCBackend --> Jailer
    FCBackend --> Network
    FCBackend --> Storage
    FCBackend --> Billing
    
    Storage -.-> AssetMgr
    Billing -.-> Billaged
    GRPC -.-> OTEL
```

### Component Interaction Diagram

```mermaid
graph LR
    subgraph "Metald Process"
        API[API Handler]
        Service[VM Service]
        Backend[FC Backend]
        Process[Process Manager]
    end
    
    subgraph "VM Process 1"
        Jailer1[Jailer]
        FC1[Firecracker]
        VM1[MicroVM]
    end
    
    subgraph "VM Process 2"
        Jailer2[Jailer]
        FC2[Firecracker]
        VM2[MicroVM]
    end
    
    API --> Service
    Service --> Backend
    Backend --> Process
    Process --> Jailer1
    Process --> Jailer2
    Jailer1 --> FC1
    Jailer2 --> FC2
    FC1 --> VM1
    FC2 --> VM2
```

## Sequence Diagrams

### VM Creation Flow

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant API as Metald API
    participant Auth
    participant Service as VM Service
    participant Backend
    participant Network as Network Mgr
    participant Jailer
    participant FC as Firecracker
    
    Client->>API: POST /CreateVm
    API->>Auth: Validate Token
    Auth-->>API: Customer Context
    API->>Service: CreateVm(customer, config)
    
    Service->>Service: Validate Config
    Service->>Service: Generate VM ID
    Service->>Backend: CreateVM(vmId, config)
    
    Backend->>Network: CreateNamespace(vmId)
    Network-->>Backend: Namespace Created
    
    Backend->>Backend: Prepare Assets
    Backend->>Jailer: RunInJail(vmId, netns)
    
    Note over Jailer: Fork Child Process
    Jailer->>Jailer: Enter Network NS
    Jailer->>Jailer: Create TAP Device
    Jailer->>Jailer: Setup Chroot
    Jailer->>Jailer: Drop Privileges
    Jailer->>FC: Exec Firecracker
    
    FC-->>Backend: Process Started
    Backend-->>Service: VM Created
    Service-->>API: Response
    API-->>Client: 200 OK + VM ID
```

### VM Boot Sequence

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Service
    participant Backend
    participant FC as Firecracker
    participant Billing
    
    Client->>API: POST /BootVm
    API->>Service: BootVm(vmId)
    Service->>Service: Verify State=CREATED
    Service->>Backend: StartVM(vmId)
    
    Backend->>FC: PUT /actions
    Note over FC: Load kernel
    Note over FC: Load rootfs
    Note over FC: Start vCPUs
    
    FC-->>Backend: InstanceStart
    
    Backend->>Billing: StartMetricsCollection
    Note over Billing: Open FIFO
    Note over Billing: Start reading
    
    Backend-->>Service: VM Running
    Service->>Service: Update State=RUNNING
    Service-->>API: Success
    API-->>Client: 200 OK
    
    loop Every 100ms
        FC->>Billing: Metrics JSON
        Billing->>Billing: Parse & Export
    end
```

### Network Setup Flow

```mermaid
sequenceDiagram
    participant Backend
    participant NetMgr as Network Manager
    participant Kernel as Linux Kernel
    participant Jailer
    
    Backend->>NetMgr: SetupNetwork(vmId, config)
    
    NetMgr->>Kernel: ip netns add ns_vm_xxxxx
    Kernel-->>NetMgr: Namespace Created
    
    NetMgr->>NetMgr: Generate TAP Name
    Note over NetMgr: 8-char ID for<br/>15-char limit
    
    NetMgr->>Kernel: ip link add veth
    NetMgr->>Kernel: ip link set veth netns
    
    NetMgr-->>Backend: Network Ready
    
    Backend->>Jailer: RunInJail(netns)
    Jailer->>Kernel: setns(netns)
    Note over Jailer: Now inside namespace
    
    Jailer->>Kernel: Create TAP device
    Note over Jailer: TAP created INSIDE ns
    Jailer->>Kernel: Configure TAP
```

### Authentication Flow

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Interceptor
    participant TokenValidator
    participant Service
    
    Client->>API: Request + Bearer Token
    API->>Interceptor: Extract Auth Header
    
    alt Missing Auth Header
        Interceptor-->>API: 401 Unauthenticated
        API-->>Client: Error Response
    end
    
    Interceptor->>Interceptor: Parse Bearer Token
    Interceptor->>TokenValidator: ValidateToken(token)
    
    alt Development Token
        Note over TokenValidator: Extract customer_id<br/>from dev_customer_xxx
        TokenValidator-->>Interceptor: Customer Context
    else Production Token
        TokenValidator->>TokenValidator: Validate JWT/API Key
        TokenValidator-->>Interceptor: Customer Context
    end
    
    Interceptor->>Interceptor: Add Context to Request
    Interceptor->>Service: Forward Request
    
    Service->>Service: Check Ownership
    Service-->>API: Process Request
    API-->>Client: Success Response
```

## Data Flow Diagrams

### Metrics Collection Flow

```mermaid
graph TB
    subgraph "VM Process"
        FC[Firecracker]
        FIFO[/tmp/vm-xxx/metrics.fifo]
    end
    
    subgraph "Metald Process"
        Reader[FIFO Reader<br/>Goroutine]
        Parser[JSON Parser]
        Buffer[Metrics Buffer]
        Exporter[Metrics Exporter]
    end
    
    subgraph "External Systems"
        Prom[Prometheus<br/>:9464]
        Billaged[Billaged<br/>:8081]
        OTEL[OpenTelemetry<br/>:4318]
    end
    
    FC -->|Write 100ms| FIFO
    FIFO -->|Read| Reader
    Reader --> Parser
    Parser --> Buffer
    Buffer --> Exporter
    Exporter --> Prom
    Exporter --> Billaged
    Exporter --> OTEL
```

### Storage Architecture

```mermaid
graph TD
    subgraph "Storage Layer"
        SQLite[(SQLite DB<br/>/var/lib/metald/metald.db)]
        
        subgraph "Tables"
            VMs[vms table<br/>- vm_id<br/>- customer_id<br/>- config<br/>- state]
            Metrics[metrics table<br/>- vm_id<br/>- timestamp<br/>- cpu/memory]
        end
    end
    
    subgraph "Asset Storage"
        Local[Local Cache<br/>/opt/metald/assets/]
        AssetMgr[AssetManager<br/>Remote Storage]
    end
    
    subgraph "Runtime Storage"
        Chroot[Chroot Dirs<br/>/srv/jailer/]
        Sockets[Socket Files<br/>/opt/metald/sockets/]
        FIFOs[FIFO Files<br/>/tmp/]
    end
    
    Service[VM Service] --> SQLite
    SQLite --> VMs
    SQLite --> Metrics
    
    Backend[FC Backend] --> Local
    Local -.->|Fetch| AssetMgr
    
    Jailer[Integrated Jailer] --> Chroot
    Backend --> Sockets
    Backend --> FIFOs
```

## Network Topology

### VM Network Architecture

```mermaid
graph TB
    subgraph "Host Network"
        Bridge[Bridge br0<br/>192.168.1.1/24]
        HostTAP1[vh_12345678]
        HostTAP2[vh_87654321]
    end
    
    subgraph "Network NS 1"
        TAP1[tap_12345678<br/>192.168.1.100/24]
        VM1[VM 1<br/>eth0]
    end
    
    subgraph "Network NS 2"
        TAP2[tap_87654321<br/>192.168.1.101/24]
        VM2[VM 2<br/>eth0]
    end
    
    Bridge --- HostTAP1
    Bridge --- HostTAP2
    
    HostTAP1 -.->|veth pair| TAP1
    HostTAP2 -.->|veth pair| TAP2
    
    TAP1 --- VM1
    TAP2 --- VM2
```

### IPv6 Network Layout

```mermaid
graph LR
    subgraph "IPv6 Subnet"
        Router[Router<br/>2001:db8::/64]
        
        subgraph "Host"
            Bridge[br0<br/>2001:db8::1/64]
            
            subgraph "VM Networks"
                VM1[VM1<br/>2001:db8::100/64]
                VM2[VM2<br/>2001:db8::101/64]
                VM3[VM3<br/>2001:db8::102/64]
            end
        end
    end
    
    Router --> Bridge
    Bridge --> VM1
    Bridge --> VM2
    Bridge --> VM3
```

## Deployment Architecture

### Single Node Deployment

```mermaid
graph TB
    subgraph "Linux Host"
        subgraph "System Services"
            Systemd[systemd]
            Metald[metald.service]
            Network[Network Stack]
        end
        
        subgraph "Metald Process"
            API[API Server<br/>:8080]
            Backend[FC Backend]
            DB[(SQLite)]
        end
        
        subgraph "VM Processes"
            VM1[FC + VM 1]
            VM2[FC + VM 2]
            VMn[FC + VM n]
        end
        
        subgraph "Capabilities"
            CAP[CAP_SYS_ADMIN<br/>CAP_NET_ADMIN<br/>CAP_SYS_CHROOT<br/>...]
        end
    end
    
    Systemd --> Metald
    Metald --> API
    API --> Backend
    Backend --> DB
    Backend --> VM1
    Backend --> VM2
    Backend --> VMn
    Metald --> CAP
```

### Multi-Node Architecture (Future)

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[HAProxy/Nginx]
    end
    
    subgraph "Metald Cluster"
        M1[Metald Node 1]
        M2[Metald Node 2]
        M3[Metald Node 3]
    end
    
    subgraph "Shared Storage"
        NFS[NFS/GlusterFS<br/>VM Images]
        PG[(PostgreSQL<br/>Cluster State)]
    end
    
    subgraph "Monitoring"
        Prom[Prometheus]
        Graf[Grafana]
    end
    
    LB --> M1
    LB --> M2
    LB --> M3
    
    M1 --> NFS
    M2 --> NFS
    M3 --> NFS
    
    M1 --> PG
    M2 --> PG
    M3 --> PG
    
    M1 -.-> Prom
    M2 -.-> Prom
    M3 -.-> Prom
    
    Prom --> Graf
```

## Security Model

### Privilege Dropping Flow

```mermaid
graph TD
    Start[Metald Start<br/>With Capabilities]
    
    Fork[Fork Child Process<br/>Inherits Capabilities]
    
    Setup[Setup Environment<br/>- Enter Namespace<br/>- Create TAP<br/>- Prepare Chroot]
    
    Chroot[Chroot to Jail<br/>Minimal Filesystem]
    
    Drop[Drop Privileges<br/>- setuid/setgid<br/>- Clear Capabilities]
    
    Exec[Exec Firecracker<br/>As Unprivileged User]
    
    Run[Run VM<br/>No Privilege Escalation]
    
    Start --> Fork
    Fork --> Setup
    Setup --> Chroot
    Chroot --> Drop
    Drop --> Exec
    Exec --> Run
    
    style Start fill:#f99
    style Run fill:#9f9
```

### Defense in Depth

```mermaid
graph TB
    subgraph "Layer 1: API"
        A1[TLS/mTLS]
        A2[Bearer Auth]
        A3[Rate Limiting]
    end
    
    subgraph "Layer 2: Application"
        B1[Input Validation]
        B2[Customer Isolation]
        B3[RBAC]
    end
    
    subgraph "Layer 3: Process"
        C1[Capabilities<br/>Not Root]
        C2[Integrated Jailer]
        C3[Namespace Isolation]
    end
    
    subgraph "Layer 4: Runtime"
        D1[Chroot Jail]
        D2[Dropped Privileges]
        D3[Seccomp Filters]
    end
    
    subgraph "Layer 5: System"
        E1[SELinux/AppArmor]
        E2[Firewall Rules]
        E3[Audit Logging]
    end
    
    A1 --> A2 --> A3
    B1 --> B2 --> B3
    C1 --> C2 --> C3
    D1 --> D2 --> D3
    E1 --> E2 --> E3
    
    A3 --> B1
    B3 --> C1
    C3 --> D1
    D3 --> E1
```

## Monitoring and Alerting

### Observability Stack

```mermaid
graph LR
    subgraph "Metald"
        App[Application]
        OTEL[OTEL SDK]
        Metrics[/metrics]
    end
    
    subgraph "Collection"
        Collector[OTEL Collector<br/>:4318]
        Prom[Prometheus<br/>:9090]
    end
    
    subgraph "Storage"
        Cortex[Cortex/Thanos]
        Loki[Loki]
        Tempo[Tempo]
    end
    
    subgraph "Visualization"
        Grafana[Grafana<br/>:3000]
        Alert[Alertmanager]
    end
    
    App --> OTEL
    OTEL --> Collector
    App --> Metrics
    Metrics --> Prom
    
    Collector --> Cortex
    Collector --> Loki
    Collector --> Tempo
    Prom --> Cortex
    
    Cortex --> Grafana
    Loki --> Grafana
    Tempo --> Grafana
    
    Cortex --> Alert
```

## Source Files

All diagrams in this document are maintained in Mermaid format for version control. To render:

1. **Mermaid CLI**: `mmdc -i diagrams.md -o diagrams.pdf`
2. **Online**: Use [mermaid.live](https://mermaid.live)
3. **VS Code**: Install Mermaid preview extension
4. **GitHub**: Renders automatically in markdown

For complex diagrams requiring more detail, see the architecture document and component-specific documentation.