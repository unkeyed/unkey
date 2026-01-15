# AGENTS.md - Unkey Development Guide

Polyglot monorepo: Go backend (root, Bazel) + TypeScript frontend (`/web`, pnpm/Turborepo).

## Commands

**Go:** `make build` (build), `make test` (test all), `make fmt` (format/lint), `make bazel` (sync BUILD files)
- Single test: `bazel test //pkg/cache:cache_test --test_filter=TestCacheName`

**TypeScript:** `pnpm --dir=web build`, `pnpm --dir=web test`, `pnpm --dir=web fmt`
- Single test: `cd web/apps/api && pnpm vitest run -c vitest.integration.ts src/routes/file.test.ts`

**Dev:** `make up` (Docker infra), `make dev` (full dev env)

## Structure

`cmd/` CLI | `svc/` services | `pkg/` shared libs | `proto/` protobufs | `gen/` generated code
`web/apps/api` Hono workers | `web/apps/dashboard` Next.js | `web/internal/` shared TS packages

## Code Style

**Go:** Use `fault` package for errors, `testify/require` for tests, run `make bazel` after adding files
**TypeScript:** Biome formatting, no `any`, use `import type`, Vitest for tests
