# Preflight architecture

Where preflight sits in the unkey stack, and why.

## Placement

```
           [ GitHub push ]                     [ preflight CronJob ]
                  │                                     │
                  ▼                                     ▼
           ctrl /webhooks/github <───── synthetic signed payload ──────┐
                  │                                                    │
                  ▼                                                    │
          Restate DeployService                                         │
                  │                                                    │
                  ▼                                                    │
              Depot build                                              │
                  │                                                    │
                  ▼                                                    │
         deployment_changes outbox                                      │
                  │                                                    │
                  ▼                                                    │
              Krane (EKS)                                              │
            gVisor pod + HPA                                           │
            + Cilium NetPolicy                                         │
                  │                                                    │
                  ▼                                                    │
        frontline (NLB, per-region) ◀── synthetic HTTP request ────────┘
                  │
                  ▼
              sentinel
            (reverse proxy)
                  │
                  ▼
               testapp
            (preflight pod)
                  │
                  ▼ (batch)
     sentinel_requests_raw_v1 ─── ClickHouse assertion
```

Preflight is the only test that traverses all of these in a single run.
The existing `svc/ctrl/integration/harness` stops at the Restate
boundary; the ALB/NLB liveness endpoints never touch the control plane.

## Components

- **CLI entry: `cmd/preflight`.** Thin wrapper: defines the `preflight`
  subcommand, parses flags, calls `svc/preflight.Run(ctx, Config)`.
  Only reason it exists as a separate package is to keep the CLI
  surface separable from the library.
- **Library: `svc/preflight/`.** Run/Config, Runner, flow/suites, and
  the probe registry. Everything substantive lives here. The binary
  supports two targets (`staging`, `prod`); a third target `dev` is
  implemented by the harness subpackage below and is only reachable
  via `go test`.
- **Probes: `svc/preflight/probes/*.go`.** One assertion per file.
  Registered at init; composed into suites via `svc/preflight/flow.go`.
- **Runner: `svc/preflight/runner.go`.** Wraps every probe with
  Prometheus metrics, OTEL spans, structured logs, and S3 artifact
  upload on failure. Implements shadow mode.
- **Test app: `svc/preflight/testapp/`.** Trivial Go HTTP server with
  endpoints that mirror each probe's assertion
  (`/meta`, `/env`, `/probe`, `/disk`, ...). Built as its own container,
  pushed to ECR, deployed through the normal unkey deploy path by the
  probes themselves.
- **Helm chart: `infra/eks-cluster/helm-chart/preflight/`.** One
  CronJob per region per suite; ArgoCD picks up new regions
  automatically.

## Tenant isolation

Every preflight run happens inside a dedicated workspace (**Unkey
Preflight**) per environment. One project, one app, one environment.
The control-plane API token is workspace-scoped, so a compromised
preflight pod cannot touch real customer data. ClickHouse rows tagged
`preflight=1` via an env var are filtered out of real dashboards.

## Scheduling

| Suite | Cadence | Destructive? | Notes |
|-------|---------|--------------|-------|
| solo (tier 1+2) | every 15 min (prod), every 5 min (staging) | no | per-region CronJob |
| tier 3 diagnostic | hourly | no | shared job, longer budget |
| tier 4 background invariants | daily | no | runs once per region |
| monthly rollback restore | `0 8 1 * *` | yes, self-heals | staging only initially |

## Alerting

Multi-window multi-burn-rate (Google SRE Workbook) on
`preflight_run_total{result="fail"}`. A short 15-minute window and a
long 1-hour window must both trip; this keeps transient flakes from
paging. `result="shadow_fail"` is excluded from the burn rate so
probes can soak before graduating.

Every `PrometheusRule` carries a `runbook_url` pointing at
`infra/docs/runbooks/preflight/<name>.md`. Runbooks live in the infra
repo next to the alert yaml that links to them, so an SRE can update a
runbook without touching app code.

Cross-repo enforcement uses a committed manifest. The unkey repo owns
the source of truth for probe names via the probe registry, exported
to `svc/preflight/probes/MANIFEST.txt` by the `probemanifest` tool.
The infra repo reads that file to assert every name has a matching
runbook. CI on both sides keeps the two in sync:

- **Unkey CI** (`.github/workflows/preflight-manifest-check.yaml`):
  fails if the committed MANIFEST.txt diverges from the live registry.
- **Infra CI** (added in phase 2): fails if MANIFEST.txt lists a name
  that has no `infra/docs/runbooks/preflight/<name>.md`.

## Related docs

Developer docs (this repo):

- [`README.md`](../../cmd/preflight/README.md)
- [`adding-a-probe.md`](./adding-a-probe.md)
- [`roadmap.md`](./roadmap.md) — phase status, decisions log, parking lot

Oncall docs (infra repo — shipped alongside the PrometheusRule yaml):

- `infra/docs/runbooks/preflight/oncall.md`
- `infra/docs/runbooks/preflight/slo.md`
- `infra/docs/runbooks/preflight/<probe_name>.md` — one per probe
