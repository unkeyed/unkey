# RFC-WIP: Unkey Deploy Braindump

---

## How to read this document

* Assume I am an idiot. 
* If you don't understand something, that's my fault, not yours. 
* If you think I'm wrong, I probably am.
* If you think I missed something, I probably did.
* Please use linear's comment feature (mark a section of text and click the speech bubble) to ask questions, highlight edge cases or anything else. If you want to branch out into slack, please create a thread and then link it here via comment

---

We are building a deployment platform that brings the ergonomics of serverless to virtual machine infrastructure. The platform enables developers to ship APIs globally with predictable performance and long-running executions. The system combines a containerized build process, global routing, microVM-based isolation, and a unified control plane.

Unlike serverless platforms that use ephemeral functions, we use long-lived, versioned deployments. Each deployment runs in isolated Firecracker microVMs across multiple AWS regions. These VMs persist and serve traffic continuously, avoiding cold starts. Rollouts use blue-green deployments and rollbacks are instant.

This RFC defines the system architecture. It covers deployment triggers, build execution, microVM scheduling, gateway traffic forwarding, and service orchestration.

[https://link.excalidraw.com/l/91TnwL7ujW6/7HoCLGHVbqh](https://link.excalidraw.com/l/91TnwL7ujW6/7HoCLGHVbqh)

## Goals & Non-Goals

This RFC defines a deployment platform that delivers serverless simplicity on virtual machine infrastructure. Developers can deploy APIs globally with minimal configuration, predictable performance, and security isolation, without infrastructure expertise.

The platform provides zero-configuration deployments: developers push code and it is built, provisioned, and served from isolated microVMs across multiple regions. The platform automatically scales workloads based on demand. Every deployment includes monitoring, debugging, and instant rollback capabilities with immutable versioning.

The platform supports API workloads, not general-purpose compute or frontend hosting. It provides a serverless developer experience on VM infrastructure with fast startup, isolation, and operational transparency.

### Non-Goals

This platform is **not** a general-purpose container orchestration system and will not provide managed databases. It focuses on API workloads, excluding frontend, mobile, or desktop applications. The first release will not accommodate every infrastructure configuration or edge case. This focused scope delivers an opinionated experience for API deployment.

## Proposed Solution

The deployment platform provides serverless developer experience on dedicated virtual machine infrastructure. The system uses a modular architecture with clear separation between control and data planes to ensure traffic serving operates independently of control plane availability.

Deployments use Firecracker microVMs for isolation, fast startup, and resource control. Each deployment creates an immutable version containing a root filesystem image, resolved environment variables, configuration, and topology (CPU, memory, autoscaling, regions). Code, configuration, or resource changes trigger new version creation. Rollbacks switch traffic at the gateway between live versions.

The platform supports Git-based and CLI-based deployments. Git deployments use webhooks to trigger builds, mapping commits to versions and environments. CLI deployments package and upload local code for iteration and CI/CD integration. Builds run isolated from the control plane in dedicated Firecracker VMs or external CI providers like Depot, producing content-addressed, deduplicated root filesystem images stored in S3.

The data plane is partitioned to minimize the splash radius when things go wrong and compliance requirements. Gateways handle TLS termination, authentication, rate limiting, and routing using locally cached configuration data.

Configuration uses hierarchical environments with variables resolved at version creation time to ensure reproducibility.

All build, deployment, and runtime events log to ClickHouse for querying and analysis. Gateways and microVMs emit structured logs and metrics for monitoring, debugging, and performance analysis. The platform provides CLI and dashboard interfaces for version management, log inspection, and system monitoring.

The deployment platform combines serverless ergonomics with VM infrastructure control and reliability, providing API deployment with global scale, instant rollbacks, and observability.

## System Architecture

### Core Concepts and Mental Model

#### Workspace

The top-level isolation boundary for a team or organization within the platform. Workspaces encapsulate billing, access control, and resource allocation, providing separation between different tenants. Each workspace is automatically provisioned with default `production` and `preview` environments and serves as the container for all projects, environments, and versions. Workspaces define the scope for user permissions and provide the organizational structure for managing multiple projects and teams.

#### Project

An individual deployable API or backend service within a workspace. Projects represent the primary unit for managing code, configuration, and operational history. Each project serves as the main context for version creation, rollback, and monitoring operations. Projects are linked to environments through configurable routing rules and maintain their own operational boundaries for logging, metrics, and access control.

#### Environment

A logical container at the workspace level for configuration and runtime parameters such as `development`, `staging`, or `production`. Environments are user-defined and scoped to a workspace, enabling teams to manage secrets, API keys, rate limits, and other settings independently for each stage of the development lifecycle. Environments support both shared configurations across multiple projects and project-specific customizations through a hierarchical inheritance model.

#### Branch

A version-controlled line of development within a project that corresponds directly to Git branches in connected source code repositories. Branches can be linked to specific environments through configurable defaults, regex-based rules, or manual overrides, ensuring that versions inherit the correct configuration for their intended stage. Examples include `main` for production, `staging` for integration testing, and feature branches for development.

#### Version

An immutable specification that captures the complete state of an application at a point in time. Versions combine a root filesystem image (containing application code and runtime), resolved environment variables, and topology settings including CPU, memory, autoscaling, and regional preferences. Any change to code, configuration, or resource allocation triggers the creation of a new version, providing complete audit trails and enabling reliable rollbacks.

#### RootFS

An immutable build artifact containing the complete application code, runtime dependencies, and operating system libraries needed to run an application in a microVM. Images are produced by our build system from specific source revisions and stored in S3.

#### Gateway

A reverse proxy deployed in each region and partition that serves as the entry point for all external traffic. Gateways handle TLS termination, request authentication, rate limiting, load balancing, and traffic forwarding to microVM instances. They operate with autonomy, relying only on the `/partition` database, ensuring traffic serving remains available regardless of control plane status. Gateways enforce security policies, generate JWT tokens for identity propagation, and provide request logging.

#### Partition

A partition is an entire self-contained dataplane in a separate AWS account that provides isolation between customer segments. Each partition contains its own global accelerator, gateways, metal hosts running metald, and regionally replicated partition databases. Partitions enable independent scaling, maintenance, and capacity management while supporting bring-your-own-cloud scenarios and different operational policies for different customer segments. No resources, network paths, or operational processes are shared between partitions.

#### metald

The microVM lifecycle management service that runs on each EC2 metal instances within each partition. metald provides a gRPC API for provisioning, monitoring, and decommissioning Firecracker microVMs on its host. It talks to firecracker over a unix socket to handle microVM management, configuration, health monitoring, and graceful shutdown procedures. metald maintains a stateless design and may either connect to the `/unkey` database directly or via the control plane.

### Core Architecture

The platform uses a strict separation between the control plane and data plane to ensure traffic serving operates independently of configuration and management operations.

The control plane orchestrates the complete version lifecycle. It creates immutable versions from Git commits or CLI uploads, resolving configuration hierarchies and maintaining deployment history. Build orchestration coordinates isolated builds in dedicated Firecracker VMs or external CI providers, producing rootfs images. The control plane manages user settings, environment variables, secrets, and resource allocation limits across workspaces and projects. When users modify settings, the control plane propagates these changes to all relevant gateway databases through asynchronous task processing. Resource allocation operates through creating and managing `instance_slots` rows in partition databases based on user-configured regional limits. The control plane operates as a single-region Kubernetes deployment with global coordination capabilities but remains intentionally decoupled from live traffic systems.

The data plane handles high-performance traffic serving and user application execution with complete autonomy from the control plane. It consists of gateways, ec2 metal hosts running metald and firecracker, and a regionally replicated database. All configuration required for request handling—routing tables, TLS certificates, API key metadata, and enforcement policies—is replicated locally and cached in memory, enabling autonomous operation. Gateways can provision new microVM instances by claiming instance_slots and coordinating directly with metald without control plane involvement. Traffic management including request routing, authentication, rate limiting, load balancing, and health monitoring operates using only local data. Partition isolation provides complete separation between customer segments with no shared instances, databases, network paths, or operational processes.

Security boundaries are enforced throughout the system. Builds run isolated from the control plane in dedicated Firecracker VMs or external CI providers, preventing untrusted code from impacting orchestration services. Partitions are isolated at the infrastructure level with no shared instances, databases, or network paths, supporting multi-tenancy, compliance, and BYOC scenarios.

### Database Architecture

The platform uses four databases, each optimized for specific responsibilities and access patterns. This separation provides clear boundaries between orchestration, traffic serving, and observability.

The **main database** (referred to as `/unkey`) tracks the complete lifecycle of every version, including configuration, build artifacts, and deployment history as well as existing data like keys, identities and permissions.

The **partition database** (referred to as `/partition`) is optimized for high-throughput, low-latency data plane access. It is regionally replicated for serving live traffic, providing gateways with access to routing tables, TLS certificates, API key metadata, and configuration blobs. Information required for request handling is precomputed and denormalized for constant-time lookups, enabling gateways to make traffic decisions without contacting the control plane or cross-region latency. This architecture supports fast failover, isolation between partitions, and operational independence for the data plane.

The **task database** powers the platform's durable task engine, coordinating all asynchronous and long-running operations. Tasks are serialized using Protocol Buffers and stored in a MySQL-compatible database, supporting reliable execution, retry semantics, and observability. The task database enables the control plane to orchestrate workflows like version creation, background tasks, and artifact cleanup with guarantees about eventual completion and failure recovery.

The **observability database** uses ClickHouse, a columnar analytics database optimized for high-ingest, high-query workloads. ClickHouse stores all build logs, version lifecycle events, gateway and microVM logs, metrics, and traces. This enables fast querying and analytics across large volumes of operational data for real-time monitoring, debugging, and performance optimization.

Database separation provides clear operational boundaries and performance optimization for each workload. The main database prioritizes consistency and durability for metadata, the partition database provides high-availability and low-latency reads, the task database handles reliable coordination and background processing, and ClickHouse manages scalable observability and analytics. Replication strategies match each system's needs: partition database replicated to all regions, main database managed for consistency and transactional integrity, and ClickHouse scaled for high ingest and analytical query performance.

#### Data Synchronization

Data synchronization ensures that changes made to the main database are eventually propagated to all partition databases, maintaining eventual consistency for authentication, routing decisions, and resource allocation. When users modify API keys, identities, or other security-critical data through the dashboard or API, these changes are queued for propagation across all regions and partitions to ensure correct request handling.

The synchronization flow operates asynchronously to provide fast user response times while ensuring reliable propagation. The dashboard and API services directly write changes to the main database, then immediately notify the control plane about the modifications. The control plane enqueues a synchronization task and responds immediately, allowing users to receive confirmation without waiting for partition database updates to complete.

NOTE

```
User Action Flow:
┌─────────────┐    1. Create/Update/Delete    ┌─────────────┐
│ Dashboard/  │──────────────────────────────▶│ Main DB     │
│ API         │                               │ (/unkey)    │
└─────────────┘                               └─────────────┘
       │
       │ 2. Sync notification
       ▼
┌─────────────┐    3. Enqueue task            ┌─────────────┐
│ Control     │──────────────────────────────▶│ Task Queue  │
│ Plane       │                               │             │
└─────────────┘                               └─────────────┘
                                                    │
                                                    │
                                                    ▼
                                             ┌─────────────┐
                                             │ Task Worker │
                                             └─────────────┘
                                                    │
                                                    │ 5. Update gateway DBs
                                                    │
                                                    ▼
                                            ┌───────────────┐
                                            │ /partition DBs│
                                            └───────────────┘
```

Task-based synchronization targets only the partition databases that require the specific changes being made.

The synchronization process includes retry logic and error handling through the task system's built-in capabilities. If any partition database update fails during task execution, the task is retried with exponential backoff until successful or maximum retry attempts are reached. Failed synchronization tasks are logged for operational review.

We will need a similar system to sync data, such as instance state, back to the `/unkey` database because we don't want out dashboard and API to query `/partition` databases directly. I'm not thrilled about this and we should search for a simpler and less error-prone system.

## Deployment System

### Version Workflow and Lifecycle

The platform supports Git-based and CLI-driven workflows for creating versions. Both processes are automated, reproducible, and auditable, creating immutable snapshots of code, configuration, and topology.

Git workflows integrate with source control providers via webhooks. When a developer pushes a commit to a tracked branch, a webhook triggers the control plane to clone the repository at the specific commit SHA. The system determines the environment for the branch using defaults, regex rules, or manual overrides. Environment variables and configuration resolve from workspace, project, and environment scopes following a precedence hierarchy. Each version is created with the configuration intended for its target environment.

CLI workflows allow developers to package local code and upload it using pre-signed S3 URLs from the control plane. This supports rapid iteration, uncommitted changes, and custom CI/CD integration. Builds run isolated from the control plane in dedicated Firecracker VMs or external CI providers, producing oot filesystem images stored in S3.

Once the build completes, the control plane creates a new version object, capturing the root filesystem image, resolved environment variables, configuration, and topology. Versions are immutable: any change to code, configuration, or resource allocation results in the creation of a new version, providing a complete audit trail and enabling reliable rollbacks.

Resource allocation for microVMs is managed through a slot-based system. When a version is created with regional capacity requirements, the control plane creates `instance_slots` rows in the corresponding partition database. Each row represents one potential microVM instance that can be provisioned on-demand by the dataplane. For example, a version configured for "max 5 instances in us-east-1" results in 5 rows with provisioned=false status.

Actual provisioning of microVMs happens on-demand in the dataplane. When gateways receive requests and need new instances, they atomically claim available slots and coordinate directly with metald for provisioning. This removes the control plane from the critical path of runtime provisioning decisions.

### Environment and Branch Linking System

Environment and branch linking manages configuration across development lifecycle stages. Environments are user-defined containers for configuration and runtime parameters (API keys, secrets, rate limits, authentication rules) scoped to a workspace and shared across projects or customized per project.

Each workspace includes two default environments: **production** and **preview**. Users can create additional environments like `staging`, `qa`, or `development` with environment-specific settings.

Branches link to environments through a configurable system. By default, the `main` branch links to `production` while feature and pull request branches link to `preview`. This can be overridden through regex patterns matching branch names to environments. For example, `release/*` branches might route to `staging`, while `hotfix/*` branches might route to a dedicated `hotfix` environment.

Manual overrides provide fine-grained control. Individual branches can be explicitly assigned to any environment, overriding defaults and regex rules. This handles exceptional cases like testing specific branches in production-like environments or temporarily routing traffic for demonstrations.

The hierarchical configuration model allows teams to define common settings at higher levels with targeted overrides at granular levels. For example, a workspace might define shared monitoring and logging configurations while projects extend those with database connections and project-specific environment variables.

#### Environment Linking User Experience

Environment linking provides teams with options for managing configuration across different deployment stages. The system starts with sensible defaults: the `main` branch links to `production` while all other branches link to `preview`. This zero-configuration approach allows teams to start deploying immediately.

Teams can customize branch linking through three configuration layers in order of precedence: manual overrides for specific branches, regex patterns for systematic rules (like `release/*` branches routing to staging), and default fallback behavior. This hierarchy provides predictable behavior while allowing increasing levels of control.

Environment variable resolution happens at version creation time, ensuring that versions remain reproducible even if the underlying configuration changes later. This design prioritizes reliability and debugging capability while preventing accidental production deployments through safe defaults.

### Control Plane Deployment Architecture

The control plane consists of multiple services deployed on a Kubernetes cluster in a single region, providing centralized coordination while maintaining operational resilience. For disaster recovery, the control plane can be quickly spun up in a secondary region if needed. Each service is designed for horizontal scaling and stateless operation, with all persistent state stored in the main database and task database.

#### API Service Deployment

The API service runs as a stateless Go application deployed on Kubernetes with a public address to be accessible from the dashboard or CLI protected by some authentication mechanism.  Multiple replicas are distributed across availability zones within each region to provide redundancy and load distribution. The API service exposes an RPC interface for communication with the control plane.

#### Task Workers Deployment

Task workers are deployed in the same kubernetes cluster but without an ingress. Workers implement graceful shutdown procedures that allow in-flight tasks to complete before termination, preventing task interruption during deployments or scaling events.

### Build System Architecture

The build system transforms source code into immutable root filesystem images through an isolated process that prevents untrusted code from affecting the control plane or other customer workloads. Builds are decoupled from the control plane runtime and execute in dedicated environments with no access to internal services or infrastructure credentials.

Build isolation uses multiple approaches depending on configuration and requirements. One approach uses dedicated Firecracker VMs created fresh for each build, configured with minimal network access for downloading public dependencies while blocking internal service access, and automatically destroyed after build completion. Alternative approaches include integration with external CI providers such as [Depot.dev](https://depot.dev).

The build workflow begins when the control plane enqueues a task in the task database containing the build specification: source reference (commit SHA or uploaded archive), build configuration, environment variables, and resource requirements. Build workers poll for available tasks, acquire leases to prevent duplicate processing, and authenticate with the designated build provider.

Source code acquisition varies by workflow type. For Git-based builds, the build environment clones the repository at the specific commit SHA, ensuring reproducible builds tied to exact code states. For CLI-based builds, the source archive is uploaded to S3 using pre-signed URLs. Build environments may validate source integrity and scan for basic security issues before proceeding with the build process.

Images are uploaded to S3 with encryption at rest, and references are stored in the rootfs_images table in the `/unkey` database.

Build monitoring and logging provide visibility into the build process. All build output, system events, and performance metrics are captured in real-time and streamed to ClickHouse for availability through dashboard and CLI tools. Each build is tracked through a record in the builds table, including metadata such as the associated version, build tool used, status lifecycle, and reference to the resulting rootfs image.

### Task System and Orchestration

The task system coordinates all asynchronous and long-running operations across the platform through a durable queue backed by MySQL/Vitess via PlanetScale. This system handles orchestration complexity of managing distributed infrastructure while providing guarantees about eventual completion, failure recovery, and operational visibility.

Example task types include building root filesystems, reconciling microVM state, updating routing configurations, cleaning up expired resources, and coordinating multi-step workflows across regions and partitions.

The control plane inserts tasks into the queue along with metadata including priority, visibility timeout, and maximum retry count.

Task execution operates under a lease system that prevents duplicate processing and enables failure recovery. A task may only be processed by one worker at a time. When a worker acquires a task, it obtains a lease for a specified duration. If the worker crashes or stalls, the lease expires and the task becomes eligible for reprocessing by another worker. Workers send periodic heartbeats to retain their lease, and long-running tasks can extend their lease window as needed to prevent premature timeout.

The system supports workflow patterns for distributed infrastructure management. Task chaining allows one task to enqueue follow-up tasks based on its results, enabling workflows such as "provision certificate, then check for domain ownership". 

Tasks include partition identifiers that enable workers to target the appropriate aws account, partition databases, and infrastructure resources.

The task database prioritizes reliable execution, visibility, and retry semantics over low-latency access. All tasks are auditable and inspectable, supporting operational transparency and debugging. The system gracefully degrades under failure conditions without losing work, and provides metrics and logging for monitoring system health and performance.

## Public API

The public API at [api.unkey.com](http://api.unkey.com) serves as the programmatic interface for developers and automation tools, exposing functionality for version management, configuration, and monitoring. The API operates as a client of the control plane rather than being part of the core infrastructure, using the same internal APIs and services that power the web dashboard and other internal tools.

Authentication and authorization for API access uses root keys scoped to specific workspaces and permissions. 

Core API functionality includes creating and managing versions through endpoints that trigger builds and coordinate infrastructure provisioning. Configuration management endpoints allow manipulation of environments, variables, and project settings. Monitoring and observability endpoints provide access to logs, metrics, and version status information. The API also supports querying metadata about workspaces, projects, and infrastructure resources.

Integration with control plane services is handled through internal service interfaces that maintain proper security boundaries and access controls. The API validates all requests, enforces permissions, and coordinates with the control plane to fulfill operations.

## Data Plane

The data plane handles live traffic serving and user application execution in isolated microVMs. It operates independently from the control plane to ensure user-facing APIs remain available during control plane maintenance or outages.

The data plane as a concept is organized into partitions that provide complete isolation between customer segments. Each partition operates independently with its own infrastructure and databases in its own AWS account.

### Request Routing and Instance Management

Request routing and instance management handle the complete flow of incoming requests through multiple scenarios based on microVM availability and regional constraints. Global Accelerator always routes traffic to the nearest region, where our gateways handle it. The gateway makes routing decisions using locally cached data about version configurations, instance availability, and regional topology to ensure requests reach the appropriate microVM instances with minimal latency. The gateway is able to resolve the hostname of the incoming request to retrieve a list of available microVMs to handle it.

#### Scenario 1: Local microVM Running

When a microVM is running in the local region and can handle the request, the gateway sends the request to it. The gateway queries its local routing database to identify healthy microVM instances for the target version, applies load balancing algorithms to select an appropriate instance, and forwards the request directly to the selected microVM. This represents the optimal case with the lowest possible latency since no cross-region communication or provisioning delays are involved.

The gateway reads health status for all microVM instances from the database. Load balancing considers both instance health and current load distribution to ensure requests are routed to instances capable of handling them efficiently. Authentication, rate limiting, and other policies are enforced at the gateway before forwarding the request to the selected microVM.

#### Scenario 2: Cross-Region Proxying

When no microVM is available locally but the version configuration restricts deployment to specific regions, the gateway transparently proxies the request to a target region where the microVM exists. The gateway queries its database to identify regions where the version is deployed and available, selects the closest target region based on proximity, and establishes a proxy connection to the target region's gateway.

Cross-region proxying maintains the client's original request context while adding minimal latency overhead. The local gateway acts as a transparent proxy, forwarding the request to the target region's gateway on a private port and streaming the response back to the client. This approach provides seamless failover capabilities without requiring client-side retry logic or exposing regional topology to end users.

The gateway in the target region does not perform the typical policy enforcement, as these were already handled by the local gateway. Instead, it simply forwards the request to an available microVM in its region.

#### Scenario 3: On-Demand Instance Provisioning

When no microVM is available for the requested version and the configuration allows provisioning in the current region, the gateway attempts to claim an available firecracker slot from the partition database. The gateway performs an atomic database operation to claim a slot atomically. If successful, the gateway proceeds with host selection and coordinates directly with `metald` to provision the instance while maintaining the client connection with appropriate timeout handling.

**Host Selection Logic**: After claiming a slot, the gateway loads all `metald` hosts from the database and filters to only include hosts with status = 'active' to avoid scheduling on provisioning, draining, or terminated hosts. The gateway then applies availability zone spreading by grouping active hosts by AZ and counting existing instances per AZ for load distribution. Within the preferred AZ (the one with fewer instances), the gateway selects the host with the lowest current load based on heartbeat data (allocated_cpu and allocated_memory_mb). If the preferred AZ has no available capacity, the gateway falls back to other AZs.

If no slots are available (user has reached their configured capacity limit), the gateway handles this through cross-region proxying or returning a 503 response. If provisioning fails or exceeds the timeout threshold, the gateway releases the claimed slot and sends an error response to the client.

metald reports instance health and lifecycle events back to the partition database via heartbeats. When instances terminate for any reason, `metald` automatically releases the slot making it available for future provisioning requests.

### microVM Lifecycle Management via metald

The metald service manages the lifecycle of Firecracker microVMs on each AWS EC2 metal instance within a partition. It operates as a long-running Go binary that provides a type-safe API for provisioning, monitoring, and decommissioning workloads while maintaining isolation between customer applications and resource enforcement.

metald is deployed on bare metal EC2 instances provisioned as part of each partition infrastructure. Every partition includes one or more metald hosts distributed across regions and availability zones to provide redundancy and capacity. The service is started via cloud-init scripts when EC2 instances boot and immediately begins exposing its management API while initializing local resource tracking and health monitoring systems.

The gRPC API exposed by metald uses gRPC for type-safe communication with gateways in the same partition. Gateways can directly request instance provisioning and termination without control plane involvement.

`metald` handles microVM management through direct interaction with the Firecracker socket. When creating a new instance, metald downloads the rootfs image from S3, configures the Firecracker VM with CPU and memory allocations, sets up network interfaces with unique IP addresses from the partition's address space, and injects environment variables before running the user's code.

`metald` tracks microVMs through their complete lifecycle from provisioning through termination. States include `PROVISIONING` during image download and VM configuration, `STARTING` during kernel boot and application initialization, `RUNNING` while actively serving traffic, `STOPPING` during graceful shutdown, `STOPPED` after clean termination, and `FAILED` for instances that encounter errors. All state transitions are logged to ClickHouse with timestamps and resource usage snapshots for operational visibility and debugging.

Health monitoring tracks all local microVMs. TCP connectivity tests verify that instances are reachable on their configured ports. HTTP health endpoint polling checks application-specific health indicators when configured. Resource utilization monitoring tracks CPU, memory, and network usage to detect instances that are failing or consuming excessive resources.

Resource allocation and limit enforcement use Linux control groups and Firecracker's built-in resource management to ensure that each microVM operates within its specified constraints. CPU limits prevent applications from consuming more cycles than allocated. Memory limits use kernel accounting to enforce hard boundaries. Network bandwidth shaping controls traffic rates. Resource metrics are continuously collected and reported to ClickHouse for billing and monitoring.

Graceful shutdown procedures handle instance termination with configurable timeouts to allow applications to complete in-flight work. When instructed to terminate an instance, `metald` sends a `SIGTERM` signal to the application process and waits for a configured duration, typically 30-60 seconds but configurable up to a platform maximum of 5 minutes. If the application fails to exit cleanly within the timeout, metald forcefully stops the VM and cleans up all associated resources including network configuration and temporary storage.

Idle timeouts provide automatic cost optimization through instance deallocation. Each version includes configurable idle timeout settings with a default of 10 minutes - instances that receive no traffic for the specified duration are automatically terminated. Users can adjust this timeout per version or disable it entirely. metald runs periodic cleanup cycles every few minutes. When an instance exceeds its configured idle timeout and has been running longer than the minimum instance lifetime (default 2 minutes to prevent thrashing), metald initiates graceful termination: updating the instance status to `STOPPING` to prevent new traffic routing, sending `SIGTERM` with the configured graceful shutdown timeout, and releasing the corresponding instance_slots row to make capacity available for future provisioning.

We assign each microVM a unique private IP address from the partition's CIDR block and configure routing rules that allow gateways to reach instances while preventing cross-customer network access.

`metald` maintains a stateless design. All authoritative information about resource allocation is maintained in the partition's partition database through the `instance_slots` table. This design enables rolling updates of `metald` itself, simplifies disaster recovery, and ensures that provisioning decisions are made autonomously by the dataplane based on available slots.

### Infrastructure Health and Discovery

Infrastructure health and discovery maintains real-time visibility into available infrastructure components across all partitions and regions through periodic heartbeat mechanisms.

#### metald Heartbeat System

`metald` instances maintain two separate heartbeat cycles to the partition database within their partition. Host-level heartbeats update the `metal_hosts` table every X seconds with current capacity, allocation, and overall host health status. This information enables gateways to make informed host selection decisions during on-demand provisioning based on available resources, host health, and operational status (active, draining, etc.).

Instance-level heartbeats update the `instances` table every X seconds with the current status and health of all microVMs running on the host. Each heartbeat includes the instance status (running, stopping, failed), health check results (healthy, unhealthy, unknown), resource utilization metrics, and network connectivity status. This frequent updating ensures that gateways have current information about which instances are available to receive traffic.

The heartbeat payloads include essential operational data for routing decisions. Host heartbeats report total and available CPU millicores, memory usage, disk space, network throughput, and any host-level alerts or degradation. Instance heartbeats include individual microVM health endpoint responses, TCP connectivity test results, CPU and memory utilization, and any application-specific health indicators configured for the version.

Failure detection operates through heartbeat timeout mechanisms. If a metald instance fails to send host heartbeats for 30 seconds, the host is marked as unavailable and no new instances are scheduled on it. If instance heartbeats are missed for 30 seconds, those specific instances are marked as unhealthy and removed from traffic routing until they resume heartbeating successfully.

#### Gateway Heartbeat System

Gateway instances maintain heartbeats to their partition's partition database every 15 seconds, updating the `gateways` table with current status, performance metrics, and capacity information.

Regional gateway discovery operates through the heartbeat system, allowing gateways to maintain awareness of healthy gateway instances in other regions for cross-region request proxying. When performing cross-region proxying, gateways query the database for healthy gateway instances in target regions and establish connections based on current load and performance metrics.

#### Health-Based Routing Integration

Gateways use heartbeat data for their routing decisions through continuous database polling and local caching. The gateway routing cache refreshes frequently, pulling updated instance health status, gateway availability, and capacity metrics from the database. This cached information drives all routing decisions without introducing database query latency into the request path.

Health status influences routing algorithms through weighted load balancing and circuit breaker patterns. Healthy instances receive full traffic weight, while instances with recent health check failures receive reduced weight until they demonstrate consistent health. Instances marked as unhealthy are completely excluded from routing until their heartbeats resume and health checks pass.

Cross-region routing decisions leverage gateway heartbeat data to select optimal target regions for request proxying. When no local instances are available, gateways choose target regions to proxy the request based on proximity and capacity.

### Gateway Infrastructure and Traffic Management

Gateways serve as the edge nodes of the platform, handling all incoming traffic destined for user applications. Deployed across multiple AWS regions and fronted by AWS Global Accelerator, gateways perform TLS termination, request authentication, rate limiting, load balancing, and traffic forwarding to microVM instances.

The gateway architecture is optimized for low latency and high availability through local caching and precomputed decision trees. All routing and enforcement logic is stored as packed Protobuf blobs in the partition's database, with each blob containing information to serve requests for a specific hostname. When a request arrives, the gateway extracts the hostname using Server Name Indication and performs a direct lookup in its cache or database. If a matching entry is found, the gateway decodes the Protobuf payload for request processing, allowing subsequent routing operations to proceed without additional database queries.

Gateway clusters are deployed on Kubernetes within each region in a partition, with multiple instances for redundancy and load distribution. Kubernetes manages health checking, rolling updates, and resource allocation for gateway pods. TLS termination is handled dynamically using Go's tls.Config.GetCertificate method for SNI-based certificate selection, loading certificate chains and private keys from the local database cache during TLS handshakes.

Load balancing distributes traffic across available microVM instances using weighted round-robin distribution based on routing weights, health status, and optional session affinity requirements. The algorithm can be influenced by blue-green rollout weights during version transitions or sticky session configuration through the Unkey-Session-Affinity header. Health status information is continuously updated by metald processes and reflected in routing decisions to ensure traffic reaches only healthy instances. During version transitions, blue-green rollout weights influence traffic distribution to gradually shift load from old to new instances, while session affinity can be configured through special headers for applications requiring server-side state.

Gateways maintain persistent connection pools to reduce connection establishment overhead, implement configurable limits and timeouts to prevent resource exhaustion, and support HTTP protocols. Failed connections trigger circuit breaker protection to let the vm recover if possible.

Request logging captures comprehensive information about every request processed by the gateway. Log entries include request identifiers, timing information, authentication details, routing decisions, response codes, and error conditions. All logging data is streamed asynchronously to ClickHouse for real-time analysis, debugging, and performance monitoring without impacting request processing latency.

## Security and Configuration

### Authentication and Authorization

Authentication and authorization are enforced directly at the gateway level. Every incoming request is subject to security policies without requiring runtime coordination with the control plane. Security decisions can be made with minimal latency and operate during control plane maintenance or outages.

Gateway-level policy enforcement uses locally cached authentication data replicated to each partition's partition database. All information required for authentication decision (including API key metadata, permission scopes, rate limiting rules, and identity information) is precomputed by the control plane an

When a request is successfully authenticated, the gateway generates a JWT that encodes the caller's identity and permissions and sends it in the `X-Unkey-Identity` header.

Each version is assigned a unique cryptographic key pair during creation, with the private key embedded into the gateway's routing configuration for local token signing and the corresponding public key injected into microVMs at boot time via environment variables. This enables offline token validation using standard libraries in any programming language.

Permission scopes and authorization policies define what actions authenticated callers are authorized to perform. The API key data model includes permission information that can be evaluated by both the gateway and the user application. Gateways enforce platform-level policies such as rate limiting and resource access, while applications can implement fine-grained authorization logic using the permissions included in JWT tokens.

### Configuration Management

Configuration management uses a hierarchical approach for flexibility and simplicity while keeping versions reproducible and secure. The system separates secret storage from variable mapping, maintaining boundaries between configuration and version creation.

#### User-Configurable Options

The platform provides configuration options organized into several categories:

**Resource Allocation and Scaling**

* **Regional instance limits**: Per-region minimum and maximum number of instances (e.g., "us-east-1: min 0, max 5")
* **Instance resources**: CPU and memory per instance

**Runtime and Lifecycle Management**

* **Idle timeout**: Duration before unused instances are automatically terminated (default 10 minutes, can disable)
* **Health check configuration**: Endpoints, timeouts, and retry policies for application health monitoring

**Networking and Domains**

* **Custom domains**: User-provided hostnames with automatic TLS certificate provisioning
* **Port configuration**: Application listening port

**Environment**

* **Environment**: The environment in which the application is deployed.
* **Key Space**: The key space used for authentication and authorization.

**Build Configuration**

* **Build settings**: Build commands, environment, and resource allocation for build process
* **Branch linking rules**: Regex patterns and manual overrides for branch-to-environment mapping

#### Environment Variable Mappings

Project variable mappings provide an indirection layer. At the project level, users define mappings from environment variable names to keys in the environment's key-value store. This indirection allows secrets to be reused across multiple projects within the same environment, enables projects to control their own environment variable naming conventions, and allows projects to selectively include only the secrets they need from the environment's key-value store. For example, a project might map `DATABASE_URL` to `production_db_connection` in the production environment and `staging_db_connection` in the staging environment.

Branch and version override mechanisms provide fine-grained control. Both branches and versions can override the base project configuration using two approaches: mapping overrides that redirect an environment variable to a different key in the environment's store, or direct value overrides that bypass the key-value store entirely and set values directly. This handles exceptional cases such as testing specific configurations or providing temporary overrides for debugging.

Resolution priority follows a precedence hierarchy for predictable configuration behavior. When creating a version, environment variables are resolved using the following order from highest to lowest priority: version direct value overrides, version mapping overrides, branch direct value overrides, branch mapping overrides, project direct values, and project base mappings. This hierarchy enables teams to establish base configurations at the project level while providing targeted overrides for specific branches or versions without duplicating configuration.

Immutable configuration snapshots are captured in each version to ensure reproducibility and prevent configuration drift. All resolved variables are captured in the immutable version object at creation time, ensuring that versions remain reproducible even if the underlying configuration changes after creation. Environment variables are baked into the rootfs image as a file and injected into the environment during machine startup, providing applications with access to their configuration through standard environment variable mechanisms.

Secret management and encryption at rest protect sensitive configuration data. Sensitive variables such as database passwords and API keys can be flagged as secrets, causing them to be displayed as redacted values in the dashboard and CLI tools while being stored encrypted in the database. Encryption uses standard algorithms with key management and rotation procedures.

System variable injection provides applications with metadata about their runtime context without requiring configuration by developers. System variables are automatically injected into every microVM during initialization and are prefixed with `UNKEY_` to prevent naming conflicts with user-defined variables. These variables include version identifiers, project and environment context, region information, JWT verification keys, Git commit information, and hostnames routing to the version.

Environment variable inheritance patterns support consistency and flexibility across the configuration hierarchy. Teams can define common settings at higher levels such as workspace or project scope while providing targeted overrides at more granular levels such as branch or version scope. For example, a workspace might define shared monitoring and logging configurations, while individual projects extend those with their own database connections and application-specific environment variables.

### TLS and Certificate Management

The TLS and certificate management system handles both wildcard subdomains for the platform and custom domains provided by customers, with automatic certificate lifecycle management.

Wildcard certificate management for \*.unkey.app provides automatic HTTPS coverage for all platform-generated subdomains without requiring per-deployment certificate provisioning. The wildcard certificate is issued through integration with either AWS Certificate Manager or cert-manager depending on the deployment environment, with automatic renewal workflows that operate without affecting traffic serving. This certificate is distributed to all gateway instances within each partition through the partition database replication system and cached locally for availability during TLS handshakes.

Custom domain support enables customers to use their own domain names for production APIs. Before a custom domain can be activated for traffic, the system must verify domain ownership to prevent unauthorized certificate issuance and domain hijacking attacks.

SNI-based certificate selection presents the correct certificate for each hostname without requiring dedicated IP addresses or complex routing configurations. The gateway maintains a mapping of hostnames to certificate data in its local database cache, enabling constant-time lookups during TLS handshakes. Certificate data is stored encrypted at rest and decrypted only in memory for handshake processing.

Certificate distribution across gateway instances operates through the standard partition database replication mechanism, ensuring that certificate updates propagate to all instances within a partition automatically. When certificates are renewed or updated, the changes flow through the database replication system and are picked up by gateway instances through their normal cache refresh cycles. This ensures consistent certificate availability across all gateway instances.

## Operations and Observability

### Monitoring and Observability

#### Internal Infrastructure Monitoring

Internal infrastructure monitoring uses Prometheus. Each region within each partition runs its own Prometheus instances to collect metrics from local infrastructure components without cross-region dependencies.

Regional Prometheus instances scrape metrics from all infrastructure components within their scope. Metal hosts can expose system-level metrics including CPU utilization, memory consumption, disk I/O, network throughput, and Firecracker VM resource allocation through node_exporter and custom metald exporters. Control plane pods running in Kubernetes expose application metrics including request rates, error counts, database connection pool status, and task queue depths through standard /metrics endpoints. Data plane components including gateways and supporting services emit operational metrics covering request processing, authentication rates, certificate usage, and health check status.

Each regional Prometheus instance uses remote write to push aggregated metrics to a central Prometheus instance for cross-region and cross-partition visibility.

#### Logging

We provide visibility into all aspects of the system through integrated logging and metrics collection. Built on ClickHouse for high-performance analytics, the system captures data from every component and makes it available for monitoring, debugging, and performance analysis.

ClickHouse serves as the foundation for logging and analytics, providing a columnar analytics database optimized for high-ingest, high-query workloads. ClickHouse stores all build logs, version lifecycle events, gateway and microVM logs, metrics, and traces with structured schemas that enable fast querying and correlation across large volumes of operational data. The system uses automatic retention policies based on customer billing tiers and compression algorithms to manage storage costs while maintaining query performance.

Request logging implementation captures information about every request processed by the gateway infrastructure. Log entries include request identifiers for correlation, timing information including response times and processing latencies, authentication details such as API key validation and identity resolution, routing decisions including target selection and load balancing choices, response codes and error conditions, and source IP addresses and user agent strings. All logging data is streamed asynchronously to ClickHouse through memory-buffered batch writes to minimize impact on request processing latency.

Performance metrics collection operates across all system components to provide visibility into system health and performance. Gateways emit metrics including request latencies, error rates, and throughput measurements. metald processes report resource utilization, instance health status, and VM lifecycle events. The build system tracks build duration, success rates, and resource consumption. The control plane monitors database performance, task queue depth, and API response times. All metrics use pull-based collection through standardized endpoints with consistent naming and labeling schemes.

#### Userspace Prometheus Monitoring

One of the best ideas from [fly.io](http://fly.io) is their automatic prometheus scraping. Users can configure Prometheus-compatible metrics endpoints as part of their version configuration, allowing the platform to scrape application-specific metrics alongside infrastructure monitoring. This provides developers with visibility into their application performance, business metrics, and custom operational data.

User metrics collection operates through a separate set of Prometheus instances dedicated to userspace monitoring, maintaining isolation from infrastructure metrics collection. Each region within each partition deploys dedicated user metrics Prometheus instances that scrape configured endpoints from microVMs within their scope. This separation ensures that user workload metrics collection cannot impact infrastructure monitoring reliability while providing independent scaling characteristics based on user adoption.

Version configuration includes optional metrics endpoint settings where developers can specify the port and path for their Prometheus-compatible metrics endpoint. When configured, the regional user metrics Prometheus instances automatically discover and scrape these endpoints from running microVM instances. The scraping configuration includes appropriate timeouts, retry logic, and error handling to prevent poorly configured user endpoints from affecting the monitoring infrastructure.

Future integration options will provide multiple ways for users to access their collected metrics. Planned capabilities include managed Grafana instances with pre-configured dashboards, integration with third-party monitoring systems through standard APIs, and native metrics visualization within our dashboard. These integration options will enable developers to use their preferred monitoring tools while benefiting from our metrics collection infrastructure.

### Failure Handling and Recovery

The system must handle transient network issues to complete regional outages without impacting the availability of user APIs.

Retry mechanisms with exponential backoff are implemented throughout the system to handle transient failures without overwhelming recovering services. Gateway requests to microVMs use configurable retry policies with exponential backoff and jitter to prevent thundering herd problems. The build system distinguishes between retryable failures such as network timeouts and resource unavailability, and permanent failures such as invalid configurations.

Circuit breaker implementation provides automatic failure isolation and recovery across all system components. Gateway circuit breakers monitor microVM health and automatically stop routing to failing instances, with configurable failure thresholds, timeout periods, and gradual recovery mechanisms. Each circuit breaker includes monitoring and alerting to provide visibility into failure patterns and recovery status.

Rollback procedures and automation enable rapid recovery from problematic versions through multiple mechanisms optimized for different time horizons. For recent versions deployed within the last thirty minutes, instant rollbacks leverage the fact that old instances are still running to enable recovery through gateway configuration changes, completing instantly seconds. For older versions where instances have been terminated, the system potentially needs to download rootFS images again and start the VMs, but still providing access to the complete version history for any recovery scenario.

Graceful degradation during partial failures keeps essential functionality available when some system components are experiencing issues. Gateway failures redirect traffic to healthy regions through Global Accelerator routing. Metal host failure initiates gateways to failover to another available host or another region, ensuring minimal disruption to service availability.

## User Experience

### CLI Architecture and Operations

The CLI serves as a unified interface for the platform, consolidating functionality that might otherwise be scattered across multiple tools into a single, compiled golang binary.

The CLI supports multiple distinct operation modes that address different aspects of the platform lifecycle. For production operations, the CLI functions as a service launcher that can start infrastructure components on provisioned machines. Commands such as `unkey run gateway` start an HTTP gateway server, while `unkey run metald` launches the microVM management daemon on EC2 metal instances. These production commands support configuration through flags or environment variables and are designed for use in automated deployment scripts and service management systems.

Local development environment support enables developers to run complete development environments on their local machines without requiring complex infrastructure setup. The `unkey dev dashboard` command uses Docker Compose or similar tooling to start the required databases and backend services needed for local development and testing. This mode bootstraps a full sandbox environment that allows frontend development and API testing without dependencies on external infrastructure.

Platform interaction commands provide the primary interface for day-to-day development operations. The `unkey deploy` command initiates version creation workflows, handling both Git-based and CLI-based source code submission. The `unkey logs` command enables real-time access to application logs and system events with filtering and search capabilities.

Authentication flow and credential management use root API keys that provide secure, delegated access to platform functionality without requiring developers to share account credentials. Users initiate authentication by running `unkey auth login`, which opens a browser-based login flow that handles OAuth, multi-factor authentication, and other security requirements. Upon successful authentication, the CLI receives an API key that is securely stored locally in a protected file such as `~/.unkey/auth.json` with appropriate file system permissions. All subsequent CLI commands include this key when interacting with [api.unkey.com](http://api.unkey.com), ensuring secure, workspace-scoped access to projects, versions, and configuration.

Composable and scriptable command design ensures that CLI operations can be integrated into existing development workflows and automation systems. Commands follow consistent patterns for argument parsing, output formatting, and error handling. Output can be formatted as human-readable text for interactive use or structured JSON for programmatic consumption. Exit codes follow standard conventions to enable reliable use in shell scripts and CI/CD pipelines.

Environment-based overrides support CI/CD integration by allowing authentication and configuration to be provided through environment variables rather than interactive prompts. This capability enables the CLI to operate in automated environments while maintaining the same functionality and security characteristics as interactive use. Configuration precedence follows a clear hierarchy from command-line flags to environment variables to configuration files, ensuring predictable behavior across different deployment scenarios.

### Domain Management

The domain management system provides both automatic subdomain generation for immediate deployment access and custom domain support for production use cases. Every version receives stable, predictable URLs that enable consistent access patterns while supporting the full range of DNS and certificate management requirements.

Automatic subdomain generation creates stable, predictable URLs for every version without requiring any configuration from developers. Git-based versions typically receive URLs in the form `<commit-hash>-<workspace_slug>.unkey.app`, or `<version_id>-<workspace_slug>.unkey.app` providing permanent access to specific versions that never changes even as new versions are created. Branch-based URLs such as `<branch-name>-<project-name>-<workspace_slug>.unkey.app` provide stable endpoints that always point to the latest version of a particular branch, supporting ongoing development workflows that need consistent URLs for testing and integration.

Wildcard certificate handling for \*.unkey.app ensures that all automatically generated subdomains are immediately available over HTTPS without requiring certificate provisioning delays or manual configuration. The wildcard certificate is issued and renewed automatically through integration with either AWS Certificate Manager or cert-manager depending on the deployment environment. This certificate is distributed to all gateway instances via the gateway database replication system and used dynamically during TLS handshakes using SNI-based certificate selection, ensuring that new versions are immediately accessible over secure connections.

Custom domain configuration and verification enable organizations to use their own domain names for production APIs while maintaining the same security and performance characteristics as platform domains. Users can configure custom domains such as `api.company.com` through the dashboard or API, but domains must be verified for ownership before activation to prevent domain hijacking or unauthorized certificate issuance. The verification process ensures that only legitimate domain owners can associate their domains with platform resources.

Domain ownership verification could use multiple methods to accommodate different customer environments and operational preferences. DNS record verification requires adding a specific verification token to the domain's DNS records, providing strong cryptographic proof of domain control. CNAME challenge verification requires creating a CNAME record pointing to a verification endpoint hosted by the platform. .

Certificate provisioning for custom domains operates automatically once domain ownership is verified. The control plane initiates certificate issuance through Let's Encrypt or AWS Certificate Manager. The resulting certificates are stored securely in the gateway database with appropriate encryption and access controls, ready for immediate use by gateway instances. Automatic renewal workflows ensure that certificates remain valid without requiring manual intervention or service interruption.

## Alternatives Considered

We're evaluating alternative approaches that would have lead to different architectural decisions and trade-offs. These alternatives span isolation technology, build system design, networking approaches, data storage strategies, and other foundational decisions.

### Isolation Technology Choices

The choice of Firecracker microVMs over container-based isolation represents a trade-off between security, performance predictability, and operational complexity. Container-based isolation using Docker on shared Kubernetes clusters would have provided easier operations. However, this approach would have created security concerns around shared kernel space and made it difficult to provide performance isolation guarantees. Containers also impose runtime restrictions and provide minimal control over the execution environment that would have limited the range of supported application patterns.

Traditional virtual machines through services like AWS EC2 were considered but would have introduced significant startup latency and resource overhead that conflicted with our goals of fast iteration and efficient resource utilization. Traditional VMs also require more complex provisioning and management workflows that would have added operational overhead without providing commensurate benefits.

Cloud-Hypervisor is another option that provides hypervisor-level isolation with startup times comparable to containers, enable precise resource allocation and performance isolation, support application patterns without runtime restrictions, and offers GPU support as well as hot migration capabilities.

Firecracker microVMs provide hypervisor-level isolation with startup times comparable to containers, enable precise resource allocation and performance isolation, support application patterns without runtime restrictions, and integrate well with AWS infrastructure while maintaining portability to other cloud providers. This choice supports the security requirements of multi-tenant infrastructure and the performance characteristics needed for production workloads.

### Build System Architecture

The decision to isolate builds in dedicated environments rather than integrating them directly into the control plane infrastructure represents a security and reliability boundary. Integrating builds directly into the control plane would have simplified the system design and reduced the number of moving parts, but would have created security risks by running untrusted user code in the same environment as core orchestration systems.

Using existing CI/CD platforms like [Depot.dev](https://depot.dev), allow us to go to market faster, they also introduce a new dependency and abstraction.

On the other hand a hand rolled solution would require significant engineering effort to build and maintain, and would introduce additional complexity and risk into the system.

### Database Architecture

A monolithic database approach would have simplified the data architecture and reduced the number of systems to operate, but would have created performance bottlenecks and operational challenges as the platform scales. Monolithic databases typically require trade-offs between read and write optimization that would have compromised either control plane responsiveness or data plane performance.

The chosen four-database approach provides separation of concerns. Each database can be optimized for its specific access patterns and performance requirements: the main database for consistency and complex queries, the gateway database for high-throughput reads, the task database for reliable coordination, and ClickHouse for analytics workloads. This separation enables independent scaling and optimization while maintaining boundaries between different types of operations.

Using DynamoDB global tables as the partition database would also be a good idea to explore.

### Other Architectural Decisions

The decision to build a custom task orchestration system rather than adopting existing solutions like Temporal represents a trade-off between functionality and operational overhead. Temporal provides workflow orchestration capabilities with consistency guarantees, but would have introduced additional infrastructure dependencies and operational complexity. The custom task system is minimal and integrated with the platform's specific requirements while providing the reliability and observability needed for infrastructure coordination.

## Open Questions & Risks

Several questions remain unresolved and will require decisions during the implementation process, spanning security, billing, observability, compliance, and scalability considerations. These questions affect both the initial implementation approach and the long-term evolution of the platform.

### Technical Open Questions

#### Networking on AWS

It's not clear to me right now what needs to happen in AWS at the network/permission layer.
Our gateways need to be able to reach any microvm in any region (cross region). We probably want to add private networking capabilities like fly or railway, that allows customer's to build clusters or have multiple services talk with each other without exposing them to the public internet. Addressing individual machines is crucial here.

#### Certificate Management Strategy

Certificate management remains an open decision regarding whether to use AWS Certificate Manager or cert-manager for TLS certificate issuance and renewal. AWS Certificate Manager provides integration with AWS IAM and centralized management capabilities, automatic renewal processes, and integration with other AWS services. However, cert-manager offers issuer flexibility including support for multiple certificate authorities, DNS challenge automation for domain verification scenarios, and native support in Kubernetes-based workflows. The decision must weigh operational complexity, latency considerations for certificate issuance, compliance requirements, multi-cloud strategy implications, and integration complexity with existing gateway infrastructure.

#### Resource Measurement and Billing

Resource measurement and billing require developing systems for tracking CPU time, memory usage, and request volume while maintaining performance and providing predictable pricing for customers. Key questions include determining the level of precision needed for billing purposes while balancing accuracy with system performance, designing user interfaces that present usage metrics through the dashboard and API, implementing real-time usage monitoring that can provide early warnings about unexpected resource consumption, and developing billing models that are fair to customers and sustainable for the business while remaining competitive with existing solutions.

#### Abuse Detection and Mitigation

Abuse detection and mitigation strategies must protect both the platform and customers from various forms of abusive behavior including excessive request volume that could impact system performance, port scanning and other reconnaissance activities that might indicate malicious intent, cryptocurrency mining or other unauthorized compute usage that violates terms of service, and flooding external APIs that could cause platform IP addresses to be blacklisted by third-party services. Developing effective abuse detection requires implementing real-time traffic analysis systems that can identify anomalous patterns without impacting legitimate usage, creating automated response mechanisms that can block suspicious activity at the gateway or network level, establishing notification and alerting systems that keep affected workspaces informed about security events, and implementing graduated response policies that can automatically disable versions or revoke credentials when necessary while providing clear appeals processes for false positives.

#### Partition Coordination Across Multiple AWS Accounts

Partition coordination across multiple AWS accounts becomes increasingly important as the platform expands into enterprise and BYOC scenarios that require infrastructure management across different cloud environments.

#### OIDC Federation for External Service Access

OIDC federation requires designing secure mechanisms for microVMs to make authenticated requests to external services without exposing long-lived credentials. This involves determining how to securely provision temporary credentials or signing keys to microVMs, implementing request signing protocols that are compatible with existing standards, managing credential rotation and revocation for outbound requests, and ensuring that the signing mechanism doesn't become a performance bottleneck for applications that make frequent external API calls.

#### AWS Networking Architecture

AWS networking architecture remains an open question that affects the fundamental implementation of cross-region communication and microVM connectivity. Critical decisions include determining how gateways in different regions can communicate with each other for cross-region request proxying, establishing network connectivity between gateways and Firecracker microVMs running on metal hosts, and designing VPC and subnet architecture that supports these requirements.

The networking design must address several technical challenges including whether to use VPC peering, transit gateways, or other AWS networking services for cross-region gateway communication, how to handle IP address allocation and routing for microVMs within metal host networking, whether gateways and metal hosts should be in the same VPC or separate VPCs with controlled connectivity, and how to maintain security boundaries while enabling the necessary traffic flows for request routing, health checks, and operational management.

Additional considerations include bandwidth costs for cross-region traffic, latency implications of different networking topologies, security group and network ACL configurations for proper traffic isolation, and how the networking architecture scales as we add more regions and partitions.

### Operational Open Questions

#### Regional Expansion Strategy and Prioritization

Regional expansion strategy and prioritization require clarification regarding which geographic regions to support initially and how to prioritize additional regions based on customer demand and regulatory requirements.

#### Autoscaling Algorithms and Policies

Autoscaling algorithms and policies need detailed design, particularly for handling sudden traffic spikes and gradual scaling down during low-traffic periods. The platform must balance responsiveness with cost efficiency while ensuring that scaling decisions don't negatively impact application performance. This includes determining appropriate metrics for scaling decisions, implementing predictive scaling based on historical patterns, managing scaling events across multiple regions, and providing user controls for scaling behavior.

#### Multi-Account Management for Enterprise Deployments

Multi-account management for enterprise deployments requires developing operational procedures and tooling for managing infrastructure across customer-controlled AWS accounts. This includes credential management, billing reconciliation, support escalation procedures, and compliance reporting across multiple organizational boundaries.

#### Performance Requirements and SLAs

Performance requirements and service level agreements need definition to guide architectural decisions and operational procedures. Key questions include establishing target response time percentiles for gateway request processing, defining acceptable throughput levels for individual metal hosts and gateway clusters, setting availability targets for different customer tiers and geographical regions, and determining cold start performance requirements for on-demand microVM provisioning scenarios.

Additional performance considerations include maximum acceptable latency for cross-region request proxying, database query performance requirements for both main and gateway databases, build system performance targets for different project sizes and complexity levels, and acceptable degradation thresholds during peak traffic or partial system failures.

#### Capacity Planning and Infrastructure Scaling

Capacity planning requires developing systematic approaches for predicting and managing infrastructure growth. Critical questions include determining metrics and thresholds for adding new metal hosts to partitions, establishing criteria for deploying additional gateway clusters in existing or new regions, planning database scaling strategies for both transactional and analytical workloads, and designing storage growth management for build artifacts and logs.

#### Cost Model and Resource Tracking

Cost modeling and resource tracking require comprehensive systems for measuring and attributing infrastructure costs to customers while maintaining competitive pricing. Key decisions include determining the granularity of resource measurement for billing purposes, designing fair allocation models for shared infrastructure components like gateways and build systems, establishing pricing tiers that balance simplicity with cost recovery, and implementing usage prediction and quota management systems.

Resource tracking must account for all platform costs including compute time for microVMs and build processes, storage costs for artifacts and logs, network transfer costs for cross-region traffic and CDN usage, and operational overhead for monitoring and management systems. The cost model must also address how to handle burst usage, idle resource allocation, and the economics of maintaining instant rollback capabilities.

#### Operational Procedures and Maintenance

Operational procedures need standardization to ensure consistent platform management across regions and partitions.

Platform update and migration procedures require special attention given the complexity of updating infrastructure while maintaining customer service availability. This includes strategies for updating gateway software across regions, migrating database schemas and data while maintaining service availability and upgrading metal host operating systems and metald software.

#### CPU Time-Based Billing Implementation

CPU time-based billing requires developing accurate measurement and attribution systems for microVM resource consumption. Key technical challenges include determining the granularity of CPU time measurement without impacting performance, implementing fair allocation of shared resources like gateway processing and build system overhead, designing real-time usage tracking that can handle high-frequency updates from thousands of microVMs, and developing aggregation strategies that balance accuracy with storage and processing costs.

The measurement system must account for various CPU usage patterns including burst usage during request processing, idle time between requests, startup and shutdown overhead, and background processes within microVMs. Additionally, the billing system needs to handle scenarios like failed deployments, health check overhead, platform maintenance activities, and cross-region request proxying where CPU time may be consumed in multiple locations for a single customer request.

Technical implementation questions include whether to measure CPU time at the hypervisor level through Firecracker metrics or within the guest OS, how to handle CPU throttling and resource contention fairly across customers, what level of precision is needed for billing accuracy versus system performance, and how to validate billing accuracy and provide transparent usage reporting to customers.

## Appendices

### Appendix B: Database Schema Reference

\[WIP\] The tables below are not exhaustive and just serve as an idea. Please don't read too much into them

#### B.1 Main Database (/unkey)

**workspaces**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| name | varchar(255) | Workspace display name |
| slug | varchar(255) | URL-safe identifier, unique |
| partition_id | varchar(255) | All versions are deployed to this partition |

**partitions**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| name | varchar(255) | Partition display name |
| description | text | Partition description |
| aws_account_id | varchar(255) | Target AWS account |
| region | varchar(255) | Primary AWS region |
| status | enum | active, draining, inactive |
| ip_v4_address | varchar(15) | IPv4 address of the partition |
| ip_v6_address | varchar(39) | IPv6 address of the partition |

**projects**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| workspace_id | varchar(255) | Reference to workspaces |
| partition_id | varchar(255) | Reference to partitions |
| name | varchar(255) | Project display name |
| slug | varchar(255) | URL-safe identifier within workspace |
| git_repository_url | varchar(500) | Git repository URL |

**environments**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| workspace_id | varchar(255) | Reference to workspaces |
| name | varchar(255) | Environment name (production, preview, etc) |
| description | text | Environment description |

**branches**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| workspace_id | varchar(255) | Reference to workspaces |
| project_id | varchar(255) | Reference to projects |
| name | varchar(255) | Git branch name |
| environment_id | varchar(255) | Reference to environments |
| is_production | boolean | Whether this is the production branch |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last modification timestamp |

**rootfs_images**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| workspace_id | varchar(255) | Reference to workspaces |
| project_id | varchar(255) | Reference to projects |
| sha256_hash | varchar(64) | Content hash, unique |
| s3_bucket | varchar(255) | S3 bucket name |
| s3_key | varchar(500) | S3 object key |
| size_bytes | bigint | Image size in bytes |
| created_at | timestamp | Creation timestamp |

**builds**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| workspace_id | varchar(255) | Reference to workspaces |
| project_id | varchar(255) | Reference to projects |
| rootfs_image_id | varchar(255) | Reference to rootfs_images |
| git_commit_sha | varchar(40) | Git commit SHA |
| git_branch | varchar(255) | Git branch name |
| status | enum | pending, running, succeeded, failed, cancelled |
| build_tool | enum | docker, depot, custom |
| error_message | text | Error details for failed builds |
| started_at | timestamp | Build start time |
| completed_at | timestamp | Build completion time |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last modification timestamp |

**versions**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| workspace_id | varchar(255) | Reference to workspaces |
| project_id | varchar(255) | Reference to projects |
| environment_id | varchar(255) | Reference to environments |
| rootfs_image_id | varchar(255) | Reference to rootfs_images |
| config_snapshot | json | Resolved environment variables and config |
| topology_config | json | CPU, memory, autoscaling, region settings |
| git_commit_sha | varchar(40) | Git commit SHA |
| git_branch | varchar(255) | Git branch name |
| status | enum | pending, deploying, active, failed, archived |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last modification timestamp |

**metal_hosts**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| partition_id | varchar(255) | Reference to partitions |
| region | varchar(255) | AWS region |
| availability_zone | varchar(255) | AWS availability zone |
| instance_type | varchar(255) | EC2 instance type |
| ec2_instance_id | varchar(255) | AWS EC2 instance ID, unique |
| private_ip | varchar(45) | Private IP address |
| status | enum | provisioning, active, draining, terminated |
| capacity_cpu_millicores | int | Total CPU capacity |
| capacity_memory_mb | int | Total memory capacity |
| allocated_cpu_millicores | int | Currently allocated CPU |
| allocated_memory_mb | int | Currently allocated memory |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last modification timestamp |

**instances**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| version_id | varchar(255) | Reference to versions |
| metal_host_id | varchar(255) | Reference to metal_hosts |
| region | varchar(255) | AWS region |
| partition_id | varchar(255) | Reference to partitions |
| private_ip | varchar(45) | Instance private IP |
| port | int | Instance port |
| cpu_millicores | int | Allocated CPU |
| memory_mb | int | Allocated memory |
| status | enum | provisioning, starting, running, stopping, stopped, failed |
| health_status | enum | unknown, healthy, unhealthy |
| started_at | timestamp | Instance start time |
| stopped_at | timestamp | Instance stop time |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last modification timestamp |

**hostnames**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| workspace_id | varchar(255) | Reference to workspaces |
| project_id | varchar(255) | Reference to projects |
| environment_id | varchar(255) | Reference to environments |
| pool_id | varchar(255) | Reference to pools |
| hostname | varchar(255) | Domain name, unique |
| is_custom_domain | boolean | Whether this is a custom domain |
| certificate_id | varchar(255) | Reference to TLS certificate |
| verification_status | enum | pending, verified, failed |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last modification timestamp |

#### B.2 Partition Database (/partition)

**gateways**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| partition_id | varchar(255) | Partition identifier |
| region | varchar(255) | AWS region |
| availability_zone | varchar(255) | AWS availability zone |
| private_ip | varchar(45) | Gateway private IP |
| public_ip | varchar(45) | Gateway public IP |
| status | enum | provisioning, active, draining, terminated |
| version | varchar(255) | Gateway software version |
| last_heartbeat | timestamp | Last health check |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last modification timestamp |

**instance_slot**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| version_id | varchar(255) | Reference to versions |
| metal_host_id | varchar(255) | Reference to metal_hosts |
| region | varchar(255) | AWS region |
| partition_id | varchar(255) | Reference to partitions |
| private_ip | varchar(45) | Instance private IP |
| port | int | Instance port |
| cpu | int | Allocated CPU |
| memory_mb | int | Allocated memory |
| status | enum | provisioning, starting, running, stopping, stopped, failed |
| health_status | enum | unknown, healthy, unhealthy |

**routing_entries**

| Column | Type | Description |
| -- | -- | -- |
| hostname | varchar(255) | Domain name, primary key |
| partition_id | varchar(255) | Partition identifier |
| workspace_id | varchar(255) | Workspace identifier |
| project_id | varchar(255) | Project identifier |
| environment_id | varchar(255) | Environment identifier |
| version_id | varchar(255) | Version identifier |
| routing_config | blob | Protobuf serialized routing configuration |
| is_enabled | boolean | Whether routing is active |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last modification timestamp |

**certificates**

| Column | Type | Description |
| -- | -- | -- |
| id | varchar(255) | Primary key |
| hostname | varchar(255) | Domain name, unique |
| certificate_pem | text | X.509 certificate in PEM format |
| private_key_encrypted | blob | Encrypted private key |

**api_keys**

| Column | Type | Description |
| -- | -- | -- |
| key_hash | varchar(64) | SHA256 hash of API key, primary key |
| workspace_id | varchar(255) | Workspace identifier |
| environment_id | varchar(255) | Environment identifier |
| keyspace_id | varchar(255) | Keyspace identifier |
| proto_blob | blob | Protobuf serialized API key configuration |

#### B.3 Task Database

**tasks**

| Column | Type | Description |
| -- | -- | -- |
| id | uuid | Primary key |
| type | varchar(100) | Task type identifier |
| payload | bytea | Protobuf-encoded task data |
| status | enum | pending, claimed, running, completed, failed, dead |
| priority | int | Task priority (higher = more urgent) |
| max_attempts | int | Maximum retry attempts |
| current_attempt | int | Current attempt number |
| created_at | timestamp | Task creation time |
| scheduled_at | timestamp | Earliest execution time |
| claimed_at | timestamp | When task was claimed by worker |
| claimed_by | varchar(255) | Worker identifier |
| lease_expires_at | timestamp | Worker lease expiration |
| completed_at | timestamp | Task completion time |
| error_message | text | Error details for failed tasks |

#### B.4 ClickHouse Observability Database

**build_logs**

| Column | Type | Description |
| -- | -- | -- |
| timestamp | DateTime64 | Log entry timestamp |
| build_id | String | Build identifier |
| workspace_id | String | Workspace identifier |
| project_id | String | Project identifier |
| level | Enum8 | debug, info, warn, error |
| message | String | Log message |
| metadata | String | JSON metadata |

**request_logs**

| Column | Type | Description |
| -- | -- | -- |
| timestamp | DateTime64 | Request timestamp |
| request_id | String | Unique request identifier |
| hostname | String | Request hostname |
| method | String | HTTP method |
| path | String | Request path |
| status_code | UInt16 | HTTP status code |
| response_time_ms | UInt32 | Response time in milliseconds |
| user_agent | String | Client user agent |
| source_ip | String | Client IP address |
| gateway_id | String | Gateway instance identifier |
| pool_id | String | Pool identifier |
| instance_id | String | Target instance identifier |
| workspace_id | String | Workspace identifier |

**system_metrics**

| Column | Type | Description |
| -- | -- | -- |
| timestamp | DateTime64 | Metric timestamp |
| metric_name | String | Metric identifier |
| value | Float64 | Metric value |
| tags | Map(String, String) | Metric labels/tags |
| source | String | Metric source (gateway, metald, etc) |
| pool_id | String | Pool identifier |

### Appendix C: Implementation Phases

The platform development follows a phased approach that balances rapid time-to-market with sustainable architecture and operational readiness. Each phase builds upon previous capabilities while introducing new functionality.

#### Phase 1 (MVP - 4 months - Oct25)

The initial release focuses on core functionality that enables developers to deploy and manage API versions through a simplified workflow. This phase establishes the foundational architecture with single region deployment  while validating core concepts. CLI-based deployment with branch support enables developers to deploy versions using `unkey deploy --branch=main`.

Branch-based preview environments automatically generate unique subdomains for each branch, enabling developers to test and share their work without manual configuration. Gateway-enforced API key authentication provides security capabilities, ensuring that deployed APIs can implement access controls. The CLI-accessible feedback loop enables developers to test their APIs after deployment, validating the workflow from code to running service.

The merge and promote flow allows developers to move versions between environments with visibility and control, supporting both automated and manual promotion strategies. Instant rollback capabilities provide safety through traffic re-routing at the gateway level, enabling sub-second recovery from problematic versions. This foundational release establishes core platform capabilities.

we build in EC2

1. cli builds image and uploads to ghcr
2. ec2 instance downloads OCI
3. ec2 builds rootFS
4. ec2 uploads rootFS to s3

demo flow

1. optionally git clone our demo repo
2. deploy via cli
3. make changes to code (change schema and/or require key auth)
4. deploy child branch
5. show fucking awesome diff
6. show key auth 
7. tell them we got more boring stuff

#### Phase 2 (Private Beta - 3 months additional - Jan26)

The private beta phase introduces automation and multi-region capabilities that transform the platform from a development tool to a production infrastructure platform. GitHub webhook integration enables automatic version creation when code is pushed to tracked repositories, eliminating manual deployment steps and enabling continuous integration workflows.

Multi-region deployment with latency-based traffic routing extends the platform's reach globally while maintaining performance characteristics. This capability enables customers to serve users worldwide with latency and availability guarantees. Custom domain support with TLS certificate management allows customers to use their own domains for production APIs, removing a barrier to production adoption while maintaining the platform's security and operational characteristics.

#### Phase 3 Public Beta (3 months additional - Apr26)

The public beta phase focuses on observability, integration, and developer experience improvements that support broader adoption and use cases. Dashboard visibility into deployment workflows provides web-based interfaces for teams that prefer graphical tools over command-line interfaces.

The documented public API enables programmatic access and integration with existing development tools and workflows, supporting automation scenarios that go beyond the built-in CLI and dashboard capabilities. Documentation and support resources enable new users to onboard and existing users to leverage platform capabilities.

#### Phase 4 General Availability (when it's ready)

General availability introduces the observability and billing capabilities necessary for operating a commercial platform at scale. Observability and monitoring provide both platform operators and customers with the visibility needed to understand system behavior, debug issues, and optimize performance.

Resource metering and billing systems enable sustainable business models while providing customers with predictable costs and usage visibility. These capabilities support both self-service adoption and enterprise sales by providing the transparency and control that customers expect from production infrastructure platforms.

#### Endgame

The endgame phase introduces enterprise capabilities and optimizations that support large-scale, mission-critical deployments. Enterprise features including BYOC pools and per-tenant infrastructure enable the platform to serve customers with compliance and isolation requirements while maintaining operational efficiency.

Autoscaling with predictive algorithms optimize cost and performance by anticipating traffic patterns and scaling proactively rather than reactively. Cache warmup and preload strategies optimize performance for applications with predictable access patterns. Traffic handling and flow observability in the dashboard provide visibility into application behavior and platform performance, enabling debugging and optimization workflows.

### Appendix D: Technology Stack

The control plane uses Go and MySQL/Vitess via PlanetScale provides durable transactional storage. Kubernetes orchestrates control plane services. Protocol Buffers enable serialization and typing across service boundaries, while protobuffers provides type-safe, and extensible schemas and communication. The custom task system provides the coordination capabilities needed without external dependencies.

The data plane leverages Firecracker on AWS EC2 metal instances to provide the isolation, performance, and resource efficiency required for multi-tenant workload execution. Go gateways on k8s deliver low-latency, high-throughput traffic handling needed for production APIs. MySQL/Vitess via PlanetScale in the data plane supports the high-read, low-latency access patterns required for traffic serving decisions.

The public API uses an RPC-based design deployed on AWS Fargate, with plans to migrate to dogfooding once the system reaches maturity and reliability. This approach reduces operational overhead during initial development while providing a path to dogfooding the platform for our own API serving needs.

The build system combines Firecracker isolation for security or Depot.dev integration and Docker toolchain compatibility for ecosystem support. S3 storage provides the durability and availability needed for build artifact management while supporting content-addressed naming and global distribution.

Observability infrastructure uses ClickHouse for high-performance log ingestion and analytics, Prometheus for metrics collection with its pull based model, OpenTelemetry for distributed tracing, and custom dashboards optimized for platform-specific monitoring and debugging needs.

The underlying infrastructure leverages AWS services including Global Accelerator for traffic routing, S3 for object storage, and VPC networking for security and isolation. While we build on top of AWS for now, we need to be careful to avoid vendor lock-in for selfhosting or other cloud providers.

### 
