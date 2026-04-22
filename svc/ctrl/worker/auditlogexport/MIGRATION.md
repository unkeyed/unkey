# Audit logs MySQL -> ClickHouse migration

This PR ships the dual-write half of the audit log migration: every audit
event is written to **both** the legacy `audit_log` + `audit_log_target`
tables (so the dashboard keeps working) **and** the new `clickhouse_outbox`
table (so the ClickHouse pipeline starts populating). The dashboard reader
and cleanup of the legacy tables are deliberately out of scope so this PR
can be merged, soaked, and reverted cleanly without touching anything
customer-visible.

There is intentionally no historical backfill. The migration relies on
the dual-write window naturally filling ClickHouse with the data users
actually look at (anything inside their workspace's audit log retention
quota). Once that window has elapsed, the dashboard cuts over and any
older data still in MySQL is by definition past retention and unqueryable.

## What's in this PR

1. **CH `audit_logs_raw_v1`** redesigned: real `workspace_id` (no
   platform-encoding hack), Nested target arrays (one row per logical
   event, not per `event x target`), `LowCardinality` on enum-ish
   columns, partition by `inserted_at` (avoids fragmenting old
   partitions during late drains), per-row `expires_at` TTL, JSON columns
   for actor/event/target metadata with materialized text mirrors and
   tokenbf indexes for future search.
2. **MySQL `clickhouse_outbox`** table: `pk + version + workspace_id +
   event_id + payload (JSON) + created_at`. No indexes beyond pk; rows
   are consumed in pk order and deleted on success. The `version`
   column (e.g. `audit_log.v1`) lets future producers share the same
   table and lets the drainer skip payloads it can't decode.
3. **Writer** (`internal/services/auditlogs/insert.go`): dual-writes
   to the legacy tables (unchanged behavior) and the outbox in the
   same MySQL transaction. JSON-encodes the canonical `auditlog.Event`
   envelope and tags it with `auditlog.OutboxVersionV1`.
4. **Drainer** (`svc/ctrl/worker/auditlogexport/run_export_handler.go`):
   reads `clickhouse_outbox` `WHERE version IN (known)` `FOR UPDATE
   SKIP LOCKED`, decodes payloads, inserts into CH, deletes the outbox
   rows. Crash-safe via CH block dedup.

## Rollback story

Reverting the deploy:

- Writer stops touching the outbox. Legacy `audit_log` writes continue
  unchanged because the existing INSERT path was preserved.
- Outbox rows still in MySQL stop being drained. They sit in the table
  harmlessly until cleaned up manually or a future deploy puts the
  worker back.
- CH `audit_logs_raw_v1` keeps the data we already inserted; nothing
  reads it yet, so nothing breaks.
- Dashboard is unaffected throughout.

The whole stack (POC + this PR) reverts as a unit since the POC is not
yet merged, so `clickhouse_outbox` and the CH schema disappear together
with the writer and drainer code.

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
   `audit_log_target_insert.sql`, `audit_log_find_target_by_id.sql`.

This PR should land *after* the dashboard cutover has soaked and
*before* anyone forgets why these tables existed.

## Operational notes

- **Outbox lag is the only thing to monitor** while we're on this
  stack. `SELECT MIN(created_at), COUNT(*) FROM clickhouse_outbox`
  tells you both the depth of the backlog and how stale the head is.
  Page if either exceeds a few minutes.
- **Unknown payload versions pile up silently.** Run `SELECT version,
  COUNT(*) FROM clickhouse_outbox GROUP BY version` periodically; if
  any version other than `audit_log.v1` has rows accumulating, the
  drainer's `knownVersions` list is missing a handler.
- **CH retention** is per-row, set at insert from each workspace's
  quota. Free tier defaults to 30 days. Plan changes don't require
  touching CH; rows expire on their existing stamp.
- **CH deduplication** is bounded by
  `non_replicated_deduplication_window = 10000` rows per partition.
  At our current ~65k events/day this gives us several hours of "safe
  to retry the exact same insert block." A drainer outage longer than
  that could in principle cause a duplicate row in CH if the same
  block is re-inserted later, but `ReplacingMergeTree` collapses
  duplicates by `event_id` over time anyway. Worst case: a brief
  window where `count(*)` over-counts until the next merge.
