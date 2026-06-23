# Product Overview

Unkey is an open-source API authentication, authorization, and deployment platform that provides secure, scalable API key management and container orchestration for developers and organizations.

## Core Products

### Unkey Auth
API authentication and authorization platform for managing API keys, permissions, and access control.

**Core Functionality:**
- **API Key Management**: Create, validate, and manage prefixed API keys with fine-grained permissions
- **KeySpaces**: Authentication configuration containers (one per API) that group keys and define validation settings. Previously called "APIs" in the dashboard UI, now renamed to "Keyspaces".
- **Rate Limiting**: Distributed rate limiting using sliding window algorithm with cluster-wide synchronization
- **Authentication & Authorization**: Role-based access control (RBAC) with URN-based resource permissions and identity management
- **Analytics**: Real-time API usage analytics and monitoring via ClickHouse
- **Multi-tenant**: Workspace-based isolation for different organizations/projects
- **Portal**: White-labeled self-service portal for end users to manage their own API keys, scoped by externalId

### Unkey Deploy
Container deployment and orchestration platform for running applications across multiple regions. Note: In product copy, avoid calling it "Unkey Deploy" as a separate product. Deploy is part of Unkey, not a standalone brand.

**Core Functionality:**
- **Project Management**: Organize related applications within workspace projects, gated by deploy plan
- **Multi-Environment**: Deploy apps to production, staging, development environments with isolated configurations
- **Container Orchestration**: Kubernetes-based deployment with StatefulSets managed by Krane
- **Regional Distribution**: Deploy to multiple regions with configurable replica counts per region
- **Git Integration**: Automatic deployments triggered by GitHub webhooks on branch pushes
- **Custom Domains**: Map user-owned domains to deployments with automatic TLS certificate provisioning
- **Deployment Workflows**: Durable workflows for build, deploy, promote, and rollback operations
- **Automatic Builds**: Railpack-based builds without requiring a Dockerfile (detects runtime automatically)
- **Resource Metering**: eBPF-based CPU, memory, egress, and disk tracking per instance via Heimdall

### Billing
Two-product billing model (Auth + Deploy) via Stripe:

- **Auth Plans**: API key verification quotas and feature tiers
- **Deploy Plans**: Starter/Pro/Business tiers with usage-based compute credits (CPU, memory, egress, disk)
- **Usage Metering**: Hourly month-to-date billing push from ClickHouse instance meters to Stripe Billing Meters
- **Credit Grants**: Included usage credits granted on invoice.paid, with prorated top-ups on mid-cycle upgrades
- **Deploy Gate**: Project creation gated by active deploy plan (observe mode with optional enforcement)

## Key Features

- **Distributed Architecture**: Designed for high availability with primary/replica database support
- **Performance**: Built for scale with caching, connection pooling, and optimized data structures
- **Security**: Encryption at rest, secure key storage via S3-compatible vault, audit logging, and MFA/passkeys via WorkOS
- **Observability**: OpenTelemetry integration, Prometheus metrics, and comprehensive logging
- **Developer Experience**: RESTful APIs, comprehensive documentation, and multiple deployment options
- **Durable Workflows**: Restate-powered workflows for exactly-once execution of deployments and infrastructure operations
- **Unified Authentication**: JWT/JWKS-based auth with WorkOS permission translation for dashboard-to-API calls

## Product Relationships

```
Workspace (Organization)
  ├── Auth Resources
  │    ├── KeySpaces (contain Keys, formerly "APIs" in UI)
  │    ├── Keys (with Permissions/Roles)
  │    ├── Identities
  │    └── Rate Limit Namespaces
  ├── Deploy Resources
  │    └── Projects (gated by deploy_plan)
  │         └── Apps
  │              ├── Environments (production, staging, etc.)
  │              │    ├── Deployments (immutable snapshots)
  │              │    ├── Environment Variables
  │              │    ├── Runtime Settings
  │              │    ├── Build Settings (Dockerfile or automatic)
  │              │    └── Regional Settings
  │              ├── Custom Domains
  │              └── Portal Configuration (0:1 per app)
  ├── Billing
  │    ├── Auth Plan (via Stripe subscription)
  │    └── Deploy Plan (starter|pro|business, mirrored to workspaces.deploy_plan)
  └── Legacy Portal Resources
       └── Portal Configuration (1 per keyspace, no app)
```

## Product Decisions

### Deleted Keys Must Remain Visible in Analytics

When an API key is deleted, its verification data in ClickHouse must still be included in analytics charts and tables. Unkey customers rely on usage data to understand how their API was used -- if a deleted key's traffic silently disappears from dashboards, the customer loses visibility into real usage that actually occurred.

- Charts and timeseries should always query ClickHouse without filtering by key existence in MySQL.
- Tables that show per-key breakdowns should include rows for deleted keys with `key_details: null`, displaying "--" for name/identity fields while still showing valid/invalid counts.
- When name or identity filters are active, deleted keys can be excluded (they can't match those filters anyway).
- The guiding principle: analytics reflect what happened, not what currently exists.

### Portal Sessions Scoped to externalId

When a portal session is created with an `externalId`, the `listKeys` endpoint automatically filters results to only show keys associated with that identity. This ensures end users only see their own keys in the portal without requiring explicit key-level filtering.

### Deploy Plan Gating

Project creation is gated by the workspace's `deploy_plan` column (synced from Stripe webhook). The ctrl-api checks entitlement before allowing CreateProject. In observe mode, it logs `deploy_gate.would_block` without blocking; enforcement mode blocks the request. The dashboard mirrors this as a paywall on the projects screen.

### Mid-Cycle Upgrade Credits

When a workspace upgrades their Deploy plan mid-cycle, the prorated fee difference produces an immediate invoice. The `invoice.paid` webhook sums net deploy fee lines and grants credits that expire at the current period end. Downgrades keep the current period's credits and start the lower fee at next renewal.

## Target Use Cases

### Auth Use Cases
- API gateway authentication
- SaaS application API key management
- Microservices authentication
- Developer platform API access control
- Enterprise API security and governance

### Deploy Use Cases
- Multi-region application deployment
- Environment-based deployment workflows (dev -> staging -> production)
- Git-based continuous deployment
- Custom domain hosting with automatic TLS
- Container orchestration for stateful applications
- Automatic builds without Dockerfile configuration
