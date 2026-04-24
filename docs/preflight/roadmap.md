# Preflight roadmap

Living document. Update in the same PR as the work it describes.

- **Owner**: whoever's currently pushing preflight (update as handoff happens).
- **Original plan**: see the project's pre-implementation plan for the
  tier-1/2/3/4 probe inventory. This file tracks execution of that plan.

---

## Status — 2026-04-24

**Shipped** on branches `preflight/skeleton` (unkey) and
`preflight/runbooks` (infra):

| #   | Summary                                                                                                                                  | Repo  | Commit      |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------- | ----- | ----------- |
| 1   | Framework scaffold: Probe interface, Runner (metrics/logs/tracing/artifacts/shadow), core types, manifest tool, developer docs, CI guard | unkey | `a2b526bb7` |
| 2a  | testapp (10 endpoints, Dockerfile, integration tests), real `wait.Poll[T]`                                                               | unkey | `7a84640a4` |
| 2b  | harness package + `TestDev` (full dev flow against real MySQL/CH/Restate/Vault + in-process ctrl API)                                    | unkey | `6dd644bf2` |
| 2c  | tier-1.1 `github_webhook` probe with `accept_valid` / `reject_invalid` phases                                                            | unkey | `6051ac310` |
| —   | this document (initial)                                                                                                                  | unkey | `b74db4614` |
| 3   | Staging scaffold: Helm chart + ApplicationSet + `setup.md`                                                                               | infra | `455ab5e`   |
| 5   | tier-1.3 `create_deployment` probe: ctrl API CreateDeployment + GetDeployment RPCs                                                       | unkey | `3b1aeb360` |
| 5b  | MySQL/ClickHouse split: CH wired into binary, MySQL stays harness-only                                                                   | unkey | `59fb6007f` |
| 5c  | `clickhouse_connectivity` probe (CH Ping prereq check)                                                                                   | unkey | `7b0c46c6d` |
| 5d  | Move code from `cmd/preflight/` to `svc/preflight/` to match `cmd/run/X` + `svc/X` convention                                            | unkey | `15cb9c723` |
| 5e  | Probe-per-subfolder + `core.Pass/Fail/Failf` + chainable `With*`                                                                         | unkey | `d3aea0e20` |
| 5f  | `wait` helper moved into `svc/preflight/internal/wait` (not a probe)                                                                     | unkey | `f98bdd430` |
| 6a  | tier-1.11 `request_logs` probe: polls `sentinel_requests_raw_v1` for the run's row                                                       | unkey | `e10cf8ad8` |
| —   | runbook directory scaffold + cross-repo CI + noop runbook                                                                                | infra | `d0d3ba5`   |
| —   | `github_webhook` runbook                                                                                                                 | infra | `c03cdc8`   |
| —   | wire ClickHouse secret into Helm chart                                                                                                   | infra | `5a2d20b`   |
| —   | `create_deployment` runbook                                                                                                              | infra | `a300bcf`   |
| —   | `clickhouse_connectivity` runbook                                                                                                        | infra | `6c69a53`   |
| —   | path updates after the cmd→svc move                                                                                                      | infra | `aa0bfb5`   |
| —   | `request_logs` runbook                                                                                                                   | infra | `9eb1846`   |

**Coverage**: 6 probes across two folders:

| Probe                     | Dev-viable        | Staging-viable             | Notes                                                                                          |
| ------------------------- | ----------------- | -------------------------- | ---------------------------------------------------------------------------------------------- |
| `noop`                    | ✓                 | ✓                          | Runner smoke signal; delete when tier-1 is fully populated                                     |
| `github_webhook`          | ✓                 | needs `setup.md` done      | Accept valid + reject invalid                                                                  |
| `create_deployment`       | ✓                 | ✓ (with project IDs wired) | CreateDeployment + GetDeployment RPC round-trip                                                |
| `clickhouse_connectivity` | ✓                 | needs CH user provisioned  | Fails first when CH is down, so later CH probes fail cleanly                                   |
| `request_logs`            | ✓ (harness seeds) | needs tier-1.10 traffic    | Reads `sentinel_requests_raw_v1` for the run ID                                                |
| `git_push` (realinfra)    | ✗                 | ✓                          | Pushes commit via GitHub App, polls per-commit hostname for the new SHA. Whole-pipeline probe. |

`TestDev` runs the full 5-probe suite end-to-end in ~23s (warm) /
~90s (cold containers). Per-probe unit tests in ~30-90s each.

---

## Decisions log

Entries are dated; rationale matters more than the decision itself.

**2026-04-23 — Runbooks live in the infra repo, not unkey.**
Rationale: `runbook_url` annotations live on the `PrometheusRule` yaml,
which lives in infra. Co-locating avoids drift. Cross-repo sync is a
committed `svc/preflight/probes/MANIFEST.txt` plus infra CI that
enforces every name has a matching `.md`. [Source: chat, tier-1.1 PR.]

**2026-04-23 — Probes are thin; the Runner owns everything cross-cutting.**
Probes forbid imports of prometheus, OTEL, logger, artifacts, alert
fan-out. Typical probe is ~40 lines. Adding a new cross-cutting output
(PagerDuty, Sentry) is a one-file Runner change.

**2026-04-23 — Dev target is test-only.**
`--target=dev` on the binary prints an error and exits. The dev flow
runs as `go test -run TestDev ./svc/preflight/harness/...` because
`pkg/dockertest` is `*testing.T`-bound and shimming it into a binary
path isn't worth the complexity.

**2026-04-23 — Dedicated harness for preflight (not `svc/ctrl/integration/harness`).**
The existing harness stops at the Restate ingress boundary and mocks the
ctrl HTTP API; preflight needs the real ctrl API. The new harness at
`svc/preflight/harness/` wraps `pkg/dockertest` + `svc/ctrl/api.Run` and
is usable from both probe unit tests and `TestDev`.

**2026-04-23 — Harness tests are sequential, not `t.Parallel`.**
Parallel harness tests competed for the shared Restate container's
admin endpoint and occasionally timed out worker registration.
Sequential within a package. Across packages: `go test -p 1` required
because probe packages also share the Restate container.

**2026-04-23 — TLS cert runway is NOT a preflight probe.**
An existing healthcheck already covers it. Tier-4 focuses on invariants
nothing else watches.

**2026-04-23 — Git-source path in prod, not just docker-image.**
A prod regression that only reproduces with a real Depot build is
exactly what we want to catch. Build cost (~96 runs/day × regions) is a
rounding error versus the regression it would miss.

**2026-04-24 — MySQL stays nil in the binary; ClickHouse gets wired.**
The first instinct was "probes should not touch databases directly."
Walking through the tier-1/3 probe list showed that's too strict for
ClickHouse and correct for MySQL.

For ClickHouse: tier 1.11 (request logs) and tier 3.22 (reconciliation
drift) both need raw analytics queries that have no RPC equivalent and
shouldn't have one; wrapping `sentinel_requests_raw_v1` in an RPC
would just be a thin passthrough. Credentials are minted via the
existing per-workspace ClickHouse user path
(`svc/ctrl/worker/clickhouseuser`); preflight uses the same mechanism
a customer SDK would. Blast radius of a leak = preflight's own
analytics data, which is designed to be customer-readable anyway.

For MySQL: the whole point of probing staging/prod is to verify the
customer surface. Bypassing to direct SQL skips the auth / RLS /
workspace-scoping layers a real regression would break first, and
blows up credential blast radius (MySQL creds leak identity, keys,
secrets). Probes that need state go through a ctrl API RPC; if no
RPC exists, the right move is to add it, not to grant SQL.

Dev harness is the inverse on purpose: `harness.Env()` populates both
DB and ClickHouse so unit tests can assert on table state without
needing an RPC first. Same probe code, gated by
`if env.DB != nil { ... }` when behavior diverges.

**2026-04-24 — `cmd/preflight` is thin; everything substantive lives in
`svc/preflight`.** Matches the `cmd/run/sentinel` + `svc/sentinel`
convention used elsewhere in the repo. `cmd/preflight/main.go` is now
~100 lines of flag definitions plus a one-line call into
`svc/preflight.Run(ctx, Config)`. Keeps the CLI surface separable from
the library and is what future readers expect.

**2026-04-24 — Each probe is its own package under `svc/preflight/probes/<name>/`.**
Probe + test sit together; package-private helpers stay private;
registering still works via a single blank-import file at
`svc/preflight/registered.go`. Probe unit tests import only their
own package, keeping cold-start small. The registry (interface,
Register, Names, ByName, All) stays at `svc/preflight/probes/` so
probes have something to import.

**2026-04-24 — `core.Pass() / core.Fail(err) / core.Failf(fmt, args)`
with `.WithPhases / .WithDims / .WithArtifacts` chainers.** Cleaner
than six-field struct literals and respects exhaustruct (constructors
fill every field at the definition site). Promoted what had been a
package-private `failf` in `create_deployment.go` so every probe uses
the same shape.

**2026-04-24 — `wait/` lives in `svc/preflight/internal/wait`, not
under `probes/`.** It is a polling helper, not a probe. Go's
`internal/` visibility rules keep it unimportable outside the preflight
subtree, which matches intent.

**2026-04-24 — Harness context budget is 300s, not 120s.**
Cold-start variance on a busy host (MySQL 15s + ClickHouse 10s + Restate
20s + admin-registration retry up to 60s + seed + ctrl API startup)
bumps against 120s occasionally. Steady-state runs still finish in
~20s; the higher ceiling only matters when containers are truly cold.

**2026-04-24 — `env.GitHub / Thanos / Kube / TestApp` deleted.**
They were `any`-typed placeholders from phase 1 that erased the type
contract the Runner and probes depend on. Removing them makes "wrong
field used" a compile error, not a runtime surprise. New probes that
need a shared client add a typed field in the same PR.

**2026-04-24 — Diagnoser interface for on-failure enrichment.**
A failed probe is the worst moment to restrict operators to
guessing. The Runner calls an optional `probes.Diagnoser.Diagnose`
method on failure and appends the returned artifacts to the bundle.
Diagnose is explicitly allowed to read `env.DB` when non-nil;
diagnostics are not assertions, so the "primary assertions go
through the customer surface" rule does not apply. Runner wraps
Diagnose in a 5s deadline and a recover() so a broken diagnostic
path never masks the probe failure it was meant to explain.

**2026-04-24 — TOML config via `pkg/config`.** Matches
`cmd/run/sentinel`. The binary's flag surface shrinks to
`--config` / `--config-data`; everything else lives in a TOML file.
Secrets are `${VAR}` references expanded at load time via
`os.ExpandEnv`; Helm injects them as env vars from ExternalSecret.
Reasoning: keeps secret rotation a Helm concern, keeps non-secret
config in a reviewable config-as-code file, matches the rest of the
repo.

**2026-04-24 — Probes split into `common/` vs `realinfra/`.**
Directory structure now encodes where a probe runs. `common/` =
every target; `realinfra/` = needs real Depot / Krane / frontline /
GitHub App. Each subfolder has a README with admission criteria so
new contributors know where their probe belongs without reading
`DevSuite` + `DefaultSolo` together.

---

## Phase plan

Phases are chronological; tiers (1/2/3/4) refer to probe-priority
classes from the original plan and cut across phases.

### Phase 3 — Staging scaffolding (CODE DONE, setup pending)

All infra-side code landed (`455ab5e` + `5a2d20b`): Helm chart,
ApplicationSet, per-region overrides, promotions entries, setup
runbook, ClickHouse secret wiring. Nothing more for me to write here.

**Pending user action** (`setup.md` in the infra repo):

- Create the **Unkey Preflight** workspace in each environment.
- Mint a workspace-scoped ctrl API token.
- Stand up a dedicated preflight GitHub App (NOT shared with prod).
- Provision a workspace-scoped read-only ClickHouse user.
- Populate `unkey/preflight` in AWS Secrets Manager with
  `PREFLIGHT_CTRL_TOKEN`, `PREFLIGHT_GITHUB_WEBHOOK_SECRET`,
  `UNKEY_CLICKHOUSE_URL`.

**Exit criteria**: `enabled: true` on one canary region; the suite
runs green every 5 minutes for 72 hours with shadow-mode alerting
(logs only, no pages). Then flip production regions in sequence.

### Phase 4 — Sentinel policy probes (TODO, gated on Phase 3)

The probes we actually want. Each configures a `sentinel_config` on the
preflight app, deploys, hits the app, asserts the policy fired.

- **`sentinel_ratelimit`**: 5 req/s policy; 6 requests → first 5 succeed,
  last 429.
- **`sentinel_apikey`**: API-key verification required; no key → 401;
  valid key → 200.
- **`sentinel_openapi`**: schema with `x-unkey-redact` on `/admin/*`;
  request body containing the sentinel token is redacted in
  `sentinel_requests_raw_v1`.
- **`sentinel_ip_allowlist`**: `1.2.3.0/24` allowed; `X-Forwarded-For`
  outside range → 403.

Each ships as: probe file + unit test against the harness +
runbook in infra + MANIFEST regeneration.

**Decision deferred**: these probes need the testapp redeployed with
different `sentinel_config` per probe. Unclear whether to (a) deploy
four testapp variants ahead of time, or (b) update the existing
testapp's sentinel config live, or (c) deploy a fresh app per probe
run. Decide during phase-4 kickoff.

### Phase 5 — Tier 1.3 `create_deployment` probe (DONE)

Dev-harness-viable. Calls `CreateDeployment` over ConnectRPC with a
`DockerImage` source, asserts `PENDING` status + non-empty
deployment ID, round-trips through `GetDeployment`.

Did **not** assert on `deployment_steps` progression or outbox rows:
dev harness has no worker to drive them forward, and staging/prod Env
does not expose direct MySQL. Later tier-1 probes (workflow-advancing)
cover that instead.

Env extensions (`PreflightProjectID`, `PreflightAppID`,
`PreflightEnvironmentSlug`) + `NullRestateServices()` gaining a
`DeployService` mock came with this phase.

### Phase 6 — Remaining tier-1 probes

| Probe                    | Status             | Notes                                                                       |
| ------------------------ | ------------------ | --------------------------------------------------------------------------- |
| 1.2 `build_depot`        | TODO               | staging only; real Depot build                                              |
| 1.4 `deployment_steps`   | TODO               | staging-preferred; dev viable if harness nullDeploy actually advances steps |
| 1.5 `env_var_encryption` | TODO               | staging only                                                                |
| 1.6 `injected_env_vars`  | TODO               | staging only                                                                |
| 1.7 `upstream_protocol`  | TODO               | staging only                                                                |
| 1.8 `ephemeral_disk`     | TODO               | staging only                                                                |
| 1.9 `healthcheck_probe`  | TODO               | staging only                                                                |
| 1.10 `frontline_routing` | TODO               | precondition for 1.11 in staging                                            |
| 1.11 `request_logs`      | DONE (`e10cf8ad8`) | dev via harness seed; staging via 1.10 traffic                              |

### Phase 7 — `api-platform` suite (TODO)

Orthogonal to deploy pipeline: tests the unkey SaaS surface itself
(key create/verify/revoke, rate-limit enforcement, analytics ingest).
Separate suite, separate CronJob schedule, shares the probe library.

- **`key_create_verify`**: create via SDK → verify → assert 200 + payload.
- **`key_ratelimit`**: limited key → hammer → assert 429 at RPS.
- **`key_revoke`**: revoke → verify → assert 401.
- **`key_analytics_ingest`**: verify → poll
  `key_verifications_raw_v2` for the event.

Harness extension needed: `harness.Config{WithAPIService: true}` runs
`cmd/run/api` in-process alongside ctrl. Staging/prod: wired via
`PREFLIGHT_API_URL` + `PREFLIGHT_API_ROOT_KEY` env vars.

### Phase 8 — Tier 2 probes (TODO, staging-gated)

Sticky-route modes, rollback, HPA scaling, graceful shutdown, gVisor,
Cilium enforcement, custom-domain ACME, cross-region forwarding. Non-
gating in the runner; emit metrics, shadow for a week before paging.

### Phase 9 — Tier 3 monitoring probes (TODO)

Deploy-duration SLO, reconciliation drift, topology spread, build-step
completeness + redaction, error-path coverage. Lower cadence (hourly).

### Phase 10 — Tier 4 background invariants (TODO)

Image retention coverage + policy, rollback restore (monthly), GitHub
App token validity, Vault keyring status, S3 lifecycle sanity. Daily
cadence.

### Phase 11 — Alert graduation + SLO enforcement (TODO)

Ship `prometheusrule.yaml` with MWMBR alerts after 72h green baseline
per probe. Route Slack first, incident.io after another week.

---

## Open questions / parking lot

Things we've raised but not decided. Drop here so they don't get lost.

- **Sentinel probe deploy fan-out**: one testapp variant per sentinel
  policy, or reconfigure in place? (See Phase 4.)

- **Dedicated preflight GitHub App instance vs shared with prod**:
  answered in setup.md step 3 (dedicated). Leaving the question here
  until the GitHub App actually exists.

- **Probe-level shadow list configuration**: currently a hardcoded
  `[]string` on the Runner. Should be per-env Helm values; lift when
  we start shadow-listing probes on specific regions.

- **ArtifactUploader S3 wiring**: interface exists, no implementation
  yet. Add when the first probe genuinely needs to upload artifacts
  (github_webhook already attaches them but we do not yet have the S3
  writer).

- **`tier-1.10 frontline_routing` precondition**: several staging probes
  (1.11 already, future 1.5-1.9) want an earlier probe that actually
  hits the deployed hostname. Decide 1.10's shape when it lands.
