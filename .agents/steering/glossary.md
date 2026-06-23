# Unkey Glossary

## Core Concepts

**API Key**
A prefixed, cryptographically secure token used for authentication. Format: `{prefix}_{shortToken}_{longToken}` (e.g., `unkey_ABC123_XYZ789`). The long token is hashed using SHA256 for secure storage.

**Workspace**
Top-level organizational unit providing multi-tenant isolation. Each workspace contains APIs, keys, identities, has its own plan and settings, and a `deploy_plan` column synced from Stripe.

**KeySpace (KeyAuth)**
Authentication configuration container for an API, stored in the `key_auth` table. Each API has exactly one keyspace (via `key_auth_id`), and all keys belong to that keyspace. Contains settings like `store_encrypted_keys`, `default_prefix`, `default_bytes`, and tracks approximate size. Keys reference their keyspace via `key_auth_id` field. The dashboard UI renamed "APIs" to "Keyspaces" to better reflect the domain model.

**Identity**
External user or entity that can be associated with keys for tracking and management. Supports external ID mapping. Portal sessions can be scoped to an identity's externalId to restrict key visibility.

## Unkey Deploy Concepts

**Project**
Organizational unit within a workspace for grouping related applications. Contains apps, environments, and deployment configurations. Each project has a unique slug and optional Depot project ID for builds. Project creation is gated by the workspace's deploy plan.

**App**
Containerized application within a project. Represents a single deployable service with its own configuration, environments, and deployment history. Has a default branch for Git-based deployments.

**Environment**
Isolated deployment target within an app (e.g., production, staging, development). Each environment has its own configuration, environment variables, runtime settings, and regional deployments.

**Deployment**
Immutable snapshot of an app at a specific point in time, including container image, environment variables, Git commit info, and runtime configuration. Tracked with status (pending, running, failed) and desired state (running, stopped, deleted).

**Deployment Topology**
Regional distribution configuration for a deployment, specifying desired replica count and status per region. Versioned for edge synchronization.

**Build**
Container image creation process from source code. Can use Docker/Depot (with Dockerfile) or Railpack (automatic, no Dockerfile required). Tracked with build ID and linked to deployments. The `app_build_settings.dockerfile` column being NULL means automatic builds via Railpack.

**Railpack**
Automatic container build system that detects runtime and dependencies without requiring a Dockerfile. Uses a two-phase approach: plan generation (detects how to build the app) and image build (BuildKit frontend consumes the plan). Runs as a git-context-only workflow where repository content never touches the control-plane worker.

**Region**
Geographic deployment location where apps run. Each region has its own cluster and can have different replica counts per deployment.

**Custom Domain**
User-owned domain mapped to an app environment. Requires ownership verification via DNS challenge (TXT or CNAME) before activation.

**Frontline Route**
Routing configuration mapping domains to deployments for TLS termination and request forwarding.

## Authentication & Authorization

**Principal**
Authenticated caller identity in the API. Three types: root key (API key-based), JWT (token from WorkOS or other JWKS provider), and portal session (browser cookie). Defined in `pkg/auth/principal/`.

**Unified Auth Service**
The `pkg/auth` package chains multiple resolvers (root key, JWT/JWKS, portal session) to authenticate API requests. Each resolver attempts to identify the caller; the first success wins, or all fail and return unauthorized.

**JWKS Authentication**
JWT-based authentication using JSON Web Key Sets for key rotation. The API fetches signing keys from a configured JWKS URL and verifies tokens. Supports provider-specific permission translation (e.g., WorkOS permission slugs to Unkey URN permissions).

**URN (Unkey Resource Name)**
Hierarchical identifier for Unkey resources following format: `urn:unkey:{resource_type}:{workspace_id}:{resource_id}`. Used for resource-level permission checks. Defined in `pkg/urn/` with typed builders per resource.

**Resource Permissions**
Fine-grained authorization model where permissions are URN-action pairs. A principal must have the matching URN permission to perform an action on a resource. Defined in `pkg/rbac/permissions/` with typed actions per resource type (keys, keyspaces, projects, ratelimit namespaces, etc.).

**RBAC (Role-Based Access Control)**
Permission system using resource type, resource ID, and action tuples. Supports complex queries with AND/OR operations. Now integrated with URN-based resource permissions.

**Permission**
Fine-grained access control defined as `{resourceType}.{resourceID}.{action}` (legacy format) or as URN-based resource permissions. Can be attached to keys or roles.

**Role**
Named collection of permissions that can be assigned to keys for simplified permission management.

**Root Key**
Special administrative key with elevated permissions for managing workspace resources via API.

**WorkOS Permission Translation**
Mapping from WorkOS permission slugs to Unkey URN permissions. Implemented in `pkg/auth/workos/` and synced via `tools/upsert-workos-permissions`.

## Rate Limiting

**Rate Limit**
Request throttling mechanism using sliding window algorithm with cluster-wide synchronization.

**Ratelimit Namespace**
Logical grouping for rate limit configurations, allowing different limits for different API endpoints or features.

**Ratelimit Override**
Custom rate limit configuration for specific identifiers, overriding namespace defaults.

**Sliding Window Algorithm**
Rate limiting approach that maintains current and previous time window counters, calculating weighted request counts for precise limiting.

**Origin Node**
Designated node (via consistent hashing) that serves as source of truth for a specific rate limit identifier.

**Bucket**
Internal data structure tracking rate limit state for unique identifier+limit+duration combinations.

## Services

**API Service**
Main service handling key verification, management operations, rate limiting, analytics queries, and project CRUD. Uses unified auth service with URN-based resource permissions for authorization. Runs as `unkey api`.

**Frontline**
Multi-tenant ingress server providing TLS termination, request routing, and key authentication policy enforcement via pluggable policy engine.

**Heimdall**
eBPF-based resource metering service that tracks CPU, memory, egress, and disk usage per deployment instance. Feeds instance meters for billing.

**Vault**
Encryption service for secure storage and retrieval of sensitive data using S3-backed storage with envelope encryption and AWS KMS.

**Control Plane (Ctrl)**
Infrastructure management service with two components:
- API: Handles provisioning, build management, orchestration requests via REST/gRPC, deploy billing gates
- Worker: Executes durable Restate workflows for deployments, builds, certificates, routing, and cron jobs

**Krane**
Kubernetes deployment service managing container lifecycles and deployments in K8s clusters. Receives deployment topology updates from control plane and applies them as StatefulSets.

## Data Storage

**MySQL**
Primary relational database with primary/replica support for key metadata, permissions, and configuration.

**Redis**
Distributed cache for rate limiting, key validation caching, and cluster state.

**ClickHouse**
Columnar analytics database storing verification events, rate limit events, sentinel logs, and usage metrics (including instance meters for billing).

**S3-Compatible Storage**
Object storage backend for Vault's encrypted key storage. Uses Garage in development, AWS S3 in production.

**Kafka**
Message queue for distributed cache invalidation and event propagation.

## Key Features

**Credits**
Usage-based quota system for keys. Each verification can deduct credits with configurable cost and automatic refill.

**Refill**
Automatic credit replenishment on configurable intervals (daily, monthly, etc.).

**Remaining**
Current credit balance for a key after deductions.

**Key Migration**
Process for bulk importing or updating keys, tracked with migration ID and error handling.

**Audit Log**
Security and compliance logging tracking all key operations and permission changes. Exported via cron job.

## Billing

**Deploy Plan**
Workspace-level column (`workspaces.deploy_plan`) mirrored from Stripe subscription. Values: starter, pro, business. Synced via `customer.subscription.*` webhook detecting the deploy plan-fee price.

**Deploy Gate**
Project creation gate that checks `deploy_plan` / `deploy_plan_override` before allowing CreateProject. Observable or enforceable via config flag.

**Billing Meter**
Stripe Billing Meter receiving hourly month-to-date usage pushes (CPU, memory, egress, disk) from the deploy billing cron job. Uses `last` aggregation so absolute values overwrite prior.

**Credit Grant**
Stripe credit balance grants issued on `invoice.paid` webhook. Covers included usage per deploy plan tier. Prorated top-ups on mid-cycle upgrades.

**Pricing Tool**
Standalone Go tool (`tools/pricing`) that manages the Stripe billing catalog (plans, meters, prices, products, webhooks) from a typed catalog. Commands: plan/apply/verify/export.

## Analytics & Observability

**Key Verification Event**
Analytics record of key validation attempt, stored in ClickHouse with request ID, timestamp, outcome, latency, and metadata.

**Instance Meter**
ClickHouse query computing per-workspace month-to-date resource usage (CPU-seconds, memory-seconds, egress bytes, disk bytes) from Heimdall checkpoint data.

**Materialized View**
Pre-aggregated analytics data in ClickHouse for fast queries (per-minute, hourly, daily, monthly).

**OpenTelemetry**
Distributed tracing and observability framework integrated across all services.

**Prometheus Metrics**
Time-series metrics collection for monitoring service health and performance. Webhook receiver emits `unkey_webhook_events_total` and `unkey_webhook_handler_duration_seconds`.

## Development & Deployment

**mise**
Task runner and tool version manager used for all development tasks. Replaces Make as the primary interface. Tasks defined in `.mise/tasks/`.

**Bazel**
Build system (v8.5.0) providing reproducible builds, precise change detection, and dependency management.

**Tilt**
Development tool for hot-reloading services in Kubernetes during local development.

**Gazelle**
Bazel tool for automatically generating and maintaining BUILD.bazel files.

**sqlc**
Code generator creating type-safe Go code from SQL queries.

**Protocol Buffers (protobuf)**
Interface definition language for gRPC services, compiled with buf.

**Zen**
Internal HTTP framework (`pkg/zen`) used for building API handlers across Go services. Manages principal context for authenticated requests.

**Restate**
Durable workflow execution framework used by control plane for orchestration. Provides exactly-once semantics, journaling, and virtual object keys for serializing conflicting operations.

**Depot**
Production container build backend (alternative to local Docker builds). Integrated with control plane worker for fast, cached container image builds.

**Virtual Object Key**
Restate concurrency control mechanism that serializes operations on a specific resource (e.g., project_id, deployment_id, domain). Prevents conflicting state mutations.

**Release Tool**
Standalone Go tool (`tools/release`) for service release automation. Generates changelogs, creates tags, and manages semantic versioning per service.

## API Key Components

**Short Token**
First part of API key after prefix, used for quick identification and routing (typically 8 characters).

**Long Token**
Secret portion of API key used for cryptographic verification (typically 24 characters).

**Long Token Hash**
SHA256 hash of long token stored in database for secure validation.

**Prefixed API Key**
Key format with structure: `{prefix}_{shortToken}_{longToken}`, based on Seam API key specification.

## Configuration

**App Environment Variable**
Configuration value scoped to an app and environment. Can be plaintext or secret (encrypted). Supports delete protection for critical variables. Key names must be valid POSIX shell names.

**App Runtime Settings**
Container runtime configuration including CPU/memory limits, port, healthcheck, shutdown signal, and command override. Scoped per app and environment.

**App Build Settings**
Build configuration specifying Dockerfile path and Docker context for container image creation. When dockerfile is NULL, automatic builds via Railpack are used. Scoped per app and environment.

**App Regional Settings**
Region-specific deployment configuration including replica count and optional horizontal autoscaling policy. Scoped per app, environment, and region.

**TOML Config**
Configuration file format used by all services, loaded from file path or inline data.

**Environment Variables**
Configuration via `UNKEY_*` prefixed variables, overridden by CLI flags.

**TLS Config**
Certificate and key configuration for HTTPS, supporting file-based or ACME automatic provisioning.

**ACME**
Automatic Certificate Management Environment protocol for Let's Encrypt TLS certificates using DNS-01 challenges.

## Infrastructure

**Instance ID**
Unique identifier for each service instance, used for distributed coordination and logging.

**Cluster**
Group of service nodes working together with state synchronization and consistent hashing.

**Consistent Hashing**
Algorithm for distributing rate limit identifiers across cluster nodes deterministically.

**Primary/Replica**
Database architecture with write operations on primary and read operations distributed across replicas.

**Health Check**
Endpoint on all services for monitoring availability and readiness.

**Garage**
S3-compatible object storage used in local development (replaced MinIO).

## Testing

**Integration Test**
Test using real dependencies (Docker containers for MySQL, Redis, ClickHouse) with build tag `//go:build integration`.

**Simulation Test**
Property-based testing for stateful systems, verifying invariants across random operation sequences.

**Fuzz Test**
Randomized input testing for parsers and validators to discover edge cases.

**Test Harness**
Reusable test infrastructure providing database setup, cleanup, and helper methods.

## Miscellaneous

**Portal Configuration**
Per-app (or per-keyspace for legacy) configuration for the Customer Portal. Stores workspace association, branding reference, and enabled state. Each app can have at most one portal configuration (enforced by `UNIQUE(app_id)`). Legacy portals without an app are scoped to a single keyspace (`UNIQUE(key_auth_id)`).

**Portal Session Token**
Short-lived (15 min), single-use token created by `portal.createSession`. The customer redirects the end user to the portal with this as a query param. Exchanged once for a long-lived browser session.

**Portal Session**
Long-lived (24 hr) browser session created when a portal session token is exchanged. Stored in MySQL and used to authenticate all subsequent portal API calls via the sessionAuth mechanism. When created with an externalId, scopes key listing to that identity.

**Webhook Receiver**
Framework (`pkg/webhook`) for handling inbound webhooks. Provides method/body-size policing, pluggable signature verification (Stripe, GitHub), event-type routing, middleware, and Prometheus metrics. Handlers return nil (handled), ErrIgnore (acked but not processed), or error (500, provider retries).

**Page Composition Primitives**
`@unkey/ui` components (PageContainer, PageHeader, PageBody, SecondaryNav) for consistent page layouts in the dashboard. PageContainer accepts width="default|full" variant.

**SSoT Routes**
Single-source-of-truth URL constructors in `web/apps/dashboard/lib/navigation/routes/`. Type-safe route builders with guard rail tests that prevent hand-rolled route strings elsewhere in the codebase.

**UID**
Unique identifier generation with typed prefixes (e.g., `key_`, `api_`, `ws_`).

**Fault**
Error handling package providing structured errors with context and stack traces.

**Clock**
Time abstraction interface allowing deterministic testing with frozen or controlled time.

**SWR (Stale-While-Revalidate)**
Caching strategy serving stale data while asynchronously refreshing in background.

**Delete Protection**
Safety flag preventing accidental deletion of critical resources (workspaces, projects, apps, environments).

**App Slug**
URL-friendly identifier for an app, used in routing and domain configuration.

**Default Branch**
Git branch used for automatic deployments when GitHub webhook receives push events.

**Promote**
Deployment operation that makes a deployment the active/live version by updating frontline routes to point to it.

**Rollback**
Deployment operation that reverts to a previous deployment by reassigning frontline routes and setting rollback flag.

**Desired State**
Target state for a deployment (running, stopped, deleted). Control plane worker orchestrates transitions to match desired state.

**Deployment Status**
Current state of a deployment (pending, running, failed, stopped). Updated by worker as deployment progresses through lifecycle.
