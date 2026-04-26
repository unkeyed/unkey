-- name: FindUnexportedAuditLogs :many
-- FindUnexportedAuditLogs returns the next batch of audit log envelopes that
-- have not yet been written to the ClickHouse audit_logs_raw_v1 table.
-- Must be called inside a transaction. FOR UPDATE SKIP LOCKED locks the
-- batch's rows so a second cron run (if Restate VO serialization ever fails)
-- silently skips them rather than re-processing the same set. The lock is
-- released when the caller commits or rolls back. Ordered by pk so retries
-- see a deterministic row set, which lets CH's block-level deduplication
-- collapse re-inserts after a partial failure. The exported_pk_idx composite
-- index makes this scan cheap even when the bulk of the table is
-- exported = true.
SELECT pk, id, workspace_id, bucket, event, time, display,
       remote_ip, user_agent, actor_type, actor_id, actor_name, actor_meta
FROM audit_log
WHERE exported = false
ORDER BY pk
LIMIT ?
FOR UPDATE SKIP LOCKED;
