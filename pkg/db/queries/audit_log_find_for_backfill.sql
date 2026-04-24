-- name: FindAuditLogsForBackfill :many
-- FindAuditLogsForBackfill returns one cursor page of legacy audit_log
-- rows for the one-shot MySQL -> ClickHouse backfill VO. Ordered by pk so
-- the cursor advances monotonically; on a crash mid-page the VO replays
-- the same range from its persisted last_pk, and CH's
-- non_replicated_deduplication_window collapses the duplicate insert
-- block.
--
-- The primary key on `pk` makes this a forward range scan; no extra index
-- needed. We pull every column the auditlog.Event envelope needs in one
-- read so the VO does not have to re-query per row.
SELECT pk, id, workspace_id, bucket, event, time, display,
       remote_ip, user_agent, actor_type, actor_id, actor_name, actor_meta
FROM audit_log
WHERE pk > sqlc.arg(after_pk)
ORDER BY pk
LIMIT ?;
