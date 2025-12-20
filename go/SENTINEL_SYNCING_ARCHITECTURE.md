# Sentinel Syncing Architecture

## Overview

The sentinel syncing architecture is a distributed system that maintains consistency between a centralized control plane and multiple Kubernetes clusters. It enables real-time deployment orchestration across geographical regions and logical shards, providing a declarative model where clusters continuously reconcile their state with the desired configuration from the control plane.

The architecture consists of three main components: the control plane (authoritative state), Krane agents (cluster-side reconciliation), and Sentinel instances (runtime service endpoints). This design follows Kubernetes patterns where agents maintain long-lived streaming connections similar to how kubelets watch the Kubernetes API server.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Control Plane                            │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────────┐    │
│  │   Database  │  │   Cluster    │  │     API Service     │    │
│  │             │  │   Service    │  │                     │    │
│  │ (State Store│  │ (gRPC/Connect│  │   (Deployment Mgmt) │    │
│  │  & Auth)    │  │   Streaming) │  │                     │    │
│  └─────────────┘  └──────────────┘  └─────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
         │                          │
         │ gRPC Streaming (Watch)   │ HTTP/REST
         │                          │
         ▼                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Region: us-east-1                         │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────────┐    │
│  │    Krane    │  │    Krane     │  │      Krane          │    │
│  │  Agent #1   │  │  Agent #2    │  │    Agent #N         │    │
│  │ (Shard: A)  │  │ (Shard: B)   │  │  (Shard: C)         │    │
│  └─────────────┘  └──────────────┘  └─────────────────────┘    │
         │                          │
         │ Kubernetes API            │ Kubernetes API
         ▼                          ▼
┌─────────────────────────┐  ┌─────────────────────────────────┐
│    K8s Cluster A        │  │       K8s Cluster B              │
│ ┌─────┐ ┌─────┐         │  │  ┌─────┐ ┌─────┐ ┌─────┐        │
│ │Sent │ │Sent │         │  │  │Sent │ │Sent │ │Sent │        │
│ │inel │ │inel │         │  │  │inel │ │inel │ │inel │        │
│ │ A-1 │ │ A-2 │         │  │  │ B-1 │ │ B-2 │ │ B-3 │        │
│ └─────┘ └─────┘         │  │  └─────┘ └─────┘ └─────┘        │
└─────────────────────────┘  └─────────────────────────────────┘
```

## Core Components

### Control Plane

The control plane serves as the authoritative source of truth for all deployment and sentinel configurations. It maintains the desired state in a relational database and provides streaming APIs for real-time state distribution.

**Key Responsibilities:**
- Store and validate deployment configurations
- Manage sentinel routing configurations
- Stream state changes to appropriate cluster agents
- Handle authentication and authorization
- Provide APIs for deployment lifecycle management

**Database Schema:**
- `deployments`: Application workload definitions with git metadata and resource specs
- `sentinels`: API gateway/routing instances with health and replica tracking
- `instances`: Runtime representation of deployment pods/containers
- `deployment_topology`: Tracks which regions each deployment should run in

### Krane Agents

Krane agents run within each Kubernetes cluster and act as the bridge between the control plane and the cluster. Each agent is identified by instance ID, region, and shard for proper event routing.

**Key Responsibilities:**
- Maintain persistent streaming connections to control plane
- Reconcile cluster state with desired configuration
- Create/update/delete Kubernetes resources (Deployments, Services, etc.)
- Monitor actual cluster state and report back to control plane
- Handle connection failures with automatic reconnection

**Internal Architecture:**
```
Krane Agent
├── Sentinel Reflector
│   ├── ControlPlane Watcher (sentinel events)
│   ├── Event Buffer (1000 events)
│   ├── Kubernetes Reconciler
│   └── Status Reporter
└── Deployment Reflector
    ├── ControlPlane Watcher (deployment events)
    ├── Event Buffer (1000 events)
    ├── Kubernetes Reconciler
    └── Status Reporter
```

### Sentinel Instances

Sentinels are runtime service endpoints that handle actual API traffic. They are deployed as standard Kubernetes Deployments with configurable replicas and resource allocations.

**Key Responsibilities:**
- Route incoming requests to appropriate deployment instances
- Load balance across healthy instances within the same region
- Provide deployment-specific routing configuration
- Report health and replica status back through Krane

## Data Flow and Syncing Process

The syncing process follows a continuous reconciliation loop with multiple mechanisms to ensure consistency:

### 1. Event Streaming (Real-time Updates)

```
Control Plane → gRPC Stream → Krane Agent → K8s API → Sentinel Instance
```

The control plane streams incremental state changes to all connected Krane agents. Each agent receives only events relevant to its region and shard configuration.

**Streaming Protocol:**
- Connect RPC with bidirectional streaming
- Two stream types: `WatchDeployments` and `WatchSentinels`
- Client authentication via bearer token
- Event filtering by region/shard selectors

### 2. Periodic Full Synchronization

To handle missed events and ensure consistency, each Krane agent performs periodic full synchronization:

```
Every 60 seconds:
1. Create synthetic stream request
2. Receive complete current state for region/shard
3. Reconcile any differences
4. Update control plane with actual cluster state
```

### 3. Bidirectional State Reporting

```
K8s Cluster State → Krane Agent → Control Plane → Database
```

Krane continuously monitors the actual state of Kubernetes resources and reports discrepancies back to the control plane, which updates the database to reflect reality.

## Detailed Event Processing Flow

### Deployment Lifecycle

```
1. API Request → Control Plane
2. Validation & Database Update
3. Stream Event to Region/Shard Krane Agents
4. Krane Reconciler Creates/Updates K8s Deployment
5. K8s Schedules Pods
6. Krane Monitors Pod Status
7. Status Updates Reported Back to Control Plane
```

### Sentinel Lifecycle

```
1. Deployment Ready → Sentinel Config Generated
2. Sentinel State Streamed to Krane Agents
3. Krane Creates Sentinel Deployment & Service
4. Sentinel Starts Routing to Deployment Instances
5. Health Checks Report Back Through Krane
```

## Key Design Patterns

### Declarative State Management

The system follows Kubernetes' declarative pattern where users declare desired state and the system handles the imperative operations. This reduces complexity and improves reliability.

### Event-Driven Architecture

Real-time event streaming enables rapid propagation of changes while minimizing resource usage compared to polling-based approaches.

### Eventual Consistency

The system embraces eventual consistency with multiple convergence mechanisms:
- Real-time streaming for immediate updates
- Periodic sync for recovery guarantees
- Bidirectional status reporting for reality reconciliation

### Circuit Breaker Pattern

All control plane communications include circuit breakers to prevent cascading failures during network issues or control plane outages.

## Tradeoffs and Design Decisions

### Streaming vs Polling

**Decision:** Event streaming with periodic sync fallback
**Rationale:** 
- Streaming provides immediate updates with minimal latency
- Periodic sync ensures recovery from missed events
- Reduces control plane load compared to high-frequency polling
- Follows proven Kubernetes patterns

**Tradeoffs:**
- More complex connection management
- Requires reconnection logic and backoff strategies

### Centralized vs Decentralized State

**Decision:** Centralized control plane with distributed execution
**Rationale:**
- Single source of truth simplifies consistency
- Centralized authentication and authorization
- Easier auditing and compliance
- Simplified multi-tenancy

**Tradeoffs:**
- Control plane becomes a scaling bottleneck
- Network dependency between regions
- Requires high availability design

### Database Schema Normalization

**Decision:** Relational model with some denormalization for performance
**Rationale:**
- Strong consistency guarantees
- Complex queries for multi-tenant isolation
- Transaction support for state updates
- Mature tooling and operational expertise

**Tradeoffs:**
- Scaling challenges at very high write volumes
- Less flexible for schema changes compared to NoSQL

### Event Buffering

**Decision:** In-memory buffers (1000 events) with no-drop policy
**Rationale:**
- Handles temporary network interruptions
- Prevents event loss during reconnection
- Bounded memory usage

**Tradeoffs:**
- Memory consumption per agent
- Potential backpressure if processing lags
- Event loss if buffer overflows (though rarely occurs)

## Failure Handling and Recovery

### Network Partitions

The system handles network partitions gracefully:
- Krane agents continue serving current configurations
- Exponential backoff for reconnection attempts
- Full sync upon reconnection to reconcile state
- Circuit breakers prevent cascading failures

### Control Plane Outages

During control plane outages:
- Krane agents operate with last known good state
- No new deployments can be created
- Existing services continue functioning
- Status reporting pauses and resumes upon recovery

### Kubernetes Cluster Failures

Cluster failures are isolated to the affected region:
- Other regions continue normal operation
- Control plane marks instances as unhealthy
- Automatic recovery when cluster returns
- No impact on control plane availability

## Scalability Considerations

### Control Plane Scaling

- Database sharding by workspace_id for multi-tenant isolation
- Stateless control plane services for horizontal scaling
- Connection pooling and efficient streaming protocols
- Regional edge deployment to reduce latency

### Agent Scaling

- One Krane agent per Kubernetes cluster/shard
- Event processing scales with cluster size
- Minimal resource footprint (CPU, memory)
- Efficient Kubernetes API usage with watchers and informers

### State Distribution Optimization

- Event filtering by region/shard reduces unnecessary traffic
- Synthetic streams enable efficient delta synchronization
- Compression and binary protocols minimize bandwidth usage
- Local caching reduces database load

## Operational Considerations

### Monitoring and Observability

The system provides comprehensive observability:
- Structured logging with correlation IDs
- Prometheus metrics for latency, throughput, errors
- Distributed tracing for request flows
- Health endpoints for all components

### Configuration Management

- Environment-specific configurations via environment variables
- Secret management through vault integration
- Configuration validation at startup
- Runtime configuration reload support

### Deployment Strategies

- Blue-green deployments for control plane
- Rolling updates for Krane agents
- Canary deployments for new features
- Automated rollback on failure detection

## Conclusion

The sentinel syncing architecture provides a robust, scalable foundation for multi-region deployment orchestration. By leveraging proven patterns from Kubernetes and distributed systems design, it achieves both operational simplicity and sophisticated capabilities.

The design prioritizes consistency through multiple convergence mechanisms while maintaining high availability through isolation and failure recovery. The declarative model and event-driven approach enable rapid, reliable deployments across distributed infrastructure while maintaining the flexibility needed for evolving requirements.

This architecture successfully abstracts the complexity of multi-cluster management while providing the performance and reliability needed for production workloads.