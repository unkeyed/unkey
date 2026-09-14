# Local anomaly beta

Run the OOMKilled/CrashLoopBackOff beta with MySQL, ClickHouse, Redis, Restate,
the control API and worker, the public API, and the dashboard. This optional
Amp orb setup does not start Tilt, Kubernetes, Vault, or build infrastructure.
Other dashboard features that need those services are not covered.

## Start or resume

Run from the repository root after the normal orb setup installs Docker, mise,
and the pinned tools. Don't run the full dashboard or Tilt stack on these ports
at the same time.

```bash
mise run anomaly-beta
```

The command prints the dashboard portal. Open `/local/alerts` on that portal.
Local authentication and the feature flag apply only to this dashboard process.
The control API and worker allowlist only `ws_local`.

The command uses the dashboard Compose project's local MySQL and ClickHouse
volumes. It never deletes volumes or migrates an existing schema. An old schema
fails the startup check. Inspect and update that disposable database separately.
Rebuilding an image does not update a populated volume's schema.

The local seed runs only when `ws_local` is absent. The fixture creates a ready
deployment and running topology, not alerts. Repeated runs preserve existing
alerts, processed inbox identities, and deployment state. The fixture refuses
to replace another current deployment or local cluster. Stop the full stack
before using this setup. Don't run fixture creation concurrently.

Binaries and private seed output live in `.cache/anomaly-beta/`. The checked-in
credentials are local test values. Never use these configs on a shared system.
Only the dashboard gets a portal. Existing `.env` files are not rewritten.

## Process a report

Send synthetic container reports through
`ctrl.v1.ClusterService/ReportInstanceEvents` on control API port 7091, with
`Authorization: Bearer local-anomaly-test-only`. Use cluster cell
`cell_anomaly_beta`, platform `dev`, region `local`, deployment
`dep_anomaly_local_beta`, and the seeded app/project/production environment IDs.
No real pod runs in this setup.

After a report succeeds, run one bounded poll:

```bash
mise run anomaly-beta tick
```

The request returns an asynchronous invocation ID, not proof that processing
finished. Check the worker logs and inbox rows. Each UTC minute has one
idempotency key, so another tick in the same minute reuses the first invocation.
No recurring caller is installed. The dashboard inbox refreshes every 60 seconds.
The regular five-minute detector and recovery need telemetry and their own
caller; this setup does not generate quiet windows or resolve alerts for you.

## Stop the application services

Stop the four supervised application services without deleting data:

```bash
mise run anomaly-beta stop
```

The dependency containers remain running. To stop them too:

```bash
mise exec -- docker compose -p anomaly-beta-local \
  -f dev/anomaly-beta/restate.yaml stop
mise exec -- docker compose \
  -f web/apps/dashboard/dev/docker-compose.yaml stop mysql clickhouse redis
```

Keep the Restate container when resuming this local experiment. It stores its
journal in the container, not a named volume. Removing or recreating it loses
local Restate state. MySQL still retains alerts and processed inbox identities.
