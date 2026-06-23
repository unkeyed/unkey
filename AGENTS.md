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

## Cursor Cloud specific instructions

The startup update script already runs `dev/install-mise`, `mise install`, and
`mise run install-web`, so the pinned toolchain and `web/` deps are ready. The
notes below are the non-obvious caveats for running the stack.

### Commit co-author email

- The startup update script also patches the Cursor-managed co-author hook
 (`~/.cursor/agent-hooks/*/commit-msg.cursor.co-author`) to use GitHub's
 canonical no-reply email `Flo <53355483+Flo4604@users.noreply.github.com>`
 instead of the legacy `Flo <Flo4604@users.noreply.github.com>`. The legacy
 form does not resolve to the GitHub account, which left the CLA bot pending.
 Keep that line in the update script; the hook is `@cursor-managed` and can be
 regenerated with the legacy email. The durable fix is to set the correct
 email in Cursor account settings.

### mise / PATH

- `mise` lives at `$HOME/.local/bin/mise` and is not on `PATH` by default in
 non-login shells. Use the full path, or `eval "$(mise activate bash --shims)"`,
 or prefix tool commands with `mise exec -- <tool>` (e.g. `mise exec -- bazel`).
 `mise run <task>` works without activation.

### Docker (required, not auto-started)

- Docker Engine is installed but the daemon is not running on boot. Start it
 once per session: `sudo dockerd > /tmp/dockerd.log 2>&1 &` (best run in a
 dedicated `tmux` session so it outlives the command).
- The `ubuntu` user is added to the `docker` group, but shells started before
 that change cannot reach the socket. Quick unblock:
 `sudo chmod 666 /var/run/docker.sock`.
- This is Docker 29; `/etc/docker/daemon.json` must set
 `storage-driver: fuse-overlayfs` and `features.containerd-snapshotter: false`
 for the VM kernel. If Docker fails to start with overlay errors, recreate that
 file (and `update-alternatives --set iptables /usr/sbin/iptables-legacy`).

### Running the full stack (dashboard + core key flows)

- Lightest path is `mise run dashboard`, but it depends on `build` + `oci-load`,
 which compile the entire Bazel graph (first run is long, ~15 min) and load 7
 service images. The dashboard docker-compose only needs 4 of them, so to save
 time you can build just those:
 `mise exec -- bazel run //build/api:load` and likewise for `control-api`,
 `control-worker`, and `vault`. The full `oci-load` also builds
 `frontline`/`heimdall`/`krane`.
- Then start infra and seed directly:
 `docker compose -f web/apps/dashboard/dev/docker-compose.yaml up -d --wait`
 then `./bin/unkey dev seed local`, then `pnpm --dir=web/apps/dashboard dev`
 (dashboard on `http://localhost:3000`).
- The dashboard needs `web/apps/dashboard/.env`; create it with
 `cp web/apps/dashboard/dev/.env.example web/apps/dashboard/.env`. Do NOT run
 `mise run bootstrap` unattended; it is interactive and requires a Depot token.
- `AUTH_PROVIDER="local"` makes the dashboard auto-authenticate into workspace
 `ws_local` with no login screen.

### Service ports

- dashboard 3000, data-plane api 7070, ctrl-api 7091, vault 8060, restate
 8081/9070, mysql 3306, clickhouse 8123/9000, redis 6379.

### Seeding and the data-plane API

- `./bin/unkey dev seed local` prints a root key (and writes `dev/.env.seed`).
 That root key is scoped to workspace `ws_local`. The seeded "project" does NOT
 create a row in the `apis` table, so to exercise the data plane create an API
 first: `POST /v2/keys`... use `/v2/apis.createApi`, then `/v2/keys.createKey`,
 then `/v2/keys.verifyKey` (all on `http://localhost:7070`, bearer = root key).
### Full deploy stack (`mise run dev`) does NOT run in Cursor Cloud

- The full Tilt + minikube deploy stack cannot start in the Cursor Cloud agent
 VM. This is an environment limitation, not a code issue.
- Root cause: the VM's cgroup v2 root is `domain threaded` (verify with
 `cat /sys/fs/cgroup/cgroup.type`). A threaded subtree cannot contain domain
 child cgroups, only exposes the `cpuset`/`cpu`/`pids` controllers (no
 `memory`/`io`), and rejects moving a process into a child cgroup with `EIO`.
 The root type cannot be reverted to `domain` from inside the VM (`EIO`).
- Consequently every Kubernetes node agent fails at startup:
 - minikube (docker driver, systemd-based kicbase node):
   `Failed to create /init.scope control group: Structure needs cleaning`.
 - k3s/k3d (kubelet):
   `failed to evacuate root cgroup: read /sys/fs/cgroup/cgroup.procs: operation not supported`.
 - kind, microk8s, etc. fail the same way (all build a `kubepods` cgroup tree).
 - The k3s control plane alone (`server --disable-agent`) does start, since it
   runs as in-process components, but it cannot schedule real pods.
- Net effect: `krane`, `heimdall`, `frontline`, topolvm, cilium, and the
 deploy/gateway flow cannot be exercised here. Use the docker-compose path
 above for the dashboard and core key/ratelimit/RBAC flows. Test deploy-stack
 code with targeted Bazel unit/integration tests instead.
- The `mise run dev` path is also heavier and historically needs a Depot token
 for builds (`dev/.env.depot`); not required for the compose path.
