# System Architecture Overview

> Comprehensive guide to the Unkey VM management platform architecture

## Table of Contents
- [High-Level Architecture](#high-level-architecture)
- [Component Overview](#component-overview)
- [Request Flow Architecture](#request-flow-architecture)
- [Data Flow Architecture](#data-flow-architecture)
- [Multi-Tenant Architecture](#multi-tenant-architecture)
- [Security Architecture](#security-architecture)
- [Deployment Architecture](#deployment-architecture)
- [Cross-References](#cross-references)

---

## High-Level Architecture

The Unkey platform consists of four core components orchestrating VM lifecycle management, billing, and analytics:

```mermaid
graph TB
    subgraph "Client Layer"
        API[Client APIs/SDKs]
        Web[Web Dashboard]
        CLI[CLI Tools]
    end
    
    subgraph "Gateway Layer"
        Gateway[Gateway Service<br/>- Authentication<br/>- Rate Limiting<br/>- Request Routing]
    end
    
    subgraph "Core Platform"
        subgraph "VM Management"
            Metald[Metald Core<br/>- VM Lifecycle<br/>- Multi-VMM Support<br/>- Process Management]
            
            subgraph "VM Backends"
                Firecracker[Firecracker VMs<br/>- Production Ready<br/>- Jailer Security<br/>- FIFO Streaming]
                CloudHypervisor[Cloud Hypervisor VMs<br/>- Development/Testing<br/>- Alternative Backend]
            end
        end
        
        subgraph "Billing Pipeline"
            Billing[Billing Service<br/>- 100ms Precision<br/>- Real-time Metrics<br/>- Aggregation]
            Aggregator[Billing Aggregator<br/>- Data Processing<br/>- Batch Operations<br/>- Analytics Prep]
        end
    end
    
    subgraph "Data Layer"
        ClickHouse[(ClickHouse<br/>- Analytics Database<br/>- Time-series Data<br/>- Billing Reports)]
        MetricsDB[(Metrics Storage<br/>- Prometheus<br/>- Time-series)]
        ConfigDB[(Configuration<br/>- System State<br/>- VM Metadata)]
    end
    
    subgraph "Infrastructure"
        IPv6Net[IPv6 Networking<br/>- Security Controls<br/>- Multi-tenant Isolation]
        Security[Security Layer<br/>- Authentication<br/>- Authorization<br/>- Isolation]
        Observability[Observability<br/>- OpenTelemetry<br/>- Distributed Tracing<br/>- Metrics]
    end
    
    API --> Gateway
    Web --> Gateway
    CLI --> Gateway
    
    Gateway --> Metald
    Metald --> Firecracker
    Metald --> CloudHypervisor
    Metald --> Billing
    
    Billing --> Aggregator
    Aggregator --> ClickHouse
    
    Metald --> ConfigDB
    Billing --> MetricsDB
    
    Metald -.-> IPv6Net
    Gateway -.-> Security
    Metald -.-> Security
    Billing -.-> Security
    
    Metald --> Observability
    Gateway --> Observability
    Billing --> Observability
    Aggregator --> Observability
```

---

## Component Overview

### Gateway Service
**Purpose**: API gateway handling authentication, routing, and rate limiting  
**Technology**: [To be determined - awaiting implementation]  
**Key Responsibilities**:
- Customer authentication and JWT validation
- Request routing to appropriate metald instances
- Rate limiting and DDoS protection
- Request/response transformation
- API versioning and compatibility

### Metald Core
**Purpose**: VM lifecycle management and multi-VMM orchestration  
**Technology**: Go, ConnectRPC, OpenTelemetry  
**Key Responsibilities**:
- VM creation, boot, shutdown, deletion
- Multi-tenant customer isolation
- Process management (1:1 VM-to-process model)
- Real-time metrics collection
- Backend abstraction (Firecracker/Cloud Hypervisor)

### Billing Service
**Purpose**: Real-time metrics collection and billing aggregation  
**Technology**: Go, 100ms precision streaming, JSON parsing  
**Key Responsibilities**:
- FIFO metrics streaming from VMs
- 100ms precision data collection
- Batch processing and aggregation
- Heartbeat monitoring and failure recovery
- Integration with billing aggregator

### ClickHouse Database
**Purpose**: Analytics database for billing and operational metrics  
**Technology**: ClickHouse (columnar time-series database)  
**Key Responsibilities**:
- Time-series billing data storage
- Real-time analytics queries
- Customer usage reports
- Historical trend analysis
- Data retention management

---

## Request Flow Architecture

### VM Creation Flow

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant Metald
    participant VMM as VM Backend
    participant Billing
    participant ClickHouse
    
    Client->>Gateway: CreateVM Request
    Gateway->>Gateway: Authenticate Customer
    Gateway->>Gateway: Validate Rate Limits
    Gateway->>Metald: Forward CreateVM
    
    Metald->>Metald: Validate Customer Ownership
    Metald->>Metald: Allocate VM Resources
    Metald->>VMM: Create VM Process
    VMM->>VMM: Initialize VM
    VMM->>Metald: VM Process Started
    
    Metald->>Billing: Start Metrics Collection
    Billing->>Billing: Initialize FIFO Stream
    Billing->>ClickHouse: Begin Data Ingestion
    
    Metald->>Gateway: VM Created Response
    Gateway->>Client: Success Response
    
    loop Every 100ms
        VMM->>Billing: Stream Metrics
        Billing->>ClickHouse: Store Metrics
    end
```

### Authentication Flow

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant Auth as Auth Service
    participant Metald
    
    Client->>Gateway: API Request + Token
    Gateway->>Gateway: Extract Bearer Token
    Gateway->>Auth: Validate Token
    Auth->>Gateway: Customer Context
    Gateway->>Gateway: Extract Customer ID
    Gateway->>Metald: Request + Customer Context
    
    Metald->>Metald: Validate Customer Ownership
    alt Authorized
        Metald->>Gateway: Process Request
        Gateway->>Client: Success Response
    else Unauthorized
        Metald->>Gateway: Ownership Violation
        Gateway->>Client: 403 Forbidden
    end
```

---

## Data Flow Architecture

### Billing Data Pipeline

```mermaid
graph LR
    subgraph "VM Runtime"
        VM1[VM Instance 1]
        VM2[VM Instance 2]
        VMn[VM Instance N]
    end
    
    subgraph "Collection Layer"
        FIFO1[FIFO Stream 1<br/>100ms precision]
        FIFO2[FIFO Stream 2<br/>100ms precision]
        FIFOn[FIFO Stream N<br/>100ms precision]
    end
    
    subgraph "Processing Layer"
        Collector[Billing Collector<br/>- JSON Parsing<br/>- Validation<br/>- Enrichment]
        Batcher[Batch Processor<br/>- Aggregation<br/>- Compression<br/>- Retry Logic]
    end
    
    subgraph "Storage Layer"
        ClickHouse[(ClickHouse<br/>- Time-series<br/>- Analytics<br/>- Reports)]
    end
    
    VM1 --> FIFO1
    VM2 --> FIFO2
    VMn --> FIFOn
    
    FIFO1 --> Collector
    FIFO2 --> Collector
    FIFOn --> Collector
    
    Collector --> Batcher
    Batcher --> ClickHouse
    
    ClickHouse --> Reports[Billing Reports]
    ClickHouse --> Analytics[Usage Analytics]
    ClickHouse --> Monitoring[Cost Monitoring]
```

### Observability Data Flow

```mermaid
graph TB
    subgraph "Application Layer"
        Gateway[Gateway Service]
        Metald[Metald Core]
        Billing[Billing Service]
    end
    
    subgraph "OpenTelemetry Layer"
        Traces[Distributed Traces]
        Metrics[Application Metrics]
        Logs[Structured Logs]
    end
    
    subgraph "Storage & Analysis"
        Prometheus[(Prometheus<br/>Metrics)]
        Jaeger[(Jaeger<br/>Traces)]
        Loki[(Loki<br/>Logs)]
    end
    
    subgraph "Visualization"
        Grafana[Grafana Dashboards]
        Alerts[Alert Manager]
    end
    
    Gateway --> Traces
    Gateway --> Metrics
    Gateway --> Logs
    
    Metald --> Traces
    Metald --> Metrics
    Metald --> Logs
    
    Billing --> Traces
    Billing --> Metrics
    Billing --> Logs
    
    Traces --> Jaeger
    Metrics --> Prometheus
    Logs --> Loki
    
    Prometheus --> Grafana
    Jaeger --> Grafana
    Loki --> Grafana
    
    Prometheus --> Alerts
```

---

## Multi-Tenant Architecture

### Customer Isolation Model

```mermaid
graph TB
    subgraph "Customer A"
        ClientA[Client A]
        TokenA[Bearer Token A]
        VMA1[VM A-1]
        VMA2[VM A-2]
        DataA[Customer A Data]
    end
    
    subgraph "Customer B"  
        ClientB[Client B]
        TokenB[Bearer Token B]
        VMB1[VM B-1]
        VMB2[VM B-2]
        DataB[Customer B Data]
    end
    
    subgraph "Shared Infrastructure"
        Gateway[Gateway<br/>- Token Validation<br/>- Customer Context]
        
        subgraph "Metald Instance"
            Auth[Authentication Layer]
            Isolation[Customer Isolation]
            VMManager[VM Manager]
        end
        
        subgraph "Billing Pipeline"
            MetricsA[Customer A Metrics]
            MetricsB[Customer B Metrics]
            ClickHouse[(ClickHouse<br/>Customer Segmented)]
        end
    end
    
    ClientA --> TokenA
    ClientB --> TokenB
    
    TokenA --> Gateway
    TokenB --> Gateway
    
    Gateway --> Auth
    Auth --> Isolation
    
    Isolation --> VMA1
    Isolation --> VMA2
    Isolation --> VMB1
    Isolation --> VMB2
    
    VMA1 --> MetricsA
    VMA2 --> MetricsA
    VMB1 --> MetricsB
    VMB2 --> MetricsB
    
    MetricsA --> ClickHouse
    MetricsB --> ClickHouse
    
    ClickHouse --> DataA
    ClickHouse --> DataB
```

### Security Boundaries

```mermaid
graph TB
    subgraph "Network Layer"
        IPv6[IPv6 Security Controls<br/>- RA Guard<br/>- Source Validation<br/>- Extension Header Filtering]
    end
    
    subgraph "Application Layer"
        subgraph "Gateway Security"
            AuthN[Authentication<br/>- JWT Validation<br/>- Customer Context]
            AuthZ[Authorization<br/>- Resource Access<br/>- Rate Limiting]
        end
        
        subgraph "Metald Security"
            Ownership[Ownership Validation<br/>- Customer VM Access<br/>- Resource Isolation]
            ProcessIsolation[Process Isolation<br/>- 1:1 VM-Process<br/>- Privilege Separation]
        end
        
        subgraph "VM Security"
            Jailer[Jailer Integration<br/>- Chroot Jail<br/>- cgroups v2<br/>- seccomp]
            VMIsolation[VM Isolation<br/>- Memory Isolation<br/>- Network Namespaces]
        end
    end
    
    subgraph "Data Layer"
        DataSegmentation[Data Segmentation<br/>- Customer ID Filtering<br/>- Access Controls<br/>- Audit Logging]
    end
    
    IPv6 --> AuthN
    AuthN --> AuthZ
    AuthZ --> Ownership
    Ownership --> ProcessIsolation
    ProcessIsolation --> Jailer
    Jailer --> VMIsolation
    VMIsolation --> DataSegmentation
```

---

## Security Architecture

### Defense in Depth Model

```mermaid
graph TB
    subgraph "Perimeter Security"
        WAF[Web Application Firewall]
        DDoS[DDoS Protection]
        RateLimit[Rate Limiting]
    end
    
    subgraph "Application Security"
        Authentication[Authentication Layer<br/>- JWT Validation<br/>- Multi-tenant Context]
        Authorization[Authorization Layer<br/>- Resource Access Control<br/>- Customer Ownership]
        InputValidation[Input Validation<br/>- Request Sanitization<br/>- Schema Validation]
    end
    
    subgraph "Runtime Security"
        ProcessSeparation[Process Separation<br/>- 1:1 VM-Process Model<br/>- Privilege Dropping]
        JailerSecurity[Jailer Security<br/>- chroot Isolation<br/>- cgroups Limits<br/>- seccomp Filtering]
        NetworkSecurity[Network Security<br/>- IPv6 Controls<br/>- Namespace Isolation]
    end
    
    subgraph "Data Security"
        Encryption[Data Encryption<br/>- TLS in Transit<br/>- Encryption at Rest]
        AccessControl[Access Control<br/>- Customer Segmentation<br/>- Audit Logging]
        DataMinimization[Data Minimization<br/>- Retention Policies<br/>- Anonymization]
    end
    
    WAF --> Authentication
    DDoS --> Authentication
    RateLimit --> Authentication
    
    Authentication --> Authorization
    Authorization --> InputValidation
    
    InputValidation --> ProcessSeparation
    ProcessSeparation --> JailerSecurity
    JailerSecurity --> NetworkSecurity
    
    NetworkSecurity --> Encryption
    Encryption --> AccessControl
    AccessControl --> DataMinimization
```

---

## Deployment Architecture

### Production Deployment Model

```mermaid
graph TB
    subgraph "Load Balancer Layer"
        LB[Load Balancer<br/>- TLS Termination<br/>- Health Checks<br/>- Geographic Routing]
    end
    
    subgraph "Application Tier"
        subgraph "Gateway Cluster"
            GW1[Gateway Instance 1]
            GW2[Gateway Instance 2]
            GWn[Gateway Instance N]
        end
        
        subgraph "Metald Cluster"
            M1[Metald Instance 1<br/>Customer Pool A]
            M2[Metald Instance 2<br/>Customer Pool B]
            Mn[Metald Instance N<br/>Customer Pool N]
        end
        
        subgraph "Billing Cluster"
            B1[Billing Instance 1]
            B2[Billing Instance 2]
            Bn[Billing Instance N]
        end
    end
    
    subgraph "Data Tier"
        subgraph "ClickHouse Cluster"
            CH1[(ClickHouse Node 1)]
            CH2[(ClickHouse Node 2)]
            CHn[(ClickHouse Node N)]
        end
        
        subgraph "Monitoring Stack"
            Prometheus[(Prometheus)]
            Grafana[Grafana]
            AlertManager[Alert Manager]
        end
    end
    
    LB --> GW1
    LB --> GW2
    LB --> GWn
    
    GW1 --> M1
    GW2 --> M2
    GWn --> Mn
    
    M1 --> B1
    M2 --> B2
    Mn --> Bn
    
    B1 --> CH1
    B2 --> CH2
    Bn --> CHn
    
    M1 --> Prometheus
    M2 --> Prometheus
    Mn --> Prometheus
    
    Prometheus --> Grafana
    Prometheus --> AlertManager
```

---

## Cross-References

### Component Deep-Dives
- **[Gateway Architecture](components/gateway.md)** - Authentication, routing, and rate limiting
- **[Metald Architecture](components/metald.md)** - VM lifecycle and multi-VMM support
- **[Billing Architecture](components/billing.md)** - Real-time metrics and aggregation
- **[ClickHouse Architecture](components/clickhouse.md)** - Analytics and data storage

### Specialized Architecture Topics
- **[IPv6 Networking](networking/ipv6.md)** - Network architecture and security
- **[Security Overview](security/overview.md)** - Comprehensive security architecture
- **[Data Flow Diagrams](data-flow.md)** - Detailed end-to-end flows

### Operational Architecture
- **[Production Deployment](../deployment/production.md)** - Complete deployment guide
- **[Monitoring Setup](../deployment/monitoring-setup.md)** - Observability architecture
- **[Reliability Guide](../operations/reliability.md)** - High availability and recovery

---

*Last updated: 2025-06-12 | Next review: Architecture Review Board*