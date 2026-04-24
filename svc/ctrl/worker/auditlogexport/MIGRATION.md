# Audit logs (and analytics) ClickHouse migration

This branch ships three things in parallel:

1. The **audit log** dual-write + drain + backfill + S3 archive pipeline.
2. **Per-row dynamic TTL** (`expires_at` column + `TTL expires_at DELETE`)
   on the four Go-written analytics tables (`key_verifications_raw_v2`,
   `ratelimits_raw_v2`, `api_requests_raw_v2`, `sentinel_requests_raw_v1`).
3. The **writer plumbing** that stamps `expires_at` from each workspace's
   `logs_retention_days` quota, so dynamic per-plan retention goes live
   on these tables the moment the deploy ships.

Audit log retention is governed by `audit_logs_retention_days` quota and
is enforced by a separate cron (S3 archive + ALTER DELETE), not by the
table's TTL clause. See "Audit log archive" below.

## Components shipped

### MySQL

- **`clickhouse_outbox`** table (added by the prior commit; this PR
  augments it):
  - `deleted_at bigint unsigned NULL` — marker stamp instead of hard
    delete. Lets ops re-queue events by clearing `deleted_at` and keeps
    an audit trail of what was exported. **No sweep job today**; revisit
    when storage matters.
  - `drainer_pending_idx (deleted_at, version, pk)` — supports the hot
    path query `WHERE version IN (...) AND deleted_at IS NULL ORDER BY pk
    LIMIT N`. Without it the scan walks every marked row before finding
    the next unmarked one (~99% wasted IO once marked rows accumulate).

### ClickHouse

- **`audit_logs_raw_v1`** — Nested target arrays, JSON metadata columns
  with materialized text mirrors + tokenbf indexes, partition by
  `inserted_at` (so backfills and late drains land in the current month
  instead of fragmenting old partitions). **No `TTL DELETE` clause**: the
  `AuditLogArchive` cron is sole authority on retention and physical
  deletion.
- **Analytics tables** (`key_verifications_raw_v2`, `ratelimits_raw_v2`,
  `api_requests_raw_v2`, `sentinel_requests_raw_v1`):
  - Added `expires_at Int64 DEFAULT time + N` (per-table N preserves the
    historical static window: 90d for verifications, 30d for the rest).
  - Replaced static TTL with `TTL toDateTime(fromUnixTimestamp64Milli(expires_at)) DELETE`.
  - Behavior is identical to today for any row the writer doesn't stamp.
  - Once stamped, retention follows `workspace.logs_retention_days`.
- `runtime_logs_raw_v1` already had its own `expires_at DateTime64(3)` +
  TTL; not touched.

### Writers (per-row `expires_at` stamping)

All four Go writers compute `expires_at = time + retentionMs` at
insert. `retentionMs` comes from `s.LogsRetentionMillis(ctx)` on the
zen Session, which reads from a `LogsRetentionResolver` wired once on
the `*zen.Server` via `srv.SetLogsRetentionResolver(...)`. The resolver
is `internal/services/quotaretention.New(quotaCache, db)` — same SWR-
cached `keysdb.Quotas` lookup the rate-limiter already uses. Free-tier
default (30d) applies on cache miss / quota row missing /
`logs_retention_days = 0`.

| Writer | Site | Notes |
|---|---|---|
| KeyVerification | `internal/services/keys/verifier.go:log` | Calls `k.session.LogsRetentionMillis(context.Background())` (deferred after request) |
| Ratelimit (single) | `svc/api/routes/v2_ratelimit_limit/handler.go` | `s.LogsRetentionMillis(ctx)` |
| Ratelimit (multi) | `svc/api/routes/v2_ratelimit_multi_limit/handler.go` | One lookup per request, applied to every result row |
| ApiRequest | `pkg/zen/middleware_metrics.go:WithMetrics` | `s.LogsRetentionMillis(ctx)` |
| SentinelRequest | `svc/sentinel/middleware/logging.go` | Sets `s.WorkspaceID` from proxy tracking, then `s.LogsRetentionMillis(ctx)` |

Resolver wiring happens in two places:
- `svc/api/run.go` — after `caches.New(...)`,
  `srv.SetLogsRetentionResolver(quotaretention.New(caches.WorkspaceQuota, database))`
- `svc/sentinel/run.go` — after `zen.New(...)`,
  `srv.SetLogsRetentionResolver(quotaretention.New(quotaCache, database))`
  (sentinel maintains its own workspace quota cache; it doesn't run the
  keys-service rate-limit path that owns the shared one in the API service).

Setting after construction works because Sessions read the resolver via
back-pointer to the Server, so `SetLogsRetentionResolver` is observed by
all subsequent and concurrently-checked-out sessions.

### Audit log workers (Restate Virtual Objects)

All three are singleton VOs keyed `"default"`, cron'd via
`dev/k8s/charts/restate-cronjobs/values.yaml`.

| VO | Schedule | Purpose |
|---|---|---|
| `AuditLogExportService` | `* * * * *` (every minute) | Drain `clickhouse_outbox` → CH `audit_logs_raw_v1`. Marks `deleted_at` on success. |
| `AuditLogBackfillService` | `0 2 * * *` (daily 02:00 UTC) | Chip through legacy `audit_log` + `audit_log_target` rows pre-dual-write. Cursor (last_pk) lives in VO state. Loops to exhaustion within each invocation. |
| `AuditLogArchiveService` | `0 3 * * *` (daily 03:00 UTC) | `INSERT INTO FUNCTION s3()` rows past `expires_at` as Parquet, then `ALTER TABLE DELETE`. **Sole authority on audit log retention deletion.** Kill switch via `AuditLogArchive.Disabled` config. |

The drainer dual-writes happen in the same MySQL transaction as the
underlying mutation, so durability of the outbox row equals durability
of the mutation. CH inserts use ReplacingMergeTree + block-level
deduplication so a crash between CH insert and `MarkOutboxBatchDeleted`
replays the same insert block as a noop.

## Rollback

Reverting this deploy:

- Writer stops touching the outbox and stops stamping `expires_at`.
  Legacy `audit_log` writes continue (the original INSERT path was
  preserved). Analytics tables fall back to the CH-side DEFAULT
  expression on `expires_at`, which reproduces each table's historical
  static TTL window.
- Outbox rows still in MySQL stop being drained; they sit in the table
  harmlessly until cleaned up manually or a future deploy puts the
  worker back. Marked rows (`deleted_at IS NOT NULL`) can be re-queued
  by clearing `deleted_at` if needed.
- CH `audit_logs_raw_v1` keeps the data we already inserted; the
  dashboard isn't reading it yet, so nothing breaks.
- Backfill VO + Archive VO sit dormant if the worker reverts; no further
  work is queued.
- Dashboard is unaffected throughout.

## Followup PR 1: dashboard read cutover

**Wait at least 30 days after this PR ships before doing the cutover.**
That's the longest free-tier audit log retention. Once 30 days have
elapsed, ClickHouse holds every event a user could possibly query
through the dashboard's filters; anything older is past retention and
the UI hides it anyway.

Then switch `web/apps/dashboard/lib/trpc/routers/audit/fetch.ts` from
MySQL to ClickHouse:

- Delete the `UNKEY_PLATFORM_WORKSPACE_ID = ""` and `platformBucketFor`
  helpers — the new schema stores `workspace_id` directly.
- Query `audit_logs_raw_v1` with `WHERE workspace_id = ? AND bucket =
  ?`. Targets come back as parallel arrays in the row; map them in the
  serializer instead of stitching with `groupUniqArray`.
- The 5-minute count cache stays. The list query becomes a straight
  `ORDER BY time DESC LIMIT 50` with no `GROUP BY` (Nested targets
  removed the need to aggregate).

Verify with:

```sql
EXPLAIN indexes = 1
SELECT * FROM default.audit_logs_raw_v1
WHERE workspace_id = '<ws>' AND bucket = 'unkey_mutations'
  AND time >= now64(3) - INTERVAL 1 DAY
ORDER BY time DESC, event_id DESC LIMIT 50;
```

Expect partition pruning by month and primary key access by
`(workspace_id, bucket)`.

Rollout:

1. Deploy the dashboard read change behind a feature flag scoped to a
   single test workspace.
2. Spot-check a few flag-on workspaces against the MySQL dashboard.
3. Ramp the flag to 100% over a day.
4. Pull the flag.

If anything looks wrong: turn the flag off. The dashboard is back on
MySQL in seconds, no deploy needed.

## Followup PR 2: legacy cleanup

Once the dashboard has been on CH for a week with no rollback signal:

1. **Stop the legacy writes.** Delete the `audit_log` and
   `audit_log_target` INSERT branches from `insertLogs`. The outbox
   INSERT is now the only write.
2. **Drop the schema.** Atlas migration: `DROP TABLE audit_log_target;
   DROP TABLE audit_log;`. PlanetScale reclaims the storage; the seven
   indexes on `audit_log` (which existed solely to serve the dashboard
   reads we just moved) stop costing write IO.
3. **Drop dead sqlc queries.** `audit_log_insert.sql`,
   `audit_log_target_insert.sql`, `audit_log_find_target_by_id.sql`,
   `audit_log_find_for_backfill.sql`,
   `audit_log_target_find_for_backfill.sql`.
4. **Decommission the backfill VO.** Once legacy tables are gone the
   cursor never advances; remove the cron entry, the binding in
   `svc/ctrl/worker/run.go`, and the `auditlogbackfill` package.

## Followup PR 3: extend dynamic retention to runtime_logs

The TS log shipper in `web/internal/clickhouse/src/runtime-logs.ts`
writes `runtime_logs_raw_v1`, which already has `expires_at DateTime64(3)`
with a static 90-day default. Plumb the workspace's `logs_retention_days`
through to that writer so it stamps a per-workspace value (same shape
as the four Go tables in this PR).

## Followup PR 4: correlation_id + idempotency_key on audit logs

Today every audit event is independent. A single user action that fans
out into 3-6 events (e.g. `v2/keys.createKey` → `key.create` + N
permission binds + N role binds) shows up as N unrelated rows in the
dashboard. The drill-down "show me everything that happened when this
key was created" is a timestamp-window guess.

Add two opt-in columns:

- **`correlation_id String CODEC(ZSTD(1))`** — empty when caller didn't
  set one, opaque ID otherwise. Groups events emitted by one logical
  user action.
- **`idempotency_key String CODEC(ZSTD(1))`** — set by writers that
  want safe retries on the MySQL outbox insert side (the CH side is
  already deduped by `event_id` via ReplacingMergeTree). Empty when
  the writer doesn't care.

Both default to empty (NULL-equivalent in CH String columns). Existing
single-event call sites need zero changes.

### Schema delta

```sql
ALTER TABLE default.audit_logs_raw_v1
  ADD COLUMN IF NOT EXISTS correlation_id String CODEC(ZSTD(1)),
  ADD COLUMN IF NOT EXISTS idempotency_key String CODEC(ZSTD(1));

-- Skip-index so dashboard "all events for correlation X" stays cheap.
ALTER TABLE default.audit_logs_raw_v1
  ADD INDEX IF NOT EXISTS idx_correlation_id correlation_id
    TYPE bloom_filter(0.01) GRANULARITY 4;
```

MySQL outbox: `idempotency_key varchar(256) NULL` on `clickhouse_outbox`,
plus a unique index `(version, idempotency_key)` so retries collide on
INSERT instead of producing duplicates.

### Envelope delta

```go
// pkg/auditlog/event.go
type Event struct {
  // ... existing
  CorrelationID  string `json:"correlation_id,omitempty"`
  IdempotencyKey string `json:"idempotency_key,omitempty"`
}
```

### Two correlation patterns the writer supports

**A. Auto-mint on batched Insert.** Most callers already pass `[]AuditLog`
to `auditSvc.Insert`. If `len(batch) > 1` and no `CorrelationID` is set,
the service mints one and stamps every event in the batch. Free
correlation for any caller that batches.

**B. Context-scoped for multi-Insert flows.** Some flows (e.g.
`v2_keys_create_key`) call `Insert` from the main handler AND from
nested helpers (`withPermissions`, `withRoles`). Add helpers:

```go
// pkg/auditlog/correlation.go
func WithCorrelation(ctx context.Context, id string) context.Context
func CorrelationFrom(ctx context.Context) string
```

Multi-Insert handlers do:
```go
ctx = auditlog.WithCorrelation(ctx, newId(uid.CorrelationPrefix))
// nested call sites read from ctx if AuditLog.CorrelationID is empty
```

Same pattern works in the dashboard via TRPC context for multi-event
procedures (`rbac.createRole`, `vercel.setupProject`).

### Why no causation_id

`causation_id` only matters for chains across requests/processes (event
A causes event B 30 seconds later in a different process). Today every
multi-event audit flow fires synchronously inside one HTTP request /
TRPC call. `correlation_id` covers all of them. Add `causation_id` if
and when we ship async fan-out (e.g. webhook deliveries that themselves
emit audit events).

### Touched call sites

API multi-Insert flows that should set context-scoped correlation:
- `svc/api/routes/v2_keys_create_key/handler.go`
- `svc/api/routes/v2_keys_update_key/handler.go`
- `svc/api/routes/v2_keys_set_roles/handler.go`
- `svc/api/routes/v2_keys_set_permissions/handler.go`
- `svc/api/routes/v2_keys_add_permissions/handler.go`
- `svc/api/routes/v2_identities_update_identity/handler.go`
- `svc/api/routes/v2_identities_create_identity/handler.go`
- `svc/api/routes/v2_identities_delete_identity/handler.go`

Dashboard:
- `web/apps/dashboard/lib/trpc/routers/rbac/createRole.ts`
- `web/apps/dashboard/lib/trpc/routers/vercel.ts` (`setupProject`)
- Any other TRPC procedure that emits ≥2 audit events

Every other call site continues to work unchanged; correlation_id stays
empty for single-event flows.

## Operational checklist

- **CH Cloud backup retention.** Bump the cluster's automated backup
  retention to at least **30 days** (default 24h). Audit log archive
  cron + Parquet exports cover compliance, but CH Cloud backups are the
  fast path for "oh shit we deleted the wrong thing" recovery.
- **S3 bucket + IAM.** Provision the bucket the `AuditLogArchive` cron
  writes to. Grant the IAM identity (used by the `s3()` table function)
  PutObject on the prefix. Configure
  `worker.toml [audit_log_archive]`:
  ```toml
  [audit_log_archive]
  endpoint = "https://your-bucket.s3.us-east-1.amazonaws.com"
  prefix = "audit-logs"
  access_key = "<aws-access-key>"
  secret_key = "<aws-secret-key>"
  disabled = false
  ```
- **Outbox lag** is the main thing to monitor. Page if either of these
  exceeds a few minutes:
  ```sql
  SELECT MIN(created_at), COUNT(*)
  FROM clickhouse_outbox
  WHERE deleted_at IS NULL;
  ```
- **Unknown payload versions pile up silently.** Run periodically:
  ```sql
  SELECT version, COUNT(*) FROM clickhouse_outbox
  WHERE deleted_at IS NULL
  GROUP BY version;
  ```
  If anything other than `audit_log.v1` is non-zero, the drainer's
  `knownVersions` list is missing a handler.
- **Backfill progress.** The `AuditLogBackfillService` cursor is
  inspectable via Restate state (`last_pk`). When it stops advancing
  AND the heartbeat is firing, backfill is complete (caught up to the
  legacy tail).
- **Archive Parquet files.** They land at
  `{prefix}/cutoff={ISO8601}/run_{cutoff_millis}.parquet`. Inventory
  weekly to confirm new files are landing on schedule and no gaps in
  the cutoff timestamps.
- **Mutation pressure.** The archive cron fires one `ALTER TABLE DELETE`
  per pass with `mutations_sync = 2`. If you stack passes (more than
  daily) check `system.mutations` to make sure deletes complete before
  the next one starts.
