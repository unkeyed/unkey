# AGENTS.md - Unkey Development Guide

This document provides essential information for AI coding agents working in this repository.

## Communication

- Be extremely concise; sacrifice grammar for brevity
- At the end of each plan, list unresolved questions (if any)

## Code Quality Standards

- Make minimal, surgical changes
- **Never compromise type safety**: No `any`, no `!` (non-null assertion), no `as Type`
- **Make illegal states unrepresentable**: Model domain with ADTs/discriminated unions; parse inputs at boundaries into typed structures
- Leave the codebase better than you found it

### Entropy

This codebase will outlive you. Every shortcut you take becomes
someone else's burden. Every hack compounds into technical debt
that slows the whole team down.

You are not just writing code. You are shaping the future of this
project. The patterns you establish will be copied. The corners
you cut will be cut again.

**Fight entropy. Leave the codebase better than you found it.**

## Specialized Subagents

- **Oracle**: code review, architecture decisions, debugging, refactor planning
- **Librarian**: understanding 3rd party libs, exploring remote repos, discovering patterns

## Project Overview

Unkey is an open-source API authentication and authorization platform. This is a **polyglot monorepo** containing:

- **Go backend** (root): Services, APIs, and shared libraries built with Bazel
- **TypeScript frontend** (`/web`): Dashboard and API workers built with pnpm/Turborepo

## Build, Lint, and Test Commands

### Go (Root Directory)

```bash
# Install dependencies
make install

# Build all Go artifacts with Bazel
make build                    # or: bazel build //...

# Run all Go tests
make test                     # or: bazel test //...

# Run a single Go test
bazel test //pkg/cache:cache_test --test_filter=TestCacheName
bazel test //pkg/fault:fault_test --test_output=all

# Format and lint Go code
make fmt                      # runs: go fmt, buf format, golangci-lint

# Sync BUILD.bazel files
make bazel                    # runs: bazel mod tidy && bazel run //:gazelle
```

### TypeScript (web/ Directory)

```bash
# Install dependencies
pnpm --dir=web install --frozen-lockfile

# Build all TypeScript packages
pnpm --dir=web build

# Run all TypeScript tests
pnpm --dir=web test

# Run a single test file (from web/apps/api)
cd web/apps/api && pnpm vitest run -c vitest.integration.ts src/routes/v1_keys_createKey.happy.test.ts

# Run tests matching a pattern
cd web/apps/api && pnpm vitest run -c vitest.integration.ts --grep "creates key"

# Format and lint TypeScript
pnpm --dir=web fmt            # runs: biome format && biome check
```

### Development Environment

```bash
make up                       # Start Docker infrastructure (MySQL, Redis, ClickHouse, etc.)
make dev                      # Start full dev environment with Tilt/minikube
make clean                    # Stop and remove all Docker services
```

## Code Style Guidelines

### Go Conventions

**Testing** - Use `testify/require` for assertions:

```go
func TestFeature(t *testing.T) {
    t.Run("scenario", func(t *testing.T) {
        require.NoError(t, err)
        require.Equal(t, expected, actual)
    })
}
```

## Detailed Guidelines

For comprehensive guidance, read these internal docs in `web/apps/engineering/content/docs/contributing/`:

- **Code Style** (`code-style.mdx`): Design philosophy (safety > performance > DX), zero technical debt policy, assertions, error handling with `fault`, scope minimization, failure handling (circuit breakers, retry with backoff, idempotency)
- **Documentation** (`documentation.mdx`): Document the "why" not the "what", use prose over bullets, match depth to complexity, verify behavior before documenting
- **Testing** (`testing/`):
  - `index.mdx` - What to test, test organization, resource cleanup
  - `unit-tests.mdx` - Table-driven tests, naming, parallel execution, test clocks
  - `integration-tests.mdx` - Docker containers, test harness, real dependencies
  - `http-handler-tests.mdx` - API endpoint testing patterns
  - `fuzz-tests.mdx` - Randomized input testing for parsers/validators
  - `simulation-tests.mdx` - Property-based testing for stateful systems
  - `anti-patterns.mdx` - Common mistakes (sleeping, over-mocking, shared state)

## Important Notes

- Always run `make bazel` after adding new Go files
- Use `make fmt` before committing Go changes
- Use `pnpm --dir=web fmt` before committing TypeScript changes
- Integration tests require Docker services running (`make up`)
- The API service deploys to Cloudflare Workers via Wrangler
