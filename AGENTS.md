# AGENTS.md - Unkey agent guide

This file is the first stop for agents working in this repo. Keep changes small,
typed, verified, and routed through `mise`.

## Communication

- Be concise.
- Say what changed and how you verified it.
- If you provide a plan, end with unresolved questions, if any.
- Do not revert or rewrite work you did not make unless explicitly asked.

## Source of truth

- Tooling and task runner: `mise.toml`, `mise.lock`, and `.mise/tasks/*`.
- Engineering docs: `docs/engineering/contributing/`.
- Product docs: `docs/product/`.
- Go build graph: `BUILD.bazel`, `MODULE.bazel`, and Gazelle.
- Web workspace: `web/package.json`, `web/pnpm-workspace.yaml`, and
  `web/pnpm-lock.yaml`.

## Repository map

- `cmd/`: Unkey CLI commands and service entrypoints.
- `svc/`: Go services (`api`, `ctrl`, `frontline`, `heimdall`, `krane`, `vault`).
- `pkg/`: shared Go libraries.
- `internal/`: shared internal Go services.
- `proto/` and `gen/`: protobuf definitions and generated code.
- `web/`: TypeScript apps, packages, database schema, and tooling.
- `docs/`: Mintlify product and engineering documentation.
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

Prefer `mise run <task>` when a task exists. Use `mise exec -- <tool>` only for
direct commands without a task.

### Common tasks

```bash
mise run build          # Bazel build, copies ./bin/unkey
mise run test           # run Bazel test suite
mise run fmt            # dprint, go fmt, buf format, pnpm fmt
mise run bazel          # bazel mod tidy and Gazelle
mise run generate       # SQL, protobuf, Go generators, Gazelle, fmt
mise run generate-bpf   # heimdall eBPF bindings
mise run dev            # local Kubernetes/Tilt dev environment
mise run dashboard      # dashboard-focused local setup
mise run down           # stop Tilt and delete minikube cluster
mise run tunnel         # port-forward 80/443 for *.unkey.local
mise run unkey -- ...   # run the Unkey CLI through Bazel
```

### Direct tool examples

```bash
mise exec -- bazel test //pkg/cache:cache_test --test_output=errors
mise exec -- bazel test //pkg/cache:cache_test --test_filter=TestCacheName
mise exec -- pnpm --dir=web test
mise exec -- pnpm --dir=web/apps/api vitest run -c vitest.integration.ts
mise exec -- go test -fuzz=FuzzParseConfig -fuzztime=30s ./pkg/config/
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
- Document why non-obvious code exists, not what each line does.

## Go conventions

- Build and test Go through Bazel, not raw `go test`, except fuzzing through
  `mise exec -- go test -fuzz ...`.
- Use `github.com/stretchr/testify/require` in tests.
- Use `t.Helper()` in test helpers.
- Use `t.Cleanup()` for resources.
- Prefer `fault` for contextual errors and `assert` for invariants.
- After adding or moving Go files, run `mise run bazel`.
- After changing generated inputs, run `mise run generate`.

## TypeScript conventions

- Run pnpm through mise: `mise exec -- pnpm --dir=web ...`.
- Keep package manager changes scoped to `web/` unless a repo task says
  otherwise.
- Do not bypass formatter or type checks by weakening types.
- Use the local app/package patterns in `web/` before introducing abstractions.

## Documentation conventions

- Follow `docs/engineering/contributing/quality/documentation.mdx`.
- Product docs live in `docs/product/` and need `docs/product/docs.json` nav
  entries when adding pages.
- Engineering docs live in `docs/engineering/` and need
  `docs/engineering/docs.json` nav entries when adding pages.
- Use `bash` for shell code blocks.
- Prefer root-relative internal doc links.
- Do not use em dashes in docs.

## Verification

Choose the smallest check that proves the change.

- Go source change: targeted `mise exec -- bazel test //path:target`.
- Go file added or imports changed: `mise run bazel`.
- Shared Go behavior or broad service change: `mise run test` when practical.
- TypeScript change: targeted `mise exec -- pnpm --dir=web ...` command.
- Formatting-sensitive change: `mise run fmt` or the narrower formatter task.
- Docs-only change: link/content review. Note if no formatter applies.

Report failed or skipped verification honestly.

## High-signal references

- Local development: `docs/engineering/contributing/local/development.mdx`.
- Bazel workflow: `docs/engineering/contributing/tooling/bazel.mdx`.
- Code quality: `docs/engineering/contributing/quality/code-quality.mdx`.
- Testing: `docs/engineering/contributing/quality/testing/index.mdx`.
- Documentation: `docs/engineering/contributing/quality/documentation.mdx`.
