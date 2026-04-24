# preflight

End-to-end probe for the unkey deploy pipeline.

Runs against a target control plane and exercises every stage a real
customer push travels through: GitHub webhook, Depot build, Krane
provisioning on EKS, Vault env decrypt, Cilium policy, frontline routing,
sentinel, and the ClickHouse log pipeline.

## Architecture at a glance

```
  ┌────────────┐      ┌──────────┐      ┌──────────┐
  │  cmd/main  │ ───▶ │  Runner  │ ───▶ │  Probe   │
  └────────────┘      └──────────┘      └──────────┘
        │                  │                  │
        │                  │  (span, timer,   │
        │                  │   metrics, log,  │
        │                  │   artifacts)     │
        │                  │                  │
        │                  ▼                  ▼
        │            Prometheus          *core.Env
        │            OTEL tracer         (shared clients)
        │            pkg/logger
        │            S3 artifacts
        │
        └────────── flow.go picks which probes
                    from the registry make up a suite
```

The key invariant: **probes do not import prometheus, OTEL, the logger, or
S3**. They return a typed `core.Result` and the Runner does the rest. If
that invariant holds, adding a probe is ~30 lines of assertion code.

## Running locally

Three modes, in order of setup cost:

### 1. Unit-test harness (fastest, no cluster)

```
go test -run TestDev ./svc/preflight/harness/...
```

Spins up MySQL, ClickHouse, Restate, Vault containers via
`pkg/dockertest`, runs the ctrl API in-process, seeds a preflight
workspace, and invokes `DevSuite`. Requires the shared test
containers from `dev/docker-compose.test.yml` (handled by
`make test`). Ideal for per-probe debugging.

### 2. Local minikube cluster (real Cilium, real gVisor, real GitHub App)

The dev stack in `dev/cluster.yaml` is a minikube cluster with the
same shape as production: Cilium CNI, gVisor runtime, metrics-server.
Paired with `unkey dev github setup` + `unkey dev github tunnel`
(ngrok-exposed webhooks), this runs the FULL suite against a real
GitHub App — including the `realinfra/git_push` probe.

```
# One-time setup
unkey dev github setup --app-name my-preflight-dev     # creates App, writes dev/.env.github + pem
unkey dev seed local --slug=preflight                  # seeds workspace + root key
# Install the GitHub App on unkeyed/preflight-test-app (or a fork)
# and note the installation ID from the URL.

# Every dev session
unkey dev github tunnel                                # starts ngrok -> localhost:7091
cp dev/preflight.toml.local.example dev/preflight.toml.local
source dev/.env.seed
source dev/.env.github
export PREFLIGHT_GITHUB_PRIVATE_KEY="$(cat dev/.github-private-key.pem)"
export PREFLIGHT_GITHUB_INSTALLATION_ID=<id-from-github-url>

UNKEY_CONFIG=dev/preflight.toml.local unkey preflight
```

Skipping the App install? Set `suite = "solo-dev"` in the TOML to
exclude `git_push`; every other probe still runs.

### 3. Staging / prod (real customer flow)

Runs as a Kubernetes CronJob deployed by
`infra/eks-cluster/helm-chart/preflight/`. See
`infra/docs/runbooks/preflight/setup.md` for the one-time setup
(workspace, GitHub App, AWS Secrets Manager, MySQL GRANTs).

Against staging (real control plane, requires a workspace-scoped
token):

```
PREFLIGHT_TARGET=staging \
PREFLIGHT_REGION=eu-central-1 \
PREFLIGHT_CTRL_URL=https://ctrl.staging.unkey.app \
PREFLIGHT_CTRL_TOKEN=... \
PREFLIGHT_GITHUB_WEBHOOK_SECRET=... \
  go run ./ preflight
```

Why the split: the harness wraps `pkg/dockertest`, which takes a
`*testing.T`. Shimming it into a binary path would add a kludgy fake
testing.T; keeping it test-only is cleaner and the developer loop
(`go test`) already gives you caching, `-run` filtering, and proper
exit codes.

## Running in Kubernetes

See `infra/eks-cluster/helm-chart/preflight/` for the Helm chart and
`docs/preflight/architecture.md` for how CronJobs are laid out per
region.

## Related docs

Developer docs (this repo):

- [`docs/preflight/adding-a-probe.md`](../../docs/preflight/adding-a-probe.md) — write a new probe in under a day.
- [`docs/preflight/architecture.md`](../../docs/preflight/architecture.md) — where this sits in the deploy pipeline.

Oncall / operational docs live in the infra repo, next to the
PrometheusRule yaml that links to them:

- `infra/docs/runbooks/preflight/<probe_name>.md` — one runbook per
  probe (enforced by infra CI against `svc/preflight/probes/MANIFEST.txt`).
- `infra/docs/runbooks/preflight/oncall.md` — what to do when preflight
  pages.
- `infra/docs/runbooks/preflight/slo.md` — success-rate SLO and error
  budget.
