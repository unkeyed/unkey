-- name: FindAuditLogsForBackfill :many
-- FindAuditLogsForBackfill returns one cursor page of legacy audit_log
-- rows for the one-shot MySQL -> ClickHouse backfill VO. Ordered by pk so
-- the cursor advances monotonically; on a crash mid-page the VO replays
-- the same range from its persisted last_pk, and CH's
-- non_replicated_deduplication_window collapses the duplicate insert
-- block.
--
-- The pk <= cutoff bound makes the VO terminate. Rows written after the
-- backfill snapshotted the legacy tail are already shipped via the live
-- drainer, so the backfill skips them. Without this bound the cursor
-- would chase a moving target forever.
--
-- The primary key on `pk` makes this a forward range scan; no extra index
-- needed. We pull every column the auditlog.Event envelope needs in one
-- read so the VO does not have to re-query per row.
SELECT pk, id, workspace_id, bucket, event, time, display,
       remote_ip, user_agent, actor_type, actor_id, actor_name, actor_meta
FROM audit_log
WHERE pk > sqlc.arg(after_pk)
  AND pk <= sqlc.arg(cutoff_pk)
ORDER BY pk
LIMIT ?;

-- name: FindAuditLogMaxPK :one
-- FindAuditLogMaxPK returns MAX(pk) of the legacy audit_log table, used
-- by the backfill VO to snapshot the cutoff on first invocation. Returns
-- 0 when the table is empty (COALESCE + CAST keep the call from erroring
-- on NULL aggregation and force sqlc to infer the result as uint64
-- instead of interface{}).
SELECT CAST(COALESCE(MAX(pk), 0) AS UNSIGNED) AS max_pk
FROM audit_log;
