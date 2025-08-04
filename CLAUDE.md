# CLAUDE.md

  

  

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

  

  

## Commands

  

  

### Development Setup

  

```bash

  

# Install dependencies

  

pnpm install

  

  

# Start local development environment (includes databases)

  

make up

  

  

# Run development servers

  

pnpm dev

  

```

  

  

### Build and Testing

  

```bash

  

# Build all packages

  

pnpm build

  

  

# Format and lint code

  

pnpm fmt

  

  

# Run tests (with concurrency control)

  

pnpm test

  

  

# Run integration tests (requires local environment)

  

make integration

  

  

# Type checking

  

pnpm typecheck # Check individual app package.json for specific commands

  

```

  

  

### Database Operations

  

```bash

  

# Run database migrations

  

pnpm migrate

  

  

# Generate database schema changes

  

make generate-sql

  

  

# ClickHouse migrations

  

make migrate-clickhouse

  

make migrate-clickhouse-reset # Reset ClickHouse schema

  

```

  

  

### Go-Specific Commands

  

```bash

  

# Go services (in /go directory)

  

go test ./...

  

go build ./cmd/...

  

  

# Deploy services (in /go/deploy directory)

  

make build # Test binary builds

  

make install # Build and install with systemd units

  

```

  

  

## Architecture Overview

  

  

Unkey is a monorepo containing both TypeScript/Node.js and Go services for API key management, authentication, and distributed rate limiting.

  

  

### Core Applications

  

  

**Dashboard** (`apps/dashboard/`)

  

- Next.js web interface for API key management

  

- Built with React, TailwindCSS, and tRPC

  

- Authentication via WorkOS

  

  

**API** (`apps/api/`)

  

- Cloudflare Workers-based API for key verification

  

- Uses Hono framework with OpenAPI validation

  

- Handles key CRUD operations and rate limiting

  

  

**Agent** (`apps/agent/`)

  

- Go-based distributed rate limiting service

  

- Uses Serf for clustering and gossip protocol

  

- Implements sliding window rate limiting algorithm

  

  

**Go Services** (`go/`)

  

- **API**: Main HTTP API server (port 7070)

  

- **Ctrl**: Control plane for infrastructure management (port 8080)

  

- **Deploy services**: VM lifecycle management (metald, builderd, etc.)

  

  

### Database Architecture

  

  

**MySQL/PlanetScale**: Primary relational data

  

- Tables: workspaces, apis, keys, permissions, roles, identities

  

- Drizzle ORM with type safety

  

- Read replica support

  

  

**ClickHouse**: Analytics and time-series data

  

- Verification metrics and rate limiting statistics

  

- Schema in `internal/clickhouse/schema/`

  

  

**Redis/Upstash**: Distributed caching and rate limiting state

  

- Multi-tier caching strategy

  

- Real-time rate limit counters

  

  

### Shared Packages (`internal/`)

  

  

Key packages for cross-app functionality:

  

- `@unkey/db`: Database schemas and connections

  

- `@unkey/cache`: Multi-tier caching implementation

  

- `@unkey/encryption`: AES-GCM encryption utilities

  

- `@unkey/keys`: Key generation and validation

  

- `@unkey/ui`: Shared React components

  

- `@unkey/validation`: Zod schema definitions

  

  

## Development Guidelines

  

  

### Code Standards

  

  

**TypeScript/JavaScript**:

  

- Use Biome for formatting and linting

  

- Prefer named exports over default exports (except Next.js pages)

  

- Follow strict TypeScript configuration

  

- Use Zod for runtime validation

  

  

**Go**:

  

- Follow comprehensive documentation guidelines (see `go/GO_DOCUMENTATION_GUIDELINES.md`)

  

- Every public function/type must be documented

  

- Use `doc.go` files for package documentation

  

- Prefer interfaces for testability

  

  

### Testing Patterns

  

  

**TypeScript**:

  

- Vitest for unit and integration tests

  

- Separate configs: `vitest.unit.ts`, `vitest.integration.ts`

  

- Integration harness for API testing

  

  

**Go**:

  

- Table-driven tests

  

- Integration tests with real dependencies

  

- Test organization by HTTP status codes

  

  

### Environment Variables

  

  

All environment variables must follow the format: `UNKEY_<SERVICE_NAME>_VARNAME`

  

  

### Key Patterns

  

  

**Authentication/Authorization**:

  

- Root keys for API access with granular permissions

  

- Workspace-based multi-tenancy

  

- RBAC with role inheritance

  

  

**Rate Limiting**:

  

- Distributed consensus for accuracy

  

- Sliding window algorithm

  

- Override capabilities for specific identifiers

  

  

**Error Handling**:

  

- Consistent error types with proper HTTP status codes

  

- Structured error responses following OpenAPI spec

  

- Circuit breaker patterns for external dependencies

  

  

**Caching**:

  

- Multi-tier strategy (Memory → Redis → Database)

  

- Stale-while-revalidate pattern

  

- Namespace-based cache invalidation

  

  

## Important Files

  

  

- `turbo.json`: Monorepo build configuration

  

- `biome.json`: Code formatting and linting rules

  

- `package.json`: Root package with workspace scripts

  

- `vitest.workspace.json`: Test workspace configuration

  

- `go/GO_DOCUMENTATION_GUIDELINES.md`: Go code documentation standards

  

- `go/deploy/CLAUDE.md`: Additional rules for deploy services

  

  

## Development Tips

  

  

1. **Database Changes**: Use Drizzle migrations, not manual SQL

  

2. **Testing**: Run integration tests locally with `make integration`

  

3. **Go Services**: Use `AIDEV-*` comments for complex/important code

  

4. **Performance**: Prioritize reliability over performance

  

5. **Security**: Never commit secrets or expose sensitive data in logs

6. **Build**: run the linter and pnpm build after all TODOs