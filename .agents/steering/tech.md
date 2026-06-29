# Technology Stack

## Backend (Go)

- **Language**: Go 1.25.1+ with Go workspaces
- **Architecture**: Microservices with CLI-based service management
- **Database**: MySQL with primary/replica support via `go-sql-driver/mysql`
- **Query Generation**: sqlc for type-safe SQL code generation
- **Caching**: Redis for distributed caching and rate limiting
- **Analytics**: ClickHouse for real-time analytics and metrics
- **Message Queue**: Kafka for distributed cache invalidation
- **Storage**: S3-compatible object storage (Garage in dev, AWS S3 in prod) for encrypted key vault
- **Billing**: Stripe API for subscription management, usage metering, and credit grants

## Frontend (TypeScript/Node.js)

- **Runtime**: Node.js 22+
- **Framework**: Next.js 16.1.5 with App Router
- **UI Library**: React 19.2.4
- **Package Manager**: pnpm 11.5.0 with workspaces
- **Build System**: Turbo (monorepo build orchestration)
- **Styling**: Tailwind CSS 4.2.1 with Radix Colors
- **State Management**: TanStack Query 4.36.1 (React Query) + tRPC 10.45.2
- **Forms**: React Hook Form 7.55.0 + Zod 4.3.5
- **Tables**: TanStack Table 8.16.0
- **Charts**: Recharts 3.7.0
- **Linting/Formatting**: Biome
- **Testing**: Vitest 3.2.4
- **TypeScript**: 5.7.3
- **Authentication**: WorkOS (with MFA/passkeys support)

## Infrastructure & Deployment

- **Containerization**: Docker with multi-stage builds
- **Orchestration**: Kubernetes with custom manifests
- **Development**: Tilt for hot reloading, Docker Compose for local services
- **Build System**: Bazel 8.5.0 for Go services and dependencies
- **Container Builds**: Depot for production builds, Railpack for automatic (Dockerfile-less) builds
- **Observability**: OpenTelemetry, Prometheus metrics
- **Protocol Buffers**: buf for gRPC service definitions
- **Task Runner**: mise for all tooling, tasks, and direct tool execution

## Key Libraries & Frameworks

### Go Dependencies
- **HTTP**: `pkg/zen` framework built on standard `net/http`
- **CLI**: `pkg/cli` framework for service commands
- **Database**: `database/sql` with MySQL driver, sqlc for code generation
- **Caching**: `github.com/redis/go-redis/v9`
- **Analytics**: `github.com/ClickHouse/clickhouse-go/v2`
- **Errors**: `pkg/fault` for structured error handling
- **Auth**: `pkg/auth` unified authentication service (root key, JWT/JWKS, portal session resolvers)
- **RBAC**: `pkg/rbac` with URN-based resource permissions (`pkg/urn`)
- **Webhooks**: `pkg/webhook` receiver framework with pluggable signature verifiers
- **TUI**: `pkg/tui` styled terminal output for CLI commands
- **Observability**: OpenTelemetry suite
- **Kubernetes**: `k8s.io/client-go` and controller-runtime
- **Workflows**: Restate for durable workflow execution
- **Testing**: `github.com/stretchr/testify`

### Frontend Dependencies
- **Monorepo**: Turbo 2.4.3+ for build orchestration
- **Code Quality**: Biome 1.9.4+ for formatting and linting
- **Testing**: Vitest for unit and integration tests
- **TypeScript**: 5.7.3+ for type safety

## Common Commands

### Development Setup
```bash
# Bootstrap the development environment (installs tools, deps, generates code)
mise run bootstrap

# Install pinned toolchain
mise install

# Discover available tasks
mise tasks

# Start infrastructure services (MySQL, Redis, ClickHouse, S3, etc.)
mise run dev

# Run local dashboard development
mise run dashboard
```

### Building
```bash
# Build all Go artifacts with Bazel
mise run build

# Build specific target
mise exec -- bazel build //pkg/db:db
```

### Code Generation
```bash
# Generate all code (protobuf, SQL, sqlc, etc.)
mise run generate

# Generate SQL schema from Drizzle definitions
mise run generate-sql
```

### Testing
```bash
# Run tests with Bazel
mise run test

# Run specific package test
mise exec -- bazel test //pkg/cache:cache_test --test_output=errors

# Run fuzz tests
mise run fuzz

# Run TypeScript tests
mise exec -- pnpm --dir=web test
```

### Code Quality
```bash
# Format all code (Go, YAML, TypeScript)
mise run fmt

# Sync Bazel BUILD files after adding new Go files or dependencies
mise run bazel
```

### Releases
```bash
# Tag and release a service
mise run release
```

### Database Operations
```bash
# Generate SQL schema from Drizzle definitions
mise run generate-sql

# Schema is generated from web/internal/db/src/schema
# and output to pkg/mysql/schema/*.sql (per-table files)
```

### Development Environment
```bash
# Start Kubernetes development environment with Tilt
mise run dev

# Start dashboard focused local setup
mise run dashboard

# Stop Tilt and delete minikube cluster
mise run down

# Forward ports for *.unkey.local
mise run tunnel

# Run unkey CLI via Bazel
mise run unkey -- <subcommand> --flags

# Run Stripe billing catalog reconciler
mise run pricing -- plan|apply|verify|export
```

## Development Workflow

1. **Local Development**: Use `mise run dev` for Kubernetes with Tilt or `mise run dashboard` for dashboard-focused setup
2. **Code Generation**: Run `mise run generate` after modifying SQL queries or protobuf definitions
3. **Testing**: Use `mise run test` for comprehensive testing with Bazel
4. **Formatting**: Always run `mise run fmt` before committing
5. **Database Changes**: Modify Drizzle schema in `web/internal/db/src/schema`, then run `mise run generate-sql`
6. **Build Management**: Use `mise run bazel` to sync BUILD.bazel files after adding new dependencies

## Architecture Patterns

- **CLI-based Services**: All services are subcommands of the main `unkey` binary
- **Unified Auth**: `pkg/auth` service chains multiple resolvers (root key, JWT/JWKS, portal session) with principal-based authorization
- **Resource Permissions**: URN-based permission model (`pkg/urn`, `pkg/rbac`) for fine-grained access control
- **Database Layer**: Type-safe SQL with sqlc, primary/replica routing
- **Caching Strategy**: Multi-level caching with Redis and in-memory layers
- **Build System**: Bazel for reproducible builds and dependency management
- **Observability**: Comprehensive tracing, metrics, and structured logging
- **Configuration**: Environment variables with CLI flag overrides
- **Webhook Handling**: `pkg/webhook` framework with signature verification, event-type routing, and Prometheus metrics

## System Architecture

Unkey runs on AWS across multiple regions using Kubernetes for container orchestration. The architecture splits between control plane and data plane services.

### Control Plane
Infrastructure management and orchestration layer:

- **Ctrl API**: REST/gRPC API for deployment operations, domain management, infrastructure provisioning, and deploy billing gates
- **Ctrl Worker**: Restate workflow executor for durable deployments, builds, certificate issuance, routing updates, and cron jobs (billing push, key refill, quota check, audit log export, rate limit cleanup)
- **Krane**: Kubernetes operator managing StatefulSets, deployments, and network policies across clusters
- **Depot**: Production container build backend for fast, cached image builds
- **Railpack**: Automatic container builds without a Dockerfile (plan generation + BuildKit frontend)

### Data Plane
Request handling and data storage layer:

- **Frontline**: Multi-tenant ingress with TLS termination, request routing, and key authentication policy enforcement
- **Heimdall**: eBPF-based resource metering service for instance CPU, memory, egress, and disk tracking
- **API**: Handles key verification, analytics queries, management operations, and project CRUD with URN-based authorization
- **Vault**: Encrypts sensitive data using envelope encryption with AWS KMS and S3 storage
- **ClickHouse**: Stores analytics events for verification logs, rate limit logs, and usage metrics
- **MySQL**: Primary relational database with primary/replica support for all metadata
- **Redis**: Distributed cache for rate limiting and key validation

### Workflow Orchestration

Restate provides durable workflow execution with exactly-once semantics:

- **Virtual Object Keys**: Serialize conflicting operations per domain, project, deployment, workspace, or region
- **Durable Steps**: Isolate side effects with automatic retries and journaling
- **Long-Running Operations**: Handle external rate limits and multi-minute workflows
- **State Persistence**: Resume workflows from last successful checkpoint after failures
- **Cron Jobs**: Unified cron service for periodic tasks (billing push, key refill, quota check, audit log export, rate limit cleanup)

## Bazel Build System

Bazel (v8.5.0) provides accurate change detection and caching for our monorepo:

### Why Bazel
- **Precise Change Detection**: Builds complete dependency graph, only rebuilds affected targets
- **Reliable Caching**: Local and remote caching based on exact inputs
- **Faster CI**: Only runs tests that are actually affected by changes
- **Explicit Dependencies**: BUILD.bazel files serve as living documentation

### Key Commands
```bash
# Sync BUILD files after dependency changes
mise run bazel

# Run affected tests only
mise run test

# Build all artifacts
mise run build

# Build specific target
mise exec -- bazel build //pkg/db:db

# Test specific package
mise exec -- bazel test //pkg/cache:cache_test
```

### Adding New Go Files
- Bazel does NOT auto-discover Go source files like `go build` does
- Every new `.go` file must be explicitly added to the `srcs` list in its directory's `BUILD.bazel`
- Run `mise run bazel` after adding new files to auto-sync BUILD files via Gazelle
- If you skip this, `go build` will succeed but `bazel build` / `mise run build` will fail with "undefined" errors

### Go Linters (nogo)
- Bazel runs `nogo` linters at build time -- these are stricter than `go vet` alone
- **exhaustruct**: All struct fields must be explicitly initialized, even optional/zero-value ones (e.g., `OnFlushError: nil`). Omitting a field that `go build` would zero-initialize will fail the Bazel build.

## Domain Model Overview

### Auth Domain
```
Workspace
  └── KeySpace (formerly "API", 1:1 with KeyAuth)
       └── Keys
            ├── Permissions (via keys_permissions)
            ├── Roles (via keys_roles)
            └── Rate Limits
```

### Deploy Domain
```
Workspace
  └── Project
       └── App
            └── Environment
                 ├── Deployment
                 │    ├── Deployment Topology (per region)
                 │    └── Build (Dockerfile or Railpack automatic)
                 ├── Environment Variables
                 ├── Runtime Settings
                 ├── Build Settings
                 └── Regional Settings (per region)
```

### Cross-Domain
- **Identities**: Can be associated with keys for tracking
- **Audit Logs**: Track operations across both domains
- **Custom Domains**: Link Deploy apps to user-owned domains
- **Certificates**: TLS certificates for custom domains
- **Billing**: Two-product billing (Auth + Deploy) via Stripe with usage metering
