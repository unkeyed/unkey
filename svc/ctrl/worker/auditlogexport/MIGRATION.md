# Audit logs ClickHouse migration

This branch ships the **audit log dual-write + drain + backfill
pipeline** moving audit logs from MySQL `audit_log` to ClickHouse
`audit_logs_raw_v1`. Plus the correlation_id grouping for multi-event
user actions.

Retention is enforced by a static 90-day TTL on the CH table —
no per-workspace lookup, no dict, no sync VO. Per-plan visibility
limits live at the dashboard read layer (`audit/fetch.ts` filters by
`workspace.quotas.auditLogsRetentionDays`); the TTL is just the outer
bound: "we never keep audit logs longer than 90 days." Bump the
INTERVAL clause in `20260508000001.sql` if the most generous plan
grows past 90d.

## Components shipped

### MySQL

- **`clickhouse_outbox`** table — fed by **two** producers, both
  dual-writing in the same MySQL transaction as the underlying mutation:
  - Go API service via `internal/services/auditlogs/insert.go`
  - Dashboard via `web/apps/dashboard/lib/audit.ts` `insertAuditLogs`
- Outbox columns and indexes:
  - `deleted_at bigint unsigned NULL` — marker stamp instead of hard
    delete. Lets ops re-queue events by clearing `deleted_at` and keeps
    an audit trail of what was exported. **No sweep job today**;
    revisit when storage matters.
  - `drainer_pending_idx (deleted_at, version, pk)` — supports the hot
    path query `WHERE version IN (...) AND deleted_at IS NULL ORDER BY
    pk LIMIT N`.

### ClickHouse

- **`audit_logs_raw_v1`** — Nested target arrays, JSON metadata
  columns with materialized text mirrors + tokenbf indexes, partition
  by `inserted_at` (so backfills and late drains land in the current
  month instead of fragmenting old partitions).
  - `TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE`
  - `correlation_id String CODEC(ZSTD(1))` + bloom filter index
    (`idx_correlation_id`) groups events emitted by one logical user
    action so the dashboard can drill from any one event to the rest.
    Auto-minted by both writers when a caller batches >1 events;
    explicitly set via `auditlog.WithCorrelation(ctx, ...)` (Go) or
    `correlationId: newId("correlation")` (TS) for handlers that fan
    out across multiple Insert calls. **Wired sites**: 8 Go API
    handlers (v2 keys/identities create/update/setRoles/etc.) and 5
    dashboard TRPC procedures. Empty on single-event flows.

### Restate Virtual Objects

Both singleton VOs keyed `"default"`, cron'd via
`dev/k8s/charts/restate-cronjobs/values.yaml`.

| VO | Schedule | Purpose |
|---|---|---|
| `AuditLogExportService` | `* * * * *` (every minute) | Drain `clickhouse_outbox` → CH `audit_logs_raw_v1`. Marks `deleted_at` on success. |
| `AuditLogBackfillService` | `0 2 * * *` (daily 02:00 UTC) | Chip through legacy `audit_log` + `audit_log_target` rows pre-dual-write. Cursor (`last_pk`) and cutoff (`cutoff_pk`, snapshotted as `MAX(pk)` on first run) live in VO state. Once `last_pk >= cutoff_pk` the runs are noops. |

The drainer dual-writes happen in the same MySQL transaction as the
underlying mutation, so durability of the outbox row equals durability
of the mutation. CH inserts use ReplacingMergeTree + block-level
deduplication so a crash between CH insert and
`MarkOutboxBatchDeleted` replays the same insert block as a noop.

## Rollback

- Writers stop touching the outbox. Legacy `audit_log` writes continue
  (the original INSERT path was preserved).
- Outbox rows still in MySQL stop being drained; they sit harmlessly
  until cleaned up manually or a future deploy puts the worker back.
  Marked rows can be re-queued by clearing `deleted_at`.
- CH `audit_logs_raw_v1` keeps the data we already inserted; the
  dashboard isn't reading it yet, so nothing breaks.
- Backfill VO sits dormant if the worker reverts; no further work is
  queued.
- Dashboard is unaffected throughout.

## Followup PR 1: dashboard read cutover

Cutover can happen as soon as the **backfill VO catches up** to the
legacy tail (no calendar wait — the backfill brings every historical
row into CH). Watch the `AuditLogBackfillService` cursor in Restate
state; when `last_pk` reaches `cutoff_pk`, backfill is done.

Then switch `web/apps/dashboard/lib/trpc/routers/audit/fetch.ts` from
MySQL to ClickHouse:

- Delete the `UNKEY_PLATFORM_WORKSPACE_ID = ""` and `platformBucketFor`
  helpers — the new schema stores `workspace_id` directly.
- Query `audit_logs_raw_v1` with `WHERE workspace_id = ? AND bucket = ?`.
  Targets come back as parallel arrays in the row; map them in the
  serializer instead of stitching with `groupUniqArray`.
- Apply the per-plan visibility filter on the read path:
  `AND time >= now64(3) - INTERVAL workspace.quotas.auditLogsRetentionDays DAY`.
  This is what enforces per-workspace retention to the user; the
  table's 90d TTL is just the outer bound.
- The 5-minute count cache stays. The list query becomes a straight
  `ORDER BY time DESC LIMIT 50` with no `GROUP BY` (Nested targets
  removed the need to aggregate).
- **Surface `correlation_id`** in the timeline: events sharing one
  correlation_id collapse into a single expandable row.

Verify:

```sql
EXPLAIN indexes = 1
SELECT * FROM default.audit_logs_raw_v1
WHERE workspace_id = '<ws>' AND bucket = 'unkey_mutations'
  AND time >= now64(3) - INTERVAL 1 DAY
ORDER BY time DESC, event_id DESC LIMIT 50;
```

Expect partition pruning by month and primary key access by
`(workspace_id, bucket)`.

Rollout: feature flag scoped to one test workspace → spot-check →
ramp to 100% → pull the flag.

## Followup PR 2: legacy cleanup

Once the dashboard has been on CH for a week with no rollback signal:

1. **Stop the legacy writes.** Delete the `audit_log` and
   `audit_log_target` INSERT branches from `insertLogs`. The outbox
   INSERT is now the only write.
2. **Drop the schema.** Atlas migration: `DROP TABLE
   audit_log_target; DROP TABLE audit_log;`. PlanetScale reclaims
   storage; the seven indexes on `audit_log` (which existed solely
   to serve the dashboard reads we just moved) stop costing write IO.
3. **Drop dead sqlc queries.** `audit_log_insert.sql`,
   `audit_log_target_insert.sql`, `audit_log_find_target_by_id.sql`,
   `audit_log_find_for_backfill.sql`,
   `audit_log_target_find_for_backfill.sql`.
4. **Decommission the backfill VO.** Once legacy tables are gone the
   cursor never advances; remove the cron entry, the binding in
   `svc/ctrl/worker/run.go`, and the `auditlogbackfill` package.

## Post-deploy smoke test

1. **Outbox is filling.** Hit any mutating endpoint. Within seconds:
   ```sql
   SELECT pk, version, workspace_id, event_id, deleted_at, created_at
   FROM clickhouse_outbox
   ORDER BY pk DESC LIMIT 5;
   ```
2. **Drainer is draining.** After the next minute tick:
   ```sql
   SELECT MIN(created_at), COUNT(*)
   FROM clickhouse_outbox
   WHERE deleted_at IS NULL;
   ```
   Backlog should be near zero.
3. **CH has the row.**
   ```sql
   SELECT event_id, workspace_id, event, correlation_id
   FROM default.audit_logs_raw_v1
   WHERE workspace_id = '<ws>'
   ORDER BY time DESC LIMIT 5;
   ```
   Expect `correlation_id` populated for any multi-event flow you
   triggered (e.g. `keys.createKey` with permissions or roles).
4. **Backfill cursor is advancing.** Inspect the
   `AuditLogBackfillService` VO state via the Restate UI; `last_pk`
   should grow until it catches up to `cutoff_pk`.

## Operational checklist

- **Grant INSERT to the ctrl user.** Before deploying the worker:
  ```sql
  GRANT INSERT ON default.audit_logs_raw_v1 TO ctrl;
  ```
  Both `AuditLogExportService` (drainer) and `AuditLogBackfillService`
  use the ctrl user. Without the grant the first drain tick 500s with
  ACCESS_DENIED. See
  `docs/engineering/infra/clickhouse/users/ctrl.mdx` for the full
  ctrl-user grant set.
- **CH Cloud backup retention.** Bump the cluster's automated backup
  retention to at least **30 days** (default 24h). TTL deletes are
  irreversible from the table itself; backups are the only undo for
  "we deleted the wrong workspace's logs."
- **Outbox lag** is the main thing to monitor:
  ```sql
  SELECT MIN(created_at), COUNT(*)
  FROM clickhouse_outbox
  WHERE deleted_at IS NULL;
  ```
  Page if it exceeds a few minutes.
- **Unknown payload versions pile up silently.** Run periodically:
  ```sql
  SELECT version, COUNT(*) FROM clickhouse_outbox
  WHERE deleted_at IS NULL
  GROUP BY version;
  ```
  If anything other than `audit_log.v1` is non-zero, the drainer's
  `knownVersions` list is missing a handler.
- **Backfill progress.** The `AuditLogBackfillService` cursor is
  inspectable via Restate state (`last_pk`); it's done when it
  reaches `cutoff_pk`.
