# Project Structure

## Repository Layout

This is a monorepo containing both Go backend services and TypeScript frontend applications.

```
unkey/
├── cmd/                    # Service entry points (CLI subcommands)
│   ├── api/                # Main API server for auth operations
│   ├── auth/               # Authentication service
│   ├── deploy/             # Deployment orchestration
│   ├── dev/                # Development utilities (seed, GitHub, stripe test clocks)
│   ├── healthcheck/        # Health check utility
│   └── version/            # Version info
├── pkg/                    # Reusable Go packages and libraries
│   ├── auth/               # Unified authentication service
│   │   ├── jwt/            # JWT/JWKS resolver with key rotation
│   │   ├── principal/      # Principal types (root key, JWT, portal session)
│   │   ├── portal_session/ # Portal session resolver
│   │   ├── root_key/       # Root key resolver
│   │   └── workos/         # WorkOS permission translation
│   ├── db/                 # Database layer with sqlc-generated queries
│   ├── mysql/              # MySQL schema (generated from Drizzle)
│   ├── cache/              # Generic caching (TTL, LRU, SWR)
│   ├── clickhouse/         # Analytics database client
│   ├── rbac/               # Role-based access control with URN permissions
│   │   └── permissions/    # Typed resource permission definitions
│   ├── urn/                # Unkey Resource Name parser and builders
│   ├── webhook/            # Webhook receiver framework
│   │   └── verifiers/      # Signature verifiers (stripe, github)
│   ├── tui/                # Styled terminal output for CLI commands
│   ├── billingperiod/      # Billing period utilities
│   ├── prefixedapikey/     # API key format and validation
│   ├── fault/              # Structured error handling
│   ├── zen/                # HTTP framework
│   ├── cli/                # CLI framework
│   ├── config/             # Configuration loading (TOML)
│   ├── runner/             # Service lifecycle management
│   ├── cluster/            # Cluster coordination
│   ├── restate/            # Restate workflow client
│   ├── encryption/         # AES encryption utilities
│   ├── rpc/                # RPC client utilities
│   └── ...                 # Other shared utilities (50+ packages)
├── svc/                    # Service implementations
│   ├── api/                # API service logic
│   │   ├── routes/         # HTTP route handlers (v2_* convention)
│   │   ├── openapi/        # OpenAPI spec (split YAML + generated)
│   │   └── internal/       # API-internal packages (middleware, errors, ctrlclient)
│   ├── ctrl/               # Control plane services
│   │   ├── api/            # Control API handlers
│   │   ├── worker/         # Restate workflow implementations
│   │   │   ├── cron/       # Unified cron service (billing, refill, quota, audit, cleanup)
│   │   │   └── deploy/     # Deploy workflows (build, railpack, scale)
│   │   └── services/       # Shared control plane logic
│   ├── demo/               # Demo API service
│   ├── frontline/          # Frontline service logic
│   ├── heimdall/           # eBPF resource metering service
│   ├── kitchensink/        # Testing/development service
│   ├── krane/              # Krane controller logic
│   └── vault/              # Vault encryption logic
├── internal/               # Internal service packages
│   └── services/           # Shared service implementations
│       ├── analytics/      # Analytics service
│       ├── auditlogs/      # Audit logging
│       ├── caches/         # Cache implementations
│       ├── keys/           # Key management and verification
│       ├── ratelimit/      # Rate limiting service
│       └── usagelimiter/   # Usage limiting
├── tools/                  # Standalone Go tools
│   ├── pricing/            # Stripe billing catalog reconciler
│   ├── release/            # Service release automation
│   ├── upsert-workos-permissions/ # WorkOS permission sync tool
│   ├── generate-rpc-clients/     # RPC client code generator
│   └── exportoneof/        # Protobuf oneof export tool
├── web/                    # Frontend monorepo (pnpm workspaces)
│   ├── apps/               # Frontend applications
│   │   ├── api/            # API worker (Cloudflare Workers)
│   │   ├── dashboard/      # Main dashboard app (Next.js)
│   │   ├── design/         # Design system documentation site (Astro)
│   │   ├── docs/           # Documentation site
│   │   ├── planetfall/     # Planetfall app
│   │   ├── portal/         # Portal app (TanStack Start)
│   │   └── workflows/      # Workflows app
│   ├── internal/           # Shared internal packages
│   │   ├── ui/             # @unkey/ui component library
│   │   ├── icons/          # @unkey/icons package
│   │   ├── db/             # Database schema (Drizzle)
│   │   ├── clickhouse/     # ClickHouse client
│   │   ├── schema/         # Shared schemas
│   │   ├── billing/        # Billing utilities
│   │   ├── rbac/           # RBAC utilities and permission definitions
│   │   ├── encryption/     # Encryption utilities
│   │   ├── error/          # Error handling
│   │   ├── events/         # Event definitions
│   │   ├── hash/           # Hashing utilities
│   │   ├── id/             # ID generation
│   │   ├── keys/           # Key utilities
│   │   ├── match/          # Pattern matching
│   │   ├── resend/         # Email via Resend
│   │   ├── validation/     # Validation utilities
│   │   ├── vault/          # Vault client
│   │   ├── vercel/         # Vercel integration
│   │   ├── checkly/        # Checkly monitoring
│   │   ├── encoding/       # Encoding utilities
│   │   └── tsconfig/       # Shared TypeScript config
│   └── tools/              # Build tools and utilities
├── proto/                  # Protocol buffer definitions
│   ├── cache/v1/           # Cache invalidation protocol
│   └── cluster/v1/         # Cluster envelope protocol
├── gen/                    # Generated code
│   ├── proto/              # Generated protobuf (cache, cluster, ctrl, hydra, sentinel, vault)
│   └── rpc/                # Generated RPC clients (ctrl, vault)
├── dev/                    # Development environment configs
│   ├── config/             # Service TOML configs
│   ├── k8s/                # Kubernetes manifests
│   └── docker-compose.yaml # Local infrastructure
├── docs/                   # Documentation
│   ├── engineering/        # Engineering docs
│   └── product/            # Product docs
├── BUILD.bazel             # Root Bazel build configuration
├── MODULE.bazel            # Bazel module definition
└── main.go                 # Main CLI entry point
```

## Go Code Organization

### Service Architecture
- **`main.go`**: Single binary with multiple subcommands for different services
- **`cmd/`**: Each service has its own directory with CLI configuration and flags
- **`svc/`**: Service implementation packages (business logic)
- **`pkg/`**: Shared libraries and utilities used across services
- **`tools/`**: Standalone Go tools (not part of the main binary)

### CLI Subcommands (main.go)
- **`api`**: API key management and verification service
- **`auth`**: Authentication service
- **`deploy`**: Deployment orchestration
- **`dev`**: Development utilities (seed, GitHub integration, stripe test clocks)
- **`healthcheck`**: Health check utility
- **`version`**: Version information

Note: Long-running services (api, ctrl, frontline, heimdall, krane, vault) are deployed as individual container images, each running its own service entrypoint. The `cmd/run/` subcommand pattern was removed in favor of per-service images.

### Key Services (svc/)
- **`api/`**: Main API server for key validation, management, project CRUD, and rate limiting. Uses URN-based resource permissions for authorization.
- **`ctrl/`**: Control plane service for cluster management and deployment orchestration
  - `api`: REST/gRPC API for deployment operations and deploy billing gates
  - `worker`: Restate workflow executor for durable deployments, cron jobs, and billing
- **`demo/`**: Demo API service for testing and examples
- **`frontline/`**: Edge service for high-performance key validation and multi-tenant TLS termination with policy-based key authentication
- **`heimdall/`**: eBPF-based resource metering service tracking CPU, memory, egress, and disk per deployment instance
- **`krane/`**: Kubernetes operator for deployment management and StatefulSet orchestration
- **`vault/`**: Encryption service for secure key storage with S3 backend

### Package Organization (pkg/)

50+ packages organized by concern:

#### Core Infrastructure
- **`db/`**: Database layer with sqlc-generated type-safe queries
- **`mysql/`**: MySQL schema files (generated from Drizzle definitions)
- **`cache/`**: Generic caching with TTL, LRU, and SWR support
- **`clickhouse/`**: ClickHouse client and analytics queries
- **`encryption/`**: AES encryption utilities

#### Authentication & Authorization
- **`auth/`**: Unified auth service with chained resolvers (root key, JWT/JWKS, portal session)
- **`auth/jwt/`**: JWT verification with JWKS key rotation and provider-specific (WorkOS) permission translation
- **`auth/principal/`**: Principal types representing authenticated callers
- **`auth/workos/`**: WorkOS permission slug to Unkey permission translation
- **`rbac/`**: Role-based access control with URN permission queries
- **`rbac/permissions/`**: Typed permission definitions per resource (keys, keyspaces, projects, etc.)
- **`urn/`**: Unkey Resource Name parser, builders, and resource type definitions

#### Service Framework
- **`cli/`**: CLI framework for service commands (with enum flags, help generation)
- **`config/`**: TOML configuration loading
- **`runner/`**: Service lifecycle management (start, stop, health)
- **`zen/`**: HTTP framework for API handlers
- **`rpc/`**: RPC client utilities
- **`restate/`**: Restate workflow client integration
- **`webhook/`**: Webhook receiver framework (signature verification, event routing, metrics)

#### Networking & Distributed Systems
- **`cluster/`**: Cluster coordination and state sync
- **`dns/`**: DNS resolution utilities
- **`tls/`**: TLS configuration and certificate management
- **`healthcheck/`**: Health check endpoint utilities

#### Utilities & Helpers
- **`array/`**: Array manipulation utilities
- **`assert/`**: Testing assertions and validation helpers
- **`batch/`**: Batch processing utilities
- **`billingperiod/`**: Billing period calculation utilities
- **`circuitbreaker/`**: Circuit breaker pattern
- **`clock/`**: Time abstraction for testing and caching
- **`codes/`**: Error/status codes
- **`conc/`**: Concurrency utilities
- **`dockertest/`**: Docker-based test helpers
- **`fault/`**: Structured error handling with context and stack traces
- **`fuzz/`**: Fuzz testing utilities
- **`hash/`**: Hashing utilities
- **`jwt/`**: JWT token handling
- **`logger/`**: Structured logging (with colored pretty handler for TTY)
- **`retry/`**: Retry logic with exponential backoff
- **`sim/`**: Simulation testing framework
- **`singleflight/`**: Request deduplication
- **`testutil/`**: Test utilities and helpers
- **`tui/`**: Styled terminal output (colors, tables, KV blocks, TTY-aware)
- **`uid/`**: Unique identifier generation
- **`validation/`**: Input validation (including POSIX env var key validation)
- **`version/`**: Version information

#### Business Logic
- **`auditlog/`**: Audit logging for security and compliance
- **`prefixedapikey/`**: Prefixed API key format (`prefix_shortToken_longToken`) with SHA256 hashing

#### Observability
- **`otel/`**: OpenTelemetry configuration and helpers
- **`prometheus/`**: Prometheus metrics collection

## Frontend Structure (web/)

```
web/
├── apps/                   # Frontend applications
│   ├── api/                # API worker (Cloudflare Workers)
│   ├── dashboard/          # Main dashboard (Next.js)
│   ├── design/             # Design system docs (Astro)
│   ├── docs/               # Documentation site
│   ├── planetfall/         # Planetfall app
│   ├── portal/             # Portal app (TanStack Start)
│   └── workflows/          # Workflows app
├── internal/               # Shared internal packages
├── tools/                  # Build tools and utilities
├── pnpm-workspace.yaml     # Workspace config
├── turbo.json              # Turbo build config
├── biome.json              # Biome linting/formatting config
└── package.json            # Root workspace configuration
```

## Code Generation

### Database (sqlc)
- **Source**: `pkg/db/queries/*.sql` - Raw SQL queries
- **Config**: `pkg/db/sqlc.json` - sqlc configuration
- **Generated**: `pkg/db/*_generated.go` - Type-safe Go code
- **Schema**: `pkg/mysql/schema/*.sql` - Per-table SQL files generated from Drizzle definitions in `web/internal/db/src/schema`
- **Models**: Core domain (Workspace, Project, App, Environment, Deployment), Auth (Key, Permission, Role), Deploy (DeploymentTopology, AppSettings)

### Protocol Buffers
- **Source**: `proto/` - .proto definitions (cache/v1, cluster/v1)
- **Service protos**: `svc/ctrl/proto/` - Control plane service definitions (hydra/v1, ctrl/v1)
- **Config**: `proto/buf.gen.yaml` - buf configuration
- **Generated**: `gen/proto/` - Go bindings (cache, cluster, ctrl, hydra, sentinel, vault)
- **Generated RPC**: `gen/rpc/` - RPC clients (ctrl, vault)
- **Services**: Control plane (deployment, environment, custom_domain, acme, cluster, project, cron), Vault, Cache invalidation

### OpenAPI (API service)
- **Source**: `svc/api/openapi/spec/` - Split YAML spec files per endpoint
- **Generated**: `svc/api/openapi/openapi-generated.yaml` - Merged spec
- **Generated Go**: `svc/api/openapi/gen.go` - Request/response types

## Configuration Patterns

### Environment Variables
- All services use consistent `UNKEY_*` prefixed environment variables
- CLI flags override environment variables
- Configuration validation at startup

### Service Discovery
- Kubernetes-native service discovery
- Health check endpoints on all services
- Graceful shutdown handling

## Documentation Standards

### Package Documentation
- Every package MUST have a `doc.go` file with comprehensive documentation
- Follow the patterns established in `GO_DOCUMENTATION_GUIDELINES.md`
- Include usage examples and cross-references

### Code Organization Rules
- **Exported functions**: Must be documented with clear purpose and usage
- **Internal functions**: Focus on "why" rather than "what"
- **Complex algorithms**: Include reasoning for design decisions
- **Error handling**: Document specific error conditions when relevant

## Development Patterns

### Testing Structure
- Unit tests alongside source code (`*_test.go`)
- Integration tests use build tags (`//go:build integration`)
- Test utilities in `pkg/testutil/` and `pkg/dockertest/`
- Fuzz tests in `pkg/fuzz/` and individual packages
- Simulation tests via `pkg/sim/`
- Benchmarks for performance-critical code

### Dependency Management
- Go workspaces for multi-module development
- Minimal external dependencies
- Prefer standard library when possible
- Pin specific versions for reproducible builds

### Build and Deployment
- Individual Docker images per service (api, ctrl, frontline, heimdall, krane, vault)
- Kubernetes manifests in `dev/k8s/manifests/`
- Tilt configuration for development hot-reloading
- Bazel for reproducible builds and dependency management
- mise tasks for common operations
- Release tool (`tools/release`) for service tagging and changelog generation

## Naming Conventions

### Go Packages
- Short, descriptive names (e.g., `cache`, `db`, `vault`)
- No underscores or mixed case
- Avoid stuttering (e.g., `cache.Cache` not `cache.CacheManager`)

### Files and Directories
- Use kebab-case for multi-word directory names
- Go files use snake_case for generated files (`*_generated.go`)
- Test files end with `_test.go`
- Documentation files are `doc.go`

### Database and API
- Database tables use snake_case
- API endpoints use kebab-case in URLs
- JSON fields use camelCase
- Environment variables use SCREAMING_SNAKE_CASE with `UNKEY_` prefix

### Dashboard Routes
- Type-safe URL constructors in `lib/navigation/routes/` (SSoT pattern)
- Guard rail tests prevent hand-rolled route strings outside the routes module
