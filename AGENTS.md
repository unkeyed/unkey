# AGENTS.md - Unkey Development Guide

This document provides essential information for AI coding agents working in this repository.

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

**Imports** - Organize in groups separated by blank lines:
1. Standard library
2. External/third-party packages
3. Internal packages (`github.com/unkeyed/unkey/internal/...`)
4. Package-level (`github.com/unkeyed/unkey/pkg/...`)
5. Service-level (`github.com/unkeyed/unkey/svc/...`)
6. Generated code (`github.com/unkeyed/unkey/gen/...`)

**Error Handling** - Use the `fault` package for structured errors:
```go
return fault.Wrap(err,
    fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
    fault.Internal("debug message for logs"),
    fault.Public("user-facing message"),
)
```

**Naming Conventions:**
- Files: `snake_case.go`, tests: `*_test.go`
- Exported functions/types: `PascalCase`
- Unexported: `camelCase`
- Receivers: short names `(s *Service)`, `(h *Handler)`
- Constants: `SCREAMING_SNAKE_CASE`

**Testing** - Use `testify/require` for assertions:
```go
func TestFeature(t *testing.T) {
    t.Run("scenario", func(t *testing.T) {
        require.NoError(t, err)
        require.Equal(t, expected, actual)
    })
}
```

### TypeScript Conventions

**Formatting** (enforced by Biome):
- 2 spaces indentation
- 100 character line width
- Use `const` over `let`
- Use template literals over string concatenation
- Use `import type` for type-only imports

**Style Rules:**
- No default exports (except Next.js pages/layouts/configs)
- No `var` declarations
- No `any` types (use `unknown` or proper types)
- No non-null assertions in production code
- Use block statements for all control flow
- Use optional chaining (`?.`) over manual checks

**Testing** - Use Vitest with descriptive assertions:
```typescript
import { describe, expect, test } from "vitest";

test("creates key", async (t) => {
  const h = await IntegrationHarness.init(t);
  const res = await h.post<Request, Response>({ ... });
  expect(res.status).toBe(200);
});
```

## Project Structure

```
/                           # Go root
├── cmd/                    # CLI entrypoints
├── svc/                    # Backend services (api, ctrl, vault, etc.)
├── pkg/                    # Shared Go libraries
├── proto/                  # Protocol buffer definitions
├── gen/                    # Generated code (proto, sqlc)
└── web/                    # TypeScript monorepo
    ├── apps/
    │   ├── api/            # Cloudflare Workers API (Hono)
    │   ├── dashboard/      # Next.js dashboard
    │   └── docs/           # Documentation site
    └── internal/           # Shared TS packages (db, ui, rbac, etc.)
```

## Key Technologies

**Go Backend:**
- Bazel for builds, Gazelle for BUILD file generation
- `pkg/zen` - Custom HTTP framework
- `pkg/fault` - Structured error handling
- `pkg/codes` - Error code URNs
- golangci-lint v2 for linting

**TypeScript Frontend:**
- pnpm workspaces + Turborepo
- Next.js 14 (dashboard)
- Hono (API workers)
- Drizzle ORM (database)
- Biome for formatting/linting
- Vitest for testing

## Common Patterns

### Go HTTP Handlers (zen framework)
```go
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
    auth, emit, err := h.Keys.GetRootKey(ctx, s)
    defer emit()
    if err != nil { return err }

    req, err := zen.BindBody[Request](s)
    if err != nil { return err }

    // Business logic...

    return s.JSON(http.StatusOK, Response{...})
}
```

### TypeScript API Tests
```typescript
const h = await IntegrationHarness.init(t);
const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
const res = await h.post<Req, Res>({
  url: "/v1/keys.createKey",
  headers: { Authorization: `Bearer ${root.key}` },
  body: { ... },
});
expect(res.status).toBe(200);
```

## Linting Configuration

**Go** (`.golangci.yaml`): Strict linting including exhaustive switch/map checks, struct initialization (`exhaustruct`), and security checks (`gosec`).

**TypeScript** (`web/biome.json`): Enforces no unused variables/imports, strict equality, proper React hooks usage, and consistent code style.

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
