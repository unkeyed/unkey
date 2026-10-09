# AGENTS.md - Unkey agent guide

This file is the first stop for agents working in this repo. Keep changes small,
typed, verified, and routed through `mise`.

## Scoped guidance and additional context

Before changing a file, read this guide and every nested `AGENTS.md` along the
path to that file. Do this explicitly if your harness does not load nested
guidance automatically. For new files, check their parent directories. Repeat
the check when work crosses into another service or package.

Nested guidance applies to its directory and descendants. Keep the root rules
and add the more specific local constraints. Each service under `svc/` has an
`AGENTS.md`; `svc/ctrl/worker/` has additional Restate guidance. Keep shared rules
here and service-specific constraints near the code, without copying them into
parallel instruction files.

The root `contributing/` directory holds testing, code quality,
and screenshot and recording standards. These standards stay in the repository,
not in Notion. Read the relevant guide for your task.

Broader engineering context, design discussions, and operational workflows are in
[Notion's Engineering wiki](https://app.notion.com/p/ed5512d643f38377b0d38164de40681e).
When repository guidance does not answer a task-relevant question, use the
Notion MCP server to search for the service or topic and fetch the relevant
pages. Search results alone are not the full context. Distinguish proposals
and historical plans from accepted decisions, and check implementation claims
against the code.

If Notion MCP is unavailable, the relevant page is inaccessible, or a material
decision is still unclear, ask your operator for the missing context. Do not
invent requirements or silently resolve conflicting guidance. Continue work
that does not depend on the missing information.

## Communication

- Be concise.
- Say what changed and how you verified it.
- If you provide a plan, end with unresolved questions, if any.
- Preserve pre-existing uncommitted changes and unrelated work. Modify existing
  code as needed for the requested task.
- Complete authorized work through verification. Resolve routine implementation
  choices using repository patterns. Ask only when missing information materially
  affects scope or correctness; continue independent work while waiting.

## Source of truth

- Tooling and task runner: `.mise/config.toml`, `.mise/mise.lock`, and
  `.mise/tasks/*`.
- Engineering standards: `contributing/`. These are normative
  standards for writing code, not only reference material for the docs site.
- Product docs: `docs/`.
- Go tooling: `go.mod`, `go.sum`, and `.golangci.yaml`. Rask is pinned as a
  tool in `.mise/config.toml`; it has no config file of its own.
- Web workspace: `web/package.json`, `web/pnpm-workspace.yaml`, and
  `web/pnpm-lock.yaml`.

## Repository map

- `cmd/`: development commands and utilities.
- `build/`: CLI and service entrypoints.
- `svc/`: Go services (`api`, `ctrl`, `frontline`, `heimdall`, `kitchensink`,
  `krane`, `logdrain`, `vault`).
- `pkg/`: shared Go libraries.
- `internal/services/`: shared internal Go services.
- `proto/` and `gen/`: protobuf definitions and generated code.
- `web/`: TypeScript apps (`web/apps/`) and shared packages, database schema,
  and tooling. Shared code lives in `web/internal/`; there is no
  `web/packages/`.
- `docs/`: Mintlify product documentation.
- `contributing/`: Engineering workflow and coding standards.
- `dev/`: local development, Tilt, Kubernetes, and formatting config.

## Tooling rules

Use `mise` for all installs, tasks, and direct tool execution. Makefiles are
legacy and should not be used.

```bash
# Install pinned toolchain
./dev/install-mise
mise install

# Discover tasks
mise tasks
mise run help
```

Prefer `mise run <task>` when a task covers the required scope. Use
`mise exec -- <tool>` when no task applies or a broader task would touch
unrelated files, such as when formatting changed files.

### Common tasks

```bash
mise run build          # lint and Go build, writes ./bin/unkey
mise run lint           # golangci-lint checks
mise run test           # run Go test suite through Rask
mise run fmt            # dprint, go fmt, buf format, pnpm fmt
mise run generate       # SQL, protobuf, Go generators, fmt
mise run generate-bpf   # heimdall eBPF bindings
mise run dev            # local Kubernetes/Tilt dev environment
mise run dashboard      # dashboard-focused local setup
mise run down           # stop minikube, preserve data (exit Tilt first)
mise run tunnel         # port-forward 80/443 for *.unkey.local
mise run unkey -- ...   # run the Unkey CLI
```

### Direct tool examples

```bash
mise exec -- rask ./pkg/cache
mise exec -- pnpm --dir=web test
mise exec -- go test -fuzz=FuzzInRange -fuzztime=30s ./pkg/assert/
```

## Code standards

- Make minimal, surgical changes.
- Preserve type safety. Do not add TypeScript `any`, non-null assertions, or
  unsafe casts.
- Model domain states explicitly. Parse untyped input at boundaries.
- Prefer existing packages, helpers, and patterns before adding new ones.
- Avoid new dependencies unless the local implementation would be worse.
- Keep variable scope small. Use clear names with units or bounds where useful.
- Handle every error. If a state is impossible, assert it rather than ignoring it.
- Add tests for Go behavior and pure TypeScript functions. Do not add React
  component, hook, or render tests unless asked.
- Do not write raw SQL in tests. Use seed and harness helpers, then generated
  sqlc queries from `pkg/db`. If no query exists, add one to
  `pkg/db/queries/` and run `mise run generate`. For ClickHouse, append
  `pkg/clickhouse/schema` structs.
- Do not write raw JSON strings in tests. Encode and decode typed structs,
  preferably existing OpenAPI or handler request and response types.
- Do not hardcode IDs in tests. Generate them with `uid.New(uid.<Entity>Prefix)`
  from `pkg/uid`.
- Exceptions to these rules, such as malformed input or exact-byte tests, are in
  `contributing/quality/testing/anti-patterns.md`.
- Be extremely conservative with code comments. 

## Code comments

- Avoid code comments by default.
- Only add a comment when the code itself cannot clearly communicate why something is necessary.
- Never comment what the code does.
- Comments should be rare and reserved for essential context that a future reader would otherwise be unable to infer, such as non-obvious constraints, intentional trade-offs, or decisions that cannot be expressed through the code itself.
- Do not add comments solely to explain changes to the current reviewer, unless explicitly asked.

## Go conventions

- Build Go through `mise run build` and test Go through Rask with
  `mise run test` or `mise exec -- rask ./path`.
- Use `mise exec -- go test` for targeted fuzzing, which needs Go's fuzz runner.
- Use `github.com/stretchr/testify/require` in tests.
- Use `t.Helper()` in test helpers.
- Use `t.Cleanup()` for resources.
- Prefer `fault` for contextual errors and `assert` for invariants.
- After changing generated inputs, run `mise run generate`.

## TypeScript conventions

- Run pnpm through mise: `mise exec -- pnpm --dir=web ...`.
- Keep package manager changes scoped to `web/` unless a repo task says
  otherwise.
- Do not bypass formatter or type checks by weakening types.
- Use the local app/package patterns in `web/` before introducing abstractions.

## Documentation conventions

- Follow `contributing/quality/documentation.md` for symbol,
  package, and site documentation.
- Product docs live in `docs/` and need `docs/docs.json` nav
  entries when adding pages.
- Use `bash` for shell code blocks.
- Prefer root-relative internal doc links.
- Do not use em dashes in docs.

## Verification

During development, choose the smallest check that proves the change. The build
and pre-push requirements below still apply.

- Go source change: targeted `mise exec -- rask ./path`.
- Go file added or imports changed: `mise run build` (includes lint).
- Shared Go behavior or broad service change: `mise run test` when practical.
- TypeScript change: targeted `mise exec -- pnpm --dir=web ...` command.
- Formatting-sensitive change: format changed files through `mise exec` or a
  scoped task. Use repository-wide `mise run fmt` only when that scope is needed.
- Docs-only change: link/content review. Note if no formatter applies.
- Before pushing: `mise run test`.
- Never install tooling or build an environment to verify a change. If the
  existing checks cannot cover it, say so instead.

After applicable checks pass, repeat or broaden verification only for subsequent
changes, failures, or unresolved risks. Report failed or skipped verification
honestly.

## PlanetScale and Query Insights

Unkey runs on PlanetScale Vitess. Every production MySQL query should carry SQLCommenter tags so Query Insights can attribute load by service, operation, and deploy.

- Go: inject tags through `db.Config.Tags` and `sqlcomment.ForService` in service `run.go`. See [`pkg/mysql/sqlcomment`](pkg/mysql/sqlcomment/doc.go).
- TypeScript: use `createCommentedPool` from `@unkey/db` instead of raw `mysql.createPool`.
- Never put high-cardinality values (user ids, key ids, request ids) in SQL comments.

## High-signal references

Read the page that covers the area before writing code in it. These pages define
how Unkey code is written; this file only summarizes them.

- Local development: `contributing/local/development.md`.
- Build workflow: `contributing/tooling/builds.md`.
- Code quality: `contributing/quality/code-quality.md`.
- Testing: `contributing/quality/testing/index.md`.
- Documentation: `contributing/quality/documentation.md`.

<!-- polylane:start -->
## Investigating production with Polylane

[Polylane](https://polylane.com/?ref=github.onboarding-pr) is an AI production engineer: it watches deploys, telemetry, incidents, and the infrastructure this repository ships to, investigates problems as they happen, and proposes fixes as pull requests. It is connected to this repository and available to coding agents through the [Polylane MCP server](https://mcp.polylane.com/mcp).

- When a question involves production behaviour (an error, a spike, a deploy, a missing signal), query Polylane through its MCP tools before reasoning from the code alone.
- When debugging a failure, start from the incident or issue Polylane recorded: it carries the evidence an investigation already gathered.
- Polylane reviews pull requests in this repository against the live infrastructure. Read its review comment before merging changes that touch production paths.
<!-- polylane:end -->
